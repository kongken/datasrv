package internal

import (
	"context"
	"fmt"
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
	syncStore          dao.SyncStore
	feedStore          dao.FeedStore
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
	syncService = service.NewIssueSyncService(syncStore, conf.Conf.GitHub, conf.Conf.GitHubSync)
	feedSyncService = service.NewFeedSyncService(feedStore, conf.Conf.FeedSync, nil)
	adminGRPC = service.NewIssueSyncAdminGRPCServer(syncStore, syncService, conf.Conf)
	adminTokenValidator = service.NewRedisAdminTokenStore(conf.Conf)
	adminAuthGRPC = service.NewAdminAuthGRPCServer(conf.Conf, adminTokenValidator)
	queryGRPC = service.NewIssueQueryGRPCServer(syncStore)
	feedAdminGRPC = service.NewFeedSyncAdminGRPCServer(feedStore, feedSyncService, conf.Conf)
	feedQueryGRPC = service.NewFeedQueryGRPCServer(feedStore)
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

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = syncService.RunSync(schedulerCtx, "")
			case <-schedulerStopC:
				return
			case <-schedulerCtx.Done():
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

			go func() {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						_, _ = feedSyncService.RunSync(feedSchedulerCtx, "")
					case <-feedSchedulerStopC:
						return
					case <-feedSchedulerCtx.Done():
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
