package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v82/github"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"golang.org/x/oauth2"
)

var issueSyncLogger = slog.Default().With("component", "datasrv.issue_sync")

// GitHubIssueClient is the external client contract used by sync flow.
type GitHubIssueClient interface {
	ListByRepo(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	ListComments(ctx context.Context, owner, repo string, issueNumber int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)
}

type defaultGitHubIssueClient struct {
	inner *github.Client
}

func (d *defaultGitHubIssueClient) ListByRepo(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	return d.inner.Issues.ListByRepo(ctx, owner, repo, opts)
}

func (d *defaultGitHubIssueClient) ListComments(ctx context.Context, owner, repo string, issueNumber int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error) {
	return d.inner.Issues.ListComments(ctx, owner, repo, issueNumber, opts)
}

type SyncRepoResult struct {
	Repo      string
	Fetched   int32
	Persisted int32
	Err       string
}

type SyncRunSummary struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []SyncRepoResult
}

// IssueSyncService orchestrates periodic/on-demand sync from GitHub to a configured datastore.
type IssueSyncService struct {
	mu      sync.Mutex
	running bool

	cfgMu sync.RWMutex
	cfg   conf.GitHubSyncConfig

	store        dao.SyncStore
	client       GitHubIssueClient
	commentStore IssueCommentStore
}

func NewIssueSyncService(store dao.SyncStore, ghCfg conf.GitHubConfig, syncCfg conf.GitHubSyncConfig, commentStore IssueCommentStore) *IssueSyncService {
	httpClient := http.DefaultClient
	if ghCfg.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghCfg.Token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}
	ghClient := github.NewClient(httpClient)
	if ghCfg.BaseURL != "" {
		if enterpriseClient, err := github.NewClient(httpClient).WithEnterpriseURLs(ghCfg.BaseURL, ghCfg.BaseURL); err == nil {
			ghClient = enterpriseClient
		}
	}

	normalized := normalizeSyncConfig(syncCfg)
	return &IssueSyncService{
		cfg:          normalized,
		store:        store,
		client:       &defaultGitHubIssueClient{inner: ghClient},
		commentStore: commentStore,
	}
}

func normalizeSyncConfig(in conf.GitHubSyncConfig) conf.GitHubSyncConfig {
	if in.IntervalSeconds <= 0 {
		in.IntervalSeconds = 300
	}
	if in.PageSize <= 0 || in.PageSize > 100 {
		in.PageSize = 100
	}
	if in.MaxPagesPerRun <= 0 {
		in.MaxPagesPerRun = 10
	}
	if in.RequestTimeoutSeconds <= 0 {
		in.RequestTimeoutSeconds = 15
	}
	return in
}

func requestTimeout(ctx context.Context, cfg conf.GitHubSyncConfig) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(cfg.RequestTimeoutSeconds)*time.Second)
}

func normalizeManagedRepos(repos []string) []string {
	seen := make(map[string]struct{}, len(repos))
	out := make([]string, 0, len(repos))
	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		if _, ok := seen[repo]; ok {
			continue
		}
		seen[repo] = struct{}{}
		out = append(out, repo)
	}
	sort.Strings(out)
	return out
}

func (s *IssueSyncService) GetConfig() conf.GitHubSyncConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *IssueSyncService) UpdateConfig(cfg conf.GitHubSyncConfig) conf.GitHubSyncConfig {
	normalized := normalizeSyncConfig(cfg)
	s.cfgMu.Lock()
	s.cfg = normalized
	s.cfgMu.Unlock()
	return normalized
}

func (s *IssueSyncService) ListManagedRepos(ctx context.Context) ([]dao.ManagedRepo, error) {
	repos, err := s.store.ListManagedRepos(ctx)
	if err != nil {
		return nil, err
	}
	return repos, nil
}

func (s *IssueSyncService) ReplaceManagedRepos(ctx context.Context, repos []string) ([]dao.ManagedRepo, error) {
	return s.store.ReplaceManagedRepos(ctx, normalizeManagedRepos(repos))
}

func (s *IssueSyncService) SeedManagedRepos(ctx context.Context, repos []string) error {
	current, err := s.store.ListManagedRepos(ctx)
	if err != nil {
		return err
	}
	if len(current) > 0 {
		return nil
	}
	normalized := normalizeManagedRepos(repos)
	if len(normalized) == 0 {
		return nil
	}
	_, err = s.store.ReplaceManagedRepos(ctx, normalized)
	return err
}

func (s *IssueSyncService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *IssueSyncService) RunSync(ctx context.Context, onlyRepo string) (SyncRunSummary, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return SyncRunSummary{}, fmt.Errorf("sync is already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	cfg := s.GetConfig()
	started := time.Now()
	summary := SyncRunSummary{StartedAt: started}
	if !cfg.Enabled {
		summary.FinishedAt = time.Now()
		return summary, nil
	}

	repos, err := s.loadReposForRun(ctx, cfg, onlyRepo)
	if err != nil {
		return SyncRunSummary{}, err
	}
	if onlyRepo != "" {
		repos = []string{onlyRepo}
	}

	for _, repo := range repos {
		result := s.syncOneRepo(ctx, cfg, strings.TrimSpace(repo))
		summary.Results = append(summary.Results, result)
	}
	summary.FinishedAt = time.Now()
	return summary, nil
}

func (s *IssueSyncService) loadReposForRun(ctx context.Context, cfg conf.GitHubSyncConfig, onlyRepo string) ([]string, error) {
	if onlyRepo != "" {
		return []string{strings.TrimSpace(onlyRepo)}, nil
	}

	managed, err := s.store.ListManagedRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list managed repos: %w", err)
	}

	if len(managed) > 0 {
		repos := make([]string, 0, len(managed))
		for _, repo := range managed {
			if repo.Repo == "" {
				continue
			}
			repos = append(repos, repo.Repo)
		}
		return repos, nil
	}

	return normalizeManagedRepos(cfg.Repos), nil
}

func (s *IssueSyncService) syncOneRepo(ctx context.Context, cfg conf.GitHubSyncConfig, repo string) SyncRepoResult {
	result := SyncRepoResult{Repo: repo}
	owner, name, err := splitRepo(repo)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	cp, err := s.store.GetRepoCheckpoint(ctx, repo)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	maxSeenUpdate := cp.LastIssueUpdatedAt
	currentPage := 1
	issueSyncLogger.Info("issue sync repo started",
		"repo", repo,
		"timeout_seconds", cfg.RequestTimeoutSeconds,
		"page_size", cfg.PageSize,
		"max_pages_per_run", cfg.MaxPagesPerRun,
		"last_issue_updated_at", cp.LastIssueUpdatedAt,
	)
	for page := 0; page < cfg.MaxPagesPerRun; page++ {
		opts := &github.IssueListByRepoOptions{
			State:       "all",
			Sort:        "updated",
			Direction:   "asc",
			Since:       cp.LastIssueUpdatedAt,
			ListOptions: github.ListOptions{Page: currentPage, PerPage: cfg.PageSize},
		}

		pageStartedAt := time.Now()
		issues, resp, fetchErr := s.listByRepoWithRetry(ctx, cfg, owner, name, opts)
		if fetchErr != nil {
			issueSyncLogger.Error("issue sync list issues failed",
				"repo", repo,
				"page", currentPage,
				"duration_ms", time.Since(pageStartedAt).Milliseconds(),
				"timeout_seconds", cfg.RequestTimeoutSeconds,
				"error", fetchErr,
			)
			result.Err = fetchErr.Error()
			_ = s.store.SaveRepoCheckpoint(ctx, dao.Checkpoint{
				Repo:               repo,
				LastSyncedAt:       time.Now(),
				LastIssueUpdatedAt: cp.LastIssueUpdatedAt,
				LastRunStatus:      "failed",
				LastError:          result.Err,
			})
			return result
		}
		if len(issues) == 0 {
			issueSyncLogger.Info("issue sync page empty",
				"repo", repo,
				"page", currentPage,
				"duration_ms", time.Since(pageStartedAt).Milliseconds(),
			)
			break
		}
		issueSyncLogger.Info("issue sync page fetched",
			"repo", repo,
			"page", currentPage,
			"issue_count", len(issues),
			"duration_ms", time.Since(pageStartedAt).Milliseconds(),
			"next_page", func() int {
				if resp == nil {
					return 0
				}
				return resp.NextPage
			}(),
		)

		normalized := make([]dao.SyncedIssue, 0, len(issues))
		for _, it := range issues {
			record := toSyncedIssue(repo, it)
			if s.commentStore != nil {
				if err := s.syncIssueComments(ctx, cfg, repo, owner, name, &record); err != nil {
					issueSyncLogger.Error("issue sync comments failed",
						"repo", repo,
						"issue_number", record.Number,
						"issue_id", record.IssueID,
						"comment_count", record.Comments,
						"error", err,
					)
					result.Err = err.Error()
					_ = s.store.SaveRepoCheckpoint(ctx, dao.Checkpoint{
						Repo:               repo,
						LastSyncedAt:       time.Now(),
						LastIssueUpdatedAt: cp.LastIssueUpdatedAt,
						LastRunStatus:      "failed",
						LastError:          result.Err,
					})
					return result
				}
			}
			normalized = append(normalized, record)
			result.Fetched++
			if record.UpdatedAt.After(maxSeenUpdate) {
				maxSeenUpdate = record.UpdatedAt
			}
		}

		persistStartedAt := time.Now()
		persisted, persistErr := s.store.UpsertIssues(ctx, repo, normalized)
		if persistErr != nil {
			issueSyncLogger.Error("issue sync persist failed",
				"repo", repo,
				"page", currentPage,
				"issue_count", len(normalized),
				"duration_ms", time.Since(persistStartedAt).Milliseconds(),
				"error", persistErr,
			)
			result.Err = persistErr.Error()
			_ = s.store.SaveRepoCheckpoint(ctx, dao.Checkpoint{
				Repo:               repo,
				LastSyncedAt:       time.Now(),
				LastIssueUpdatedAt: cp.LastIssueUpdatedAt,
				LastRunStatus:      "failed",
				LastError:          result.Err,
			})
			return result
		}
		result.Persisted += int32(persisted)
		issueSyncLogger.Info("issue sync page persisted",
			"repo", repo,
			"page", currentPage,
			"issue_count", len(normalized),
			"persisted", persisted,
			"duration_ms", time.Since(persistStartedAt).Milliseconds(),
		)

		if resp == nil || resp.NextPage == 0 {
			break
		}
		currentPage = resp.NextPage
	}

	_ = s.store.SaveRepoCheckpoint(ctx, dao.Checkpoint{
		Repo:               repo,
		LastSyncedAt:       time.Now(),
		LastIssueUpdatedAt: maxSeenUpdate,
		LastRunStatus:      "success",
		LastError:          "",
	})

	return result
}

func (s *IssueSyncService) syncIssueComments(ctx context.Context, cfg conf.GitHubSyncConfig, repo, owner, name string, issue *dao.SyncedIssue) error {
	if issue.Comments <= 0 {
		return nil
	}

	commentFetchStartedAt := time.Now()
	comments, err := s.fetchIssueComments(ctx, cfg, owner, name, int(issue.Number))
	if err != nil {
		return err
	}
	issueSyncLogger.Info("issue sync comments fetched",
		"repo", repo,
		"issue_number", issue.Number,
		"issue_id", issue.IssueID,
		"comment_count", len(comments),
		"duration_ms", time.Since(commentFetchStartedAt).Milliseconds(),
	)

	saveCtx, cancel := requestTimeout(ctx, cfg)
	defer cancel()

	saveStartedAt := time.Now()
	if err := s.commentStore.SaveComments(saveCtx, repo, issue.IssueID, issue.Number, comments); err != nil {
		return fmt.Errorf("save issue comments for %s#%d: %w", repo, issue.Number, err)
	}
	issueSyncLogger.Info("issue sync comments stored",
		"repo", repo,
		"issue_number", issue.Number,
		"issue_id", issue.IssueID,
		"comment_count", len(comments),
		"duration_ms", time.Since(saveStartedAt).Milliseconds(),
		"timeout_seconds", cfg.RequestTimeoutSeconds,
	)
	return nil
}

func (s *IssueSyncService) fetchIssueComments(ctx context.Context, cfg conf.GitHubSyncConfig, owner, repo string, issueNumber int) ([]dao.IssueComment, error) {
	currentPage := 1
	out := make([]dao.IssueComment, 0)
	for {
		pageStartedAt := time.Now()
		requestCtx, cancel := requestTimeout(ctx, cfg)
		items, resp, err := s.client.ListComments(requestCtx, owner, repo, issueNumber, &github.IssueListCommentsOptions{
			Sort:      github.Ptr("created"),
			Direction: github.Ptr("asc"),
			ListOptions: github.ListOptions{
				Page:    currentPage,
				PerPage: 100,
			},
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list issue comments for %s/%s#%d: %w", owner, repo, issueNumber, err)
		}
		issueSyncLogger.Info("issue sync comments page fetched",
			"repo", owner+"/"+repo,
			"issue_number", issueNumber,
			"page", currentPage,
			"comment_count", len(items),
			"duration_ms", time.Since(pageStartedAt).Milliseconds(),
			"next_page", func() int {
				if resp == nil {
					return 0
				}
				return resp.NextPage
			}(),
		)

		for _, item := range items {
			out = append(out, toIssueComment(item))
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		currentPage = resp.NextPage
	}
	return out, nil
}

func (s *IssueSyncService) listByRepoWithRetry(ctx context.Context, cfg conf.GitHubSyncConfig, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		attemptStartedAt := time.Now()
		requestCtx, cancel := requestTimeout(ctx, cfg)
		issues, resp, err := s.client.ListByRepo(requestCtx, owner, repo, opts)
		cancel()
		if err == nil {
			issueSyncLogger.Info("issue sync list issues request succeeded",
				"repo", owner+"/"+repo,
				"page", opts.ListOptions.Page,
				"attempt", attempt+1,
				"issue_count", len(issues),
				"duration_ms", time.Since(attemptStartedAt).Milliseconds(),
				"timeout_seconds", cfg.RequestTimeoutSeconds,
			)
			return issues, resp, nil
		}
		lastErr = err
		issueSyncLogger.Warn("issue sync list issues request failed",
			"repo", owner+"/"+repo,
			"page", opts.ListOptions.Page,
			"attempt", attempt+1,
			"duration_ms", time.Since(attemptStartedAt).Milliseconds(),
			"timeout_seconds", cfg.RequestTimeoutSeconds,
			"error", err,
		)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 300 * time.Millisecond):
		}
	}
	return nil, nil, fmt.Errorf("list issues failed after retries: %w", lastErr)
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(repo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo %q, expect owner/repo", repo)
	}
	return parts[0], parts[1], nil
}

func toSyncedIssue(repo string, issue *github.Issue) dao.SyncedIssue {
	assignees := make([]string, 0, len(issue.Assignees))
	for _, a := range issue.Assignees {
		assignees = append(assignees, a.GetLogin())
	}
	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		labels = append(labels, l.GetName())
	}
	raw, _ := json.Marshal(issue)

	var closedAt *time.Time
	if issue.ClosedAt != nil {
		v := issue.ClosedAt.Time
		closedAt = &v
	}

	isPR := issue.PullRequestLinks != nil
	return dao.SyncedIssue{
		Repo:          repo,
		IssueID:       issue.GetID(),
		Number:        int32(issue.GetNumber()),
		Title:         issue.GetTitle(),
		Body:          issue.GetBody(),
		State:         issue.GetState(),
		Author:        issue.GetUser().GetLogin(),
		Assignees:     assignees,
		Labels:        labels,
		Comments:      int32(issue.GetComments()),
		IsPullRequest: isPR,
		HTMLURL:       issue.GetHTMLURL(),
		CreatedAt:     issue.GetCreatedAt().Time,
		UpdatedAt:     issue.GetUpdatedAt().Time,
		ClosedAt:      closedAt,
		Raw:           string(raw),
	}
}

func toIssueComment(in *github.IssueComment) dao.IssueComment {
	out := dao.IssueComment{
		ID:      in.GetID(),
		Body:    in.GetBody(),
		HTMLURL: in.GetHTMLURL(),
	}
	if in.User != nil {
		out.UserLogin = in.User.GetLogin()
		out.UserURL = in.User.GetHTMLURL()
		out.UserAvatarURL = in.User.GetAvatarURL()
	}
	if in.CreatedAt != nil {
		out.CreatedAt = in.GetCreatedAt().Time
	}
	if in.UpdatedAt != nil {
		out.UpdatedAt = in.GetUpdatedAt().Time
	}
	return out
}
