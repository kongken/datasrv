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
	managed     map[string]dao.ManagedRepo
}

type fakeIssueCommentStore struct {
	saved   map[string][]dao.IssueComment
	loadErr error
	saveErr error
}

func newFakeSyncStore() *fakeSyncStore {
	return &fakeSyncStore{
		checkpoints: map[string]dao.Checkpoint{},
		issues:      map[string][]dao.SyncedIssue{},
		managed:     map[string]dao.ManagedRepo{},
	}
}

func newFakeIssueCommentStore() *fakeIssueCommentStore {
	return &fakeIssueCommentStore{saved: map[string][]dao.IssueComment{}}
}

func (f *fakeSyncStore) UpsertIssues(_ context.Context, repo string, issues []dao.SyncedIssue) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	existing := append([]dao.SyncedIssue(nil), f.issues[repo]...)
	for _, incoming := range issues {
		replaced := false
		for i, current := range existing {
			if current.IssueID == incoming.IssueID {
				if incoming.AISummary == "" {
					incoming.AISummary = current.AISummary
				}
				existing[i] = incoming
				replaced = true
				break
			}
		}
		if !replaced {
			existing = append(existing, incoming)
		}
	}
	f.issues[repo] = existing
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

func (f *fakeSyncStore) ListManagedRepos(_ context.Context) ([]dao.ManagedRepo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dao.ManagedRepo, 0, len(f.managed))
	for _, repo := range f.managed {
		out = append(out, repo)
	}
	return out, nil
}

func (f *fakeSyncStore) ReplaceManagedRepos(_ context.Context, repos []string) ([]dao.ManagedRepo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	next := make(map[string]dao.ManagedRepo, len(repos))
	now := time.Now().UTC()
	for _, repo := range repos {
		item, ok := f.managed[repo]
		if !ok {
			item = dao.ManagedRepo{Repo: repo, CreatedAt: now}
		}
		item.UpdatedAt = now
		next[repo] = item
	}
	f.managed = next

	out := make([]dao.ManagedRepo, 0, len(f.managed))
	for _, repo := range f.managed {
		out = append(out, repo)
	}
	return out, nil
}

func (f *fakeSyncStore) UpdateIssueAISummary(_ context.Context, repo string, issueID int64, number int32, summary string) (dao.SyncedIssue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := f.issues[repo]
	for i, row := range rows {
		if issueID > 0 && row.IssueID != issueID {
			continue
		}
		if issueID == 0 && number > 0 && row.Number != number {
			continue
		}
		row.AISummary = summary
		rows[i] = row
		f.issues[repo] = rows
		return row, nil
	}
	return dao.SyncedIssue{}, dao.ErrIssueNotFound
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
	mu               sync.Mutex
	calls            []listCall
	responses        []fakeGitHubResponse
	commentResponses map[int][]fakeGitHubCommentResponse
}

type fakeGitHubResponse struct {
	issues   []*github.Issue
	nextPage int
	err      error
}

type fakeGitHubCommentResponse struct {
	comments []*github.IssueComment
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

func (f *fakeGitHubIssueClient) ListComments(_ context.Context, owner, repo string, issueNumber int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.commentResponses == nil || len(f.commentResponses[issueNumber]) == 0 {
		return nil, &github.Response{}, nil
	}
	resp := f.commentResponses[issueNumber][0]
	f.commentResponses[issueNumber] = f.commentResponses[issueNumber][1:]
	if resp.err != nil {
		return nil, nil, resp.err
	}
	return resp.comments, &github.Response{NextPage: resp.nextPage}, nil
}

func (f *fakeIssueCommentStore) SaveComments(_ context.Context, repo string, issueID int64, issueNumber int32, comments []dao.IssueComment) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	key := fmt.Sprintf("%s/%d-%d.json", repo, issueID, issueNumber)
	f.saved[key] = append([]dao.IssueComment(nil), comments...)
	return nil
}

func (f *fakeIssueCommentStore) LoadComments(_ context.Context, repo string, issueID int64, issueNumber int32) ([]dao.IssueComment, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	key := fmt.Sprintf("%s/%d-%d.json", repo, issueID, issueNumber)
	return append([]dao.IssueComment(nil), f.saved[key]...), nil
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
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{Enabled: false, Repos: []string{"a/b"}}, nil)
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
	}, nil)
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
	}, nil)
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

func TestRunSyncPreservesExistingAISummary(t *testing.T) {
	store := newFakeSyncStore()
	existingUpdated := time.Now().UTC().Round(time.Second)
	_, _ = store.UpsertIssues(context.Background(), "owner/repo", []dao.SyncedIssue{{
		Repo:      "owner/repo",
		IssueID:   1,
		Number:    1,
		Title:     "existing",
		State:     "open",
		Author:    "alice",
		UpdatedAt: existingUpdated,
		AISummary: "keep me",
	}})

	client := &fakeGitHubIssueClient{
		responses: []fakeGitHubResponse{
			{issues: []*github.Issue{ghIssue(1, 1, existingUpdated.Add(time.Minute))}, nextPage: 0},
		},
	}
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{
		Enabled:               true,
		Repos:                 []string{"owner/repo"},
		RequestTimeoutSeconds: 5,
	}, nil)
	svc.client = client

	if _, err := svc.RunSync(context.Background(), ""); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	rows, err := store.ListIssues(context.Background(), dao.SyncIssueFilter{Repo: "owner/repo", Number: 1, Limit: 1})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if rows[0].AISummary != "keep me" {
		t.Fatalf("ai_summary = %q, want keep me", rows[0].AISummary)
	}
}

func TestRunSyncPersistsCommentsToBlobStore(t *testing.T) {
	store := newFakeSyncStore()
	commentStore := newFakeIssueCommentStore()
	updated := time.Now().UTC()
	client := &fakeGitHubIssueClient{
		responses: []fakeGitHubResponse{
			{issues: []*github.Issue{ghIssue(7, 77, updated)}, nextPage: 0},
		},
		commentResponses: map[int][]fakeGitHubCommentResponse{
			77: {{
				comments: []*github.IssueComment{{
					ID:      github.Ptr(int64(7001)),
					Body:    github.Ptr("first reply"),
					HTMLURL: github.Ptr("https://example.com/comment/7001"),
					User: &github.User{
						Login: github.Ptr("alice"),
					},
				}},
			}},
		},
	}
	svc := NewIssueSyncService(store, conf.GitHubConfig{}, conf.GitHubSyncConfig{
		Enabled:               true,
		Repos:                 []string{"owner/repo"},
		RequestTimeoutSeconds: 5,
	}, commentStore)
	svc.client = client

	if _, err := svc.RunSync(context.Background(), ""); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	rows, err := store.ListIssues(context.Background(), dao.SyncIssueFilter{Repo: "owner/repo", Number: 77, Limit: 1})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	key := "owner/repo/7-77.json"
	if len(commentStore.saved[key]) != 1 {
		t.Fatalf("saved comments len = %d, want 1", len(commentStore.saved[key]))
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
