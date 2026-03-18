package internal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
	"google.golang.org/grpc"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	appLogger          = slog.Default().With("component", "datasrv.app")
	syncStore          dao.SyncStore
	feedStore          dao.FeedStore
	commentStore       service.IssueCommentStore
	syncService        *service.IssueSyncService
	feedSyncService    *service.FeedSyncService
	adminGRPC          *service.IssueSyncAdminGRPCServer
	adminAuthGRPC      *service.AdminAuthGRPCServer
	queryGRPC          *service.IssueQueryGRPCServer
	feedAdminGRPC      *service.FeedSyncAdminGRPCServer
	feedQueryGRPC      *service.FeedQueryGRPCServer
	schedulerStopC     chan struct{}
	schedulerStop      context.CancelFunc
	feedSchedulerStopC chan struct{}
	feedSchedulerStop  context.CancelFunc
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:       conf.Conf,
		Namespace:    "auto",
		Service:      "datasrv",
		Router:       setupHTTPRouter,
		GRPCRegister: registerGRPC,
		InitFunc: []func() error{
			initSyncComponents,
			initGatewayHandler,
			startSyncScheduler,
		},
		TeardownFunc: []func() error{
			stopSyncScheduler,
			closeSyncStore,
		},
	})
	return app
}

func registerGRPC(server *grpc.Server) {
	if adminGRPC != nil {
		issuesv1.RegisterIssueSyncAdminServiceServer(server, adminGRPC)
	}
	if adminAuthGRPC != nil {
		issuesv1.RegisterAdminAuthServiceServer(server, adminAuthGRPC)
	}
	if queryGRPC != nil {
		issuesv1.RegisterIssueQueryServiceServer(server, queryGRPC)
	}
	if feedAdminGRPC != nil {
		feedsv1.RegisterFeedSyncAdminServiceServer(server, feedAdminGRPC)
	}
	if feedQueryGRPC != nil {
		feedsv1.RegisterFeedQueryServiceServer(server, feedQueryGRPC)
	}
}

func initSyncComponents() error {
	storage := conf.Conf.Storage
	if storage.Driver == "" {
		storage = conf.Conf.Database
	}
	driver := strings.ToLower(strings.TrimSpace(storage.Driver))
	if driver == "" {
		driver = "postgres"
	}

	var err error
	var combined interface{}
	commentStore, err = service.NewIssueCommentStore(conf.Conf.IssueCommentStorage)
	if err != nil {
		return fmt.Errorf("init issue comment store: %w", err)
	}
	switch driver {
	case "mongo", "mongodb":
		uri := storage.MongoURI
		if uri == "" {
			uri = storage.DSN
		}
		combined, err = dao.NewMongoSyncStore(uri, storage.MongoDB)
	case "postgres", "postgresql":
		dsn := storage.PostgresDSN
		if dsn == "" {
			dsn = storage.DSN
		}
		combined, err = dao.NewGormSyncStore(dsn)
	default:
		return fmt.Errorf("unsupported storage driver %q", driver)
	}
	if err != nil {
		return err
	}
	var ok bool
	syncStore, ok = combined.(dao.SyncStore)
	if !ok {
		return fmt.Errorf("store %T does not implement issue sync store", combined)
	}
	feedStore, ok = combined.(dao.FeedStore)
	if !ok {
		return fmt.Errorf("store %T does not implement feed store", combined)
	}

	conf.Conf.Storage.Driver = driver
	syncService = service.NewIssueSyncService(syncStore, conf.Conf.GitHub, conf.Conf.GitHubSync, commentStore)
	if err := syncService.SeedManagedRepos(context.Background(), conf.Conf.GitHubSync.Repos); err != nil {
		return fmt.Errorf("seed managed repos: %w", err)
	}
	managedRepos, err := syncService.ListManagedRepos(context.Background())
	if err != nil {
		return fmt.Errorf("list managed repos after seed: %w", err)
	}
	feedSyncService = service.NewFeedSyncService(feedStore, conf.Conf.FeedSync, nil)
	adminGRPC = service.NewIssueSyncAdminGRPCServer(syncStore, syncService, conf.Conf)
	adminTokenValidator = service.NewRedisAdminTokenStore(conf.Conf)
	adminAuthGRPC = service.NewAdminAuthGRPCServer(conf.Conf, adminTokenValidator)
	queryGRPC = service.NewIssueQueryGRPCServer(syncStore, commentStore)
	feedAdminGRPC = service.NewFeedSyncAdminGRPCServer(feedStore, feedSyncService, conf.Conf)
	feedQueryGRPC = service.NewFeedQueryGRPCServer(feedStore)
	appLogger.Info("sync components initialized",
		"storage_driver", driver,
		"issue_comment_storage_enabled", conf.Conf.IssueCommentStorage.Enabled,
		"issue_comment_storage_provider", conf.Conf.IssueCommentStorage.Provider,
		"issue_comment_storage_bucket", conf.Conf.IssueCommentStorage.Bucket,
		"issue_comment_storage_endpoint", conf.Conf.IssueCommentStorage.Endpoint,
		"issue_sync_enabled", conf.Conf.GitHubSync.Enabled,
		"managed_repo_count", len(managedRepos),
		"managed_repos", managedRepoNames(managedRepos),
		"feed_sync_enabled", conf.Conf.FeedSync.Enabled,
	)
	return nil
}

func startSyncScheduler() error {
	if syncService == nil {
		return fmt.Errorf("sync service is not initialized")
	}
	cfg := syncService.GetConfig()
	if !cfg.Enabled {
		return nil
	}

	if schedulerStopC != nil {
		return nil
	}
	schedulerStopC = make(chan struct{})
	var schedulerCtx context.Context
	schedulerCtx, schedulerStop = context.WithCancel(context.Background())

	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	managedRepos, err := syncService.ListManagedRepos(context.Background())
	if err != nil {
		return fmt.Errorf("list managed repos before scheduler start: %w", err)
	}
	appLogger.Info("issue sync scheduler started",
		"interval", interval.String(),
		"managed_repo_count", len(managedRepos),
		"managed_repos", managedRepoNames(managedRepos),
	)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				appLogger.Info("issue sync tick triggered")
				summary, err := syncService.RunSync(schedulerCtx, "")
				if err != nil {
					appLogger.Error("issue sync run failed", "error", err)
					continue
				}
				logIssueSyncSummary(summary)
			case <-schedulerStopC:
				appLogger.Info("issue sync scheduler stopped")
				return
			case <-schedulerCtx.Done():
				appLogger.Info("issue sync scheduler context done")
				return
			}
		}
	}()

	if feedSyncService != nil {
		feedCfg := feedSyncService.GetConfig()
		if feedCfg.Enabled && feedSchedulerStopC == nil {
			feedSchedulerStopC = make(chan struct{})
			var feedSchedulerCtx context.Context
			feedSchedulerCtx, feedSchedulerStop = context.WithCancel(context.Background())

			interval := time.Duration(feedCfg.IntervalSeconds) * time.Second
			if interval <= 0 {
				interval = 5 * time.Minute
			}
			appLogger.Info("feed sync scheduler started", "interval", interval.String())

			go func() {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						appLogger.Info("feed sync tick triggered")
						summary, err := feedSyncService.RunSync(feedSchedulerCtx, "")
						if err != nil {
							appLogger.Error("feed sync run failed", "error", err)
							continue
						}
						appLogger.Info("feed sync run finished",
							"started_at", summary.StartedAt,
							"finished_at", summary.FinishedAt,
							"result_count", len(summary.Results),
						)
					case <-feedSchedulerStopC:
						appLogger.Info("feed sync scheduler stopped")
						return
					case <-feedSchedulerCtx.Done():
						appLogger.Info("feed sync scheduler context done")
						return
					}
				}
			}()
		}
	}
	return nil
}

func stopSyncScheduler() error {
	if schedulerStop != nil {
		schedulerStop()
		schedulerStop = nil
	}
	if schedulerStopC != nil {
		close(schedulerStopC)
		schedulerStopC = nil
	}
	if feedSchedulerStop != nil {
		feedSchedulerStop()
		feedSchedulerStop = nil
	}
	if feedSchedulerStopC != nil {
		close(feedSchedulerStopC)
		feedSchedulerStopC = nil
	}
	return nil
}

func closeSyncStore() error {
	switch {
	case syncStore != nil:
		return syncStore.Close()
	case feedStore != nil:
		return feedStore.Close()
	}
	return nil
}

func managedRepoNames(repos []dao.ManagedRepo) []string {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.Repo)
	}
	return names
}

func logIssueSyncSummary(summary service.SyncRunSummary) {
	appLogger.Info("issue sync run finished",
		"started_at", summary.StartedAt,
		"finished_at", summary.FinishedAt,
		"result_count", len(summary.Results),
	)
	for _, result := range summary.Results {
		if result.Err != "" {
			appLogger.Error("issue sync repo failed",
				"repo", result.Repo,
				"fetched", result.Fetched,
				"persisted", result.Persisted,
				"error", result.Err,
			)
			continue
		}
		appLogger.Info("issue sync repo finished",
			"repo", result.Repo,
			"fetched", result.Fetched,
			"persisted", result.Persisted,
		)
	}
}
