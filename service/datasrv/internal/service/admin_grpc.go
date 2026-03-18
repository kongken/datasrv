package service

import (
	"context"
	"sync"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	repos, err := s.syncSvc.ListManagedRepos(context.Background())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list managed repos: %v", err)
	}
	return &issuesv1.GetSyncConfigResponse{
		Enabled:               cfg.Enabled,
		Repos:                 managedRepoNames(repos),
		IntervalSeconds:       int32(cfg.IntervalSeconds),
		PageSize:              int32(cfg.PageSize),
		MaxPagesPerRun:        int32(cfg.MaxPagesPerRun),
		RequestTimeoutSeconds: int32(cfg.RequestTimeoutSeconds),
		StorageDriver:         s.cfg.Storage.Driver,
		GithubTokenConfigured: s.cfg.GitHub.Token != "",
	}, nil
}

func (s *IssueSyncAdminGRPCServer) UpdateSyncConfig(ctx context.Context, req *issuesv1.UpdateSyncConfigRequest) (*issuesv1.GetSyncConfigResponse, error) {
	updated := s.syncSvc.UpdateConfig(conf.GitHubSyncConfig{
		Enabled:               req.GetEnabled(),
		Repos:                 s.syncSvc.GetConfig().Repos,
		IntervalSeconds:       int(req.GetIntervalSeconds()),
		PageSize:              int(req.GetPageSize()),
		MaxPagesPerRun:        int(req.GetMaxPagesPerRun()),
		RequestTimeoutSeconds: int(req.GetRequestTimeoutSeconds()),
	})

	var managedRepos []dao.ManagedRepo
	var err error
	if req.Repos != nil {
		managedRepos, err = s.syncSvc.ReplaceManagedRepos(ctx, req.GetRepos())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "replace managed repos: %v", err)
		}
	} else {
		managedRepos, err = s.syncSvc.ListManagedRepos(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list managed repos: %v", err)
		}
	}

	s.cfg.GitHubSync = updated
	return &issuesv1.GetSyncConfigResponse{
		Enabled:               updated.Enabled,
		Repos:                 managedRepoNames(managedRepos),
		IntervalSeconds:       int32(updated.IntervalSeconds),
		PageSize:              int32(updated.PageSize),
		MaxPagesPerRun:        int32(updated.MaxPagesPerRun),
		RequestTimeoutSeconds: int32(updated.RequestTimeoutSeconds),
		StorageDriver:         s.cfg.Storage.Driver,
		GithubTokenConfigured: s.cfg.GitHub.Token != "",
	}, nil
}

func (s *IssueSyncAdminGRPCServer) ListManagedSyncRepos(ctx context.Context, _ *emptypb.Empty) (*issuesv1.ListManagedSyncReposResponse, error) {
	repos, err := s.syncSvc.ListManagedRepos(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list managed repos: %v", err)
	}
	return &issuesv1.ListManagedSyncReposResponse{Repos: toProtoManagedRepos(repos)}, nil
}

func (s *IssueSyncAdminGRPCServer) ReplaceManagedSyncRepos(ctx context.Context, req *issuesv1.ReplaceManagedSyncReposRequest) (*issuesv1.ListManagedSyncReposResponse, error) {
	repos, err := s.syncSvc.ReplaceManagedRepos(ctx, req.GetRepos())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "replace managed repos: %v", err)
	}
	return &issuesv1.ListManagedSyncReposResponse{Repos: toProtoManagedRepos(repos)}, nil
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

func (s *IssueSyncAdminGRPCServer) UpdateIssueAISummary(ctx context.Context, req *issuesv1.UpdateIssueAISummaryRequest) (*issuesv1.GetIssueResponse, error) {
	if req.GetRepo() == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	var issueID int64
	var number int32
	switch {
	case req.GetIssueId() > 0:
		issueID = req.GetIssueId()
	case req.GetNumber() > 0:
		number = req.GetNumber()
	default:
		return nil, status.Error(codes.InvalidArgument, "either issue_id or number is required")
	}

	updated, err := s.store.UpdateIssueAISummary(ctx, req.GetRepo(), issueID, number, req.GetAiSummary())
	if err != nil {
		if err == dao.ErrIssueNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "update issue ai summary: %v", err)
	}
	return &issuesv1.GetIssueResponse{Issue: toProtoIssue(updated)}, nil
}

func toProtoManagedRepos(repos []dao.ManagedRepo) []*issuesv1.ManagedSyncRepo {
	out := make([]*issuesv1.ManagedSyncRepo, 0, len(repos))
	for _, repo := range repos {
		item := &issuesv1.ManagedSyncRepo{Repo: repo.Repo}
		if !repo.CreatedAt.IsZero() {
			item.CreatedAt = timestamppb.New(repo.CreatedAt)
		}
		if !repo.UpdatedAt.IsZero() {
			item.UpdatedAt = timestamppb.New(repo.UpdatedAt)
		}
		out = append(out, item)
	}
	return out
}

func managedRepoNames(repos []dao.ManagedRepo) []string {
	out := make([]string, 0, len(repos))
	for _, repo := range repos {
		if repo.Repo == "" {
			continue
		}
		out = append(out, repo.Repo)
	}
	return out
}
