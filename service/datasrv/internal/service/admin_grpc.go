package service

import (
	"context"
	"sync"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IssueSyncAdminGRPCServer implements issues.v1.IssueSyncAdminService.
type IssueSyncAdminGRPCServer struct {
	issuesv1.UnimplementedIssueSyncAdminServiceServer

	store   dao.SyncStore
	syncSvc *IssueSyncService
	cfg     *conf.Config

	statusMu sync.RWMutex
	lastRun  SyncRunSummary
}

func NewIssueSyncAdminGRPCServer(store dao.SyncStore, syncSvc *IssueSyncService, cfg *conf.Config) *IssueSyncAdminGRPCServer {
	return &IssueSyncAdminGRPCServer{store: store, syncSvc: syncSvc, cfg: cfg}
}

func (s *IssueSyncAdminGRPCServer) SyncIssues(ctx context.Context, req *issuesv1.SyncIssuesRequest) (*issuesv1.SyncIssuesResponse, error) {
	summary, err := s.syncSvc.RunSync(ctx, req.GetRepo())
	if err != nil {
		return nil, err
	}

	s.statusMu.Lock()
	s.lastRun = summary
	s.statusMu.Unlock()

	results := make([]*issuesv1.SyncRepoResult, 0, len(summary.Results))
	for _, it := range summary.Results {
		results = append(results, &issuesv1.SyncRepoResult{
			Repo:      it.Repo,
			Fetched:   it.Fetched,
			Persisted: it.Persisted,
			Error:     it.Err,
		})
	}

	return &issuesv1.SyncIssuesResponse{
		StartedAt:  timestamppb.New(summary.StartedAt),
		FinishedAt: timestamppb.New(summary.FinishedAt),
		Results:    results,
	}, nil
}

func (s *IssueSyncAdminGRPCServer) GetSyncConfig(context.Context, *emptypb.Empty) (*issuesv1.GetSyncConfigResponse, error) {
	cfg := s.syncSvc.GetConfig()
	return &issuesv1.GetSyncConfigResponse{
		Enabled:               cfg.Enabled,
		Repos:                 cfg.Repos,
		IntervalSeconds:       int32(cfg.IntervalSeconds),
		PageSize:              int32(cfg.PageSize),
		MaxPagesPerRun:        int32(cfg.MaxPagesPerRun),
		RequestTimeoutSeconds: int32(cfg.RequestTimeoutSeconds),
		StorageDriver:         s.cfg.Storage.Driver,
		GithubTokenConfigured: s.cfg.GitHub.Token != "",
	}, nil
}

func (s *IssueSyncAdminGRPCServer) UpdateSyncConfig(_ context.Context, req *issuesv1.UpdateSyncConfigRequest) (*issuesv1.GetSyncConfigResponse, error) {
	updated := s.syncSvc.UpdateConfig(conf.GitHubSyncConfig{
		Enabled:               req.GetEnabled(),
		Repos:                 req.GetRepos(),
		IntervalSeconds:       int(req.GetIntervalSeconds()),
		PageSize:              int(req.GetPageSize()),
		MaxPagesPerRun:        int(req.GetMaxPagesPerRun()),
		RequestTimeoutSeconds: int(req.GetRequestTimeoutSeconds()),
	})

	s.cfg.GitHubSync = updated
	return &issuesv1.GetSyncConfigResponse{
		Enabled:               updated.Enabled,
		Repos:                 updated.Repos,
		IntervalSeconds:       int32(updated.IntervalSeconds),
		PageSize:              int32(updated.PageSize),
		MaxPagesPerRun:        int32(updated.MaxPagesPerRun),
		RequestTimeoutSeconds: int32(updated.RequestTimeoutSeconds),
		StorageDriver:         s.cfg.Storage.Driver,
		GithubTokenConfigured: s.cfg.GitHub.Token != "",
	}, nil
}

func (s *IssueSyncAdminGRPCServer) GetSyncStatus(ctx context.Context, _ *emptypb.Empty) (*issuesv1.GetSyncStatusResponse, error) {
	s.statusMu.RLock()
	lastRun := s.lastRun
	s.statusMu.RUnlock()

	checkpoints, err := s.store.ListCheckpoints(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*issuesv1.SyncRepoResult, 0, len(lastRun.Results))
	for _, it := range lastRun.Results {
		results = append(results, &issuesv1.SyncRepoResult{
			Repo:      it.Repo,
			Fetched:   it.Fetched,
			Persisted: it.Persisted,
			Error:     it.Err,
		})
	}

	cpItems := make([]*issuesv1.SyncCheckpoint, 0, len(checkpoints))
	for _, cp := range checkpoints {
		cpItems = append(cpItems, &issuesv1.SyncCheckpoint{
			Repo:               cp.Repo,
			LastSyncedAt:       timestamppb.New(cp.LastSyncedAt),
			LastIssueUpdatedAt: timestamppb.New(cp.LastIssueUpdatedAt),
			LastRunStatus:      cp.LastRunStatus,
			LastError:          cp.LastError,
		})
	}

	resp := &issuesv1.GetSyncStatusResponse{
		Running:     s.syncSvc.IsRunning(),
		LastResults: results,
		Checkpoints: cpItems,
	}
	if !lastRun.StartedAt.IsZero() {
		resp.LastStartedAt = timestamppb.New(lastRun.StartedAt)
	}
	if !lastRun.FinishedAt.IsZero() {
		resp.LastFinishedAt = timestamppb.New(lastRun.FinishedAt)
	}
	return resp, nil
}
