package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
	"google.golang.org/grpc"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

var (
	syncStore      dao.SyncStore
	syncService    *service.IssueSyncService
	adminGRPC      *service.IssueSyncAdminGRPCServer
	queryGRPC      *service.IssueQueryGRPCServer
	schedulerStopC chan struct{}
	schedulerStop  context.CancelFunc
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:  conf.Conf,
		Service: "datasrv",
		// Router:  http.Router,
		GRPCRegister: registerGRPC,
		InitFunc: []func() error{
			initSyncComponents,
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
	if queryGRPC != nil {
		issuesv1.RegisterIssueQueryServiceServer(server, queryGRPC)
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
	switch driver {
	case "mongo", "mongodb":
		uri := storage.MongoURI
		if uri == "" {
			uri = storage.DSN
		}
		syncStore, err = dao.NewMongoSyncStore(uri, storage.MongoDB)
	case "postgres", "postgresql":
		dsn := storage.PostgresDSN
		if dsn == "" {
			dsn = storage.DSN
		}
		syncStore, err = dao.NewGormSyncStore(dsn)
	default:
		return fmt.Errorf("unsupported storage driver %q", driver)
	}
	if err != nil {
		return err
	}

	conf.Conf.Storage.Driver = driver
	syncService = service.NewIssueSyncService(syncStore, conf.Conf.GitHub, conf.Conf.GitHubSync)
	adminGRPC = service.NewIssueSyncAdminGRPCServer(syncStore, syncService, conf.Conf)
	queryGRPC = service.NewIssueQueryGRPCServer(syncStore)
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
	return nil
}

func closeSyncStore() error {
	if syncStore != nil {
		return syncStore.Close()
	}
	return nil
}
