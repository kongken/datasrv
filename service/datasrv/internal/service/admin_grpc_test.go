package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v82/github"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestIssueSyncAdminGRPCServer_GetAndUpdateSyncConfig(t *testing.T) {
	store := newFakeSyncStore()
	cfg := &conf.Config{
		Storage: conf.StorageConfig{Driver: "postgres"},
		GitHub:  conf.GitHubConfig{Token: "secret-token"},
	}
	syncSvc := NewIssueSyncService(store, cfg.GitHub, conf.GitHubSyncConfig{Enabled: true, Repos: []string{"a/b"}})
	srv := NewIssueSyncAdminGRPCServer(store, syncSvc, cfg)

	resp, err := srv.GetSyncConfig(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetSyncConfig() error = %v", err)
	}
	if !resp.GithubTokenConfigured {
		t.Fatalf("GithubTokenConfigured = false, want true")
	}
	if resp.StorageDriver != "postgres" {
		t.Fatalf("StorageDriver = %q, want postgres", resp.StorageDriver)
	}

	updated, err := srv.UpdateSyncConfig(context.Background(), &issuesv1.UpdateSyncConfigRequest{
		Enabled:               true,
		Repos:                 []string{"octo/repo"},
		IntervalSeconds:       60,
		PageSize:              50,
		MaxPagesPerRun:        5,
		RequestTimeoutSeconds: 8,
	})
	if err != nil {
		t.Fatalf("UpdateSyncConfig() error = %v", err)
	}
	if len(updated.Repos) != 1 || updated.Repos[0] != "octo/repo" {
		t.Fatalf("updated repos = %#v, want [octo/repo]", updated.Repos)
	}
}

func TestIssueSyncAdminGRPCServer_SyncIssuesAndStatus(t *testing.T) {
	store := newFakeSyncStore()
	cfg := &conf.Config{
		Storage: conf.StorageConfig{Driver: "mongo"},
		GitHub:  conf.GitHubConfig{},
	}
	syncSvc := NewIssueSyncService(store, cfg.GitHub, conf.GitHubSyncConfig{
		Enabled:               true,
		Repos:                 []string{"owner/repo"},
		RequestTimeoutSeconds: 5,
	})
	syncSvc.client = &fakeGitHubIssueClient{
		responses: []fakeGitHubResponse{{issues: []*github.Issue{ghIssue(10, 10, time.Now().UTC())}, nextPage: 0}},
	}

	srv := NewIssueSyncAdminGRPCServer(store, syncSvc, cfg)
	_, err := srv.SyncIssues(context.Background(), &issuesv1.SyncIssuesRequest{})
	if err != nil {
		t.Fatalf("SyncIssues() error = %v", err)
	}

	status, err := srv.GetSyncStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetSyncStatus() error = %v", err)
	}
	if len(status.LastResults) != 1 {
		t.Fatalf("LastResults len = %d, want 1", len(status.LastResults))
	}
	if status.LastResults[0].Repo != "owner/repo" {
		t.Fatalf("LastResults[0].Repo = %q, want owner/repo", status.LastResults[0].Repo)
	}
	if len(status.Checkpoints) != 1 {
		t.Fatalf("Checkpoints len = %d, want 1", len(status.Checkpoints))
	}
}

func TestIssueSyncAdminGRPCServer_GetSyncStatusError(t *testing.T) {
	store := &errorSyncStore{}
	cfg := &conf.Config{}
	syncSvc := NewIssueSyncService(newFakeSyncStore(), conf.GitHubConfig{}, conf.GitHubSyncConfig{})
	srv := NewIssueSyncAdminGRPCServer(store, syncSvc, cfg)

	if _, err := srv.GetSyncStatus(context.Background(), &emptypb.Empty{}); err == nil {
		t.Fatalf("GetSyncStatus() should fail when store returns error")
	}
}

type errorSyncStore struct{}

func (e *errorSyncStore) UpsertIssues(context.Context, string, []dao.SyncedIssue) (int, error) {
	return 0, nil
}
func (e *errorSyncStore) ListIssues(context.Context, dao.SyncIssueFilter) ([]dao.SyncedIssue, error) {
	return nil, nil
}
func (e *errorSyncStore) GetRepoCheckpoint(context.Context, string) (dao.Checkpoint, error) {
	return dao.Checkpoint{}, nil
}
func (e *errorSyncStore) SaveRepoCheckpoint(context.Context, dao.Checkpoint) error { return nil }
func (e *errorSyncStore) ListCheckpoints(context.Context) ([]dao.Checkpoint, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorSyncStore) Close() error { return nil }
