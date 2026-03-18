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
	_, _ = store.ReplaceManagedRepos(context.Background(), []string{"a/b"})
	cfg := &conf.Config{
		Storage: conf.StorageConfig{Driver: "postgres"},
		GitHub:  conf.GitHubConfig{Token: "secret-token"},
	}
	syncSvc := NewIssueSyncService(store, cfg.GitHub, conf.GitHubSyncConfig{Enabled: true, Repos: []string{"a/b"}}, nil)
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

func TestIssueSyncAdminGRPCServer_ListAndReplaceManagedSyncRepos(t *testing.T) {
	store := newFakeSyncStore()
	_, _ = store.ReplaceManagedRepos(context.Background(), []string{"o/a"})
	srv := NewIssueSyncAdminGRPCServer(store, NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{}, nil), &conf.Config{})

	listed, err := srv.ListManagedSyncRepos(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListManagedSyncRepos() error = %v", err)
	}
	if len(listed.Repos) != 1 || listed.Repos[0].GetRepo() != "o/a" {
		t.Fatalf("listed repos = %#v, want [o/a]", listed.Repos)
	}

	replaced, err := srv.ReplaceManagedSyncRepos(context.Background(), &issuesv1.ReplaceManagedSyncReposRequest{
		Repos: []string{"o/b", "o/c"},
	})
	if err != nil {
		t.Fatalf("ReplaceManagedSyncRepos() error = %v", err)
	}
	if len(replaced.Repos) != 2 {
		t.Fatalf("replaced repos len = %d, want 2", len(replaced.Repos))
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
	}, nil)
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
	syncSvc := NewIssueSyncService(newFakeSyncStore(), conf.GitHubConfig{}, conf.GitHubSyncConfig{}, nil)
	srv := NewIssueSyncAdminGRPCServer(store, syncSvc, cfg)

	if _, err := srv.GetSyncStatus(context.Background(), &emptypb.Empty{}); err == nil {
		t.Fatalf("GetSyncStatus() should fail when store returns error")
	}
}

func TestIssueSyncAdminGRPCServer_UpdateIssueAISummaryByNumber(t *testing.T) {
	store := newFakeSyncStore()
	now := time.Now().UTC()
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{
		{Repo: "o/r", IssueID: 10, Number: 100, Title: "hello", State: "open", Author: "alice", UpdatedAt: now},
	})

	srv := NewIssueSyncAdminGRPCServer(store, NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{}, nil), &conf.Config{})
	resp, err := srv.UpdateIssueAISummary(context.Background(), &issuesv1.UpdateIssueAISummaryRequest{
		Repo:      "o/r",
		Selector:  &issuesv1.UpdateIssueAISummaryRequest_Number{Number: 100},
		AiSummary: "summary text",
	})
	if err != nil {
		t.Fatalf("UpdateIssueAISummary() error = %v", err)
	}
	if resp.GetIssue().GetAiSummary() != "summary text" {
		t.Fatalf("ai_summary = %q, want summary text", resp.GetIssue().GetAiSummary())
	}
}

type errorSyncStore struct{}

func (e *errorSyncStore) UpsertIssues(context.Context, string, []dao.SyncedIssue) (int, error) {
	return 0, nil
}
func (e *errorSyncStore) ListIssues(context.Context, dao.SyncIssueFilter) ([]dao.SyncedIssue, error) {
	return nil, nil
}
func (e *errorSyncStore) UpdateIssueAISummary(context.Context, string, int64, int32, string) (dao.SyncedIssue, error) {
	return dao.SyncedIssue{}, context.DeadlineExceeded
}
func (e *errorSyncStore) ListManagedRepos(context.Context) ([]dao.ManagedRepo, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorSyncStore) ReplaceManagedRepos(context.Context, []string) ([]dao.ManagedRepo, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorSyncStore) GetRepoCheckpoint(context.Context, string) (dao.Checkpoint, error) {
	return dao.Checkpoint{}, nil
}
func (e *errorSyncStore) SaveRepoCheckpoint(context.Context, dao.Checkpoint) error { return nil }
func (e *errorSyncStore) ListCheckpoints(context.Context) ([]dao.Checkpoint, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorSyncStore) Close() error { return nil }
