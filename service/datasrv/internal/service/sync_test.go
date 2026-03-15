package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v82/github"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type fakeSyncStore struct {
	mu          sync.Mutex
	checkpoints map[string]dao.Checkpoint
	issues      map[string][]dao.SyncedIssue
}

func newFakeSyncStore() *fakeSyncStore {
	return &fakeSyncStore{
		checkpoints: map[string]dao.Checkpoint{},
		issues:      map[string][]dao.SyncedIssue{},
	}
}

func (f *fakeSyncStore) UpsertIssues(_ context.Context, repo string, issues []dao.SyncedIssue) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issues[repo] = append(f.issues[repo], issues...)
	return len(issues), nil
}

func (f *fakeSyncStore) ListIssues(_ context.Context, filter dao.SyncIssueFilter) ([]dao.SyncedIssue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var source []dao.SyncedIssue
	if filter.Repo == "" {
		for _, items := range f.issues {
			source = append(source, items...)
		}
	} else {
		source = append(source, f.issues[filter.Repo]...)
	}

	filtered := make([]dao.SyncedIssue, 0, len(source))
	for _, it := range source {
		if filter.State != "" && filter.State != "all" && it.State != filter.State {
			continue
		}
		if filter.IssueID > 0 && it.IssueID != filter.IssueID {
			continue
		}
		if filter.Number > 0 && it.Number != filter.Number {
			continue
		}
		filtered = append(filtered, it)
	}

	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start > len(filtered) {
		return []dao.SyncedIssue{}, nil
	}
	end := len(filtered)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return append([]dao.SyncedIssue(nil), filtered[start:end]...), nil
}

func (f *fakeSyncStore) GetRepoCheckpoint(_ context.Context, repo string) (dao.Checkpoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cp, ok := f.checkpoints[repo]; ok {
		return cp, nil
	}
	return dao.Checkpoint{Repo: repo}, nil
}

func (f *fakeSyncStore) SaveRepoCheckpoint(_ context.Context, checkpoint dao.Checkpoint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checkpoints[checkpoint.Repo] = checkpoint
	return nil
}

func (f *fakeSyncStore) ListCheckpoints(_ context.Context) ([]dao.Checkpoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dao.Checkpoint, 0, len(f.checkpoints))
	for _, cp := range f.checkpoints {
		out = append(out, cp)
	}
	return out, nil
}

func (f *fakeSyncStore) Close() error { return nil }

type listCall struct {
	owner string
	repo  string
	opts  github.IssueListByRepoOptions
}

type fakeGitHubIssueClient struct {
	mu        sync.Mutex
	calls     []listCall
	responses []fakeGitHubResponse
}

type fakeGitHubResponse struct {
	issues   []*github.Issue
	nextPage int
	err      error
}

func (f *fakeGitHubIssueClient) ListByRepo(_ context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, listCall{owner: owner, repo: repo, opts: *opts})
	if len(f.responses) == 0 {
		return nil, &github.Response{}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	if resp.err != nil {
		return nil, nil, resp.err
	}
	return resp.issues, &github.Response{NextPage: resp.nextPage}, nil
}

func ghIssue(id int64, number int, updated time.Time) *github.Issue {
	user := &github.User{Login: github.Ptr("alice")}
	title := fmt.Sprintf("issue-%d", number)
	state := "open"
	comments := 1
	htmlURL := fmt.Sprintf("https://example.com/%d", number)
	created := github.Timestamp{Time: updated.Add(-time.Hour)}
	updatedTS := github.Timestamp{Time: updated}
	return &github.Issue{
		ID:        github.Ptr(id),
		Number:    github.Ptr(number),
		Title:     github.Ptr(title),
		State:     github.Ptr(state),
		User:      user,
		Comments:  github.Ptr(comments),
		HTMLURL:   github.Ptr(htmlURL),
		CreatedAt: &created,
		UpdatedAt: &updatedTS,
	}
}

func TestNormalizeSyncConfigDefaults(t *testing.T) {
	cfg := normalizeSyncConfig(conf.GitHubSyncConfig{})
	if cfg.IntervalSeconds != 300 || cfg.PageSize != 100 || cfg.MaxPagesPerRun != 10 || cfg.RequestTimeoutSeconds != 15 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestRunSyncDisabled(t *testing.T) {
	store := newFakeSyncStore()
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{Enabled: false, Repos: []string{"a/b"}})
	summary, err := svc.RunSync(context.Background(), "")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if len(summary.Results) != 0 {
		t.Fatalf("results len = %d, want 0", len(summary.Results))
	}
}

func TestRunSyncSuccessWithPaginationAndCheckpoint(t *testing.T) {
	store := newFakeSyncStore()
	updated1 := time.Now().UTC().Round(time.Second)
	updated2 := updated1.Add(2 * time.Minute)
	client := &fakeGitHubIssueClient{
		responses: []fakeGitHubResponse{
			{issues: []*github.Issue{ghIssue(1, 1, updated1)}, nextPage: 2},
			{issues: []*github.Issue{ghIssue(2, 2, updated2)}, nextPage: 0},
		},
	}
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{
		Enabled:               true,
		Repos:                 []string{"owner/repo"},
		PageSize:              100,
		MaxPagesPerRun:        10,
		RequestTimeoutSeconds: 5,
	})
	svc.client = client

	summary, err := svc.RunSync(context.Background(), "")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(summary.Results))
	}
	if got := summary.Results[0].Persisted; got != 2 {
		t.Fatalf("persisted = %d, want 2", got)
	}

	cp, err := store.GetRepoCheckpoint(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("GetRepoCheckpoint() error = %v", err)
	}
	if cp.LastRunStatus != "success" {
		t.Fatalf("last status = %q, want success", cp.LastRunStatus)
	}
	if !cp.LastIssueUpdatedAt.Equal(updated2) {
		t.Fatalf("checkpoint updated_at = %v, want %v", cp.LastIssueUpdatedAt, updated2)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(client.calls))
	}
}

func TestRunSyncRetryThenSuccess(t *testing.T) {
	store := newFakeSyncStore()
	client := &fakeGitHubIssueClient{
		responses: []fakeGitHubResponse{
			{err: fmt.Errorf("temporary-1")},
			{err: fmt.Errorf("temporary-2")},
			{issues: []*github.Issue{ghIssue(3, 3, time.Now().UTC())}, nextPage: 0},
		},
	}
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{
		Enabled:               true,
		Repos:                 []string{"owner/repo"},
		RequestTimeoutSeconds: 5,
	})
	svc.client = client

	summary, err := svc.RunSync(context.Background(), "")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if summary.Results[0].Err != "" {
		t.Fatalf("result err = %q, want empty", summary.Results[0].Err)
	}
	if len(client.calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(client.calls))
	}
}

func TestSplitRepo(t *testing.T) {
	owner, repo, err := splitRepo("octo/hello")
	if err != nil || owner != "octo" || repo != "hello" {
		t.Fatalf("splitRepo valid failed: owner=%q repo=%q err=%v", owner, repo, err)
	}
	if _, _, err := splitRepo("invalid"); err == nil {
		t.Fatalf("splitRepo invalid should fail")
	}
}
