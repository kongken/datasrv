package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	corelog "butterfly.orx.me/core/log"
	"github.com/google/go-github/v82/github"
	genpkg "github.com/kongken/datasrv/pkg/gen"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"golang.org/x/oauth2"
)

const defaultMaxDiffSize = 100 * 1024 // 100KB

// PRReviewService orchestrates periodic AI review generation for pull requests.
type PRReviewService struct {
	store    dao.SyncStore
	prStore  dao.PRReviewStore
	reviewer *genpkg.PRReviewer
	ghClient *github.Client
	cfg      conf.PRReviewConfig
}

type PRReviewRunSummary struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []PRReviewRepoResult
}

type PRReviewRepoResult struct {
	Repo     string
	Scanned  int
	Reviewed int
	Skipped  int
	Failed   int
	Errors   []string
}

func NewPRReviewService(store dao.SyncStore, prStore dao.PRReviewStore, reviewer *genpkg.PRReviewer, ghCfg conf.GitHubConfig, cfg conf.PRReviewConfig) *PRReviewService {
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

	return &PRReviewService{
		store:    store,
		prStore:  prStore,
		reviewer: reviewer,
		ghClient: ghClient,
		cfg:      cfg,
	}
}

func (s *PRReviewService) GetConfig() conf.PRReviewConfig {
	return s.cfg
}

func (s *PRReviewService) Run(ctx context.Context) (PRReviewRunSummary, error) {
	summary := PRReviewRunSummary{
		StartedAt: time.Now().UTC(),
	}
	defer func() {
		summary.FinishedAt = time.Now().UTC()
	}()

	if s == nil || s.store == nil || s.prStore == nil {
		return summary, fmt.Errorf("pr review store is not initialized")
	}
	if s.reviewer == nil {
		return summary, fmt.Errorf("pr reviewer is not initialized")
	}

	logger := corelog.FromContext(ctx).With("component", "datasrv.pr_review")

	repos, err := s.store.ListManagedRepos(ctx)
	if err != nil {
		return summary, fmt.Errorf("list managed repos: %w", err)
	}

	repoNames := make([]string, 0, len(repos))
	for _, repo := range repos {
		repoNames = append(repoNames, repo.Repo)
	}

	maxPRs := s.cfg.MaxPRsPerRun
	if maxPRs <= 0 {
		maxPRs = 20
	}

	logger.Info("pr review run started",
		"repo_count", len(repos),
		"max_prs_per_run", maxPRs,
	)

	prs, err := s.prStore.ListUnreviewedPRs(ctx, repoNames, maxPRs)
	if err != nil {
		return summary, fmt.Errorf("list unreviewed prs: %w", err)
	}

	logger.Info("pr review found unreviewed prs", "count", len(prs))

	maxDiffSize := s.cfg.MaxDiffSize
	if maxDiffSize <= 0 {
		maxDiffSize = defaultMaxDiffSize
	}

	resultByRepo := make(map[string]*PRReviewRepoResult)
	repoOrder := make([]string, 0)
	getResult := func(repo string) *PRReviewRepoResult {
		if result, ok := resultByRepo[repo]; ok {
			return result
		}
		result := &PRReviewRepoResult{Repo: repo}
		resultByRepo[repo] = result
		repoOrder = append(repoOrder, repo)
		return result
	}

	for _, pr := range prs {
		result := getResult(pr.Repo)
		result.Scanned++

		prLogger := logger.With("repo", pr.Repo, "number", pr.Number, "title", pr.Title)

		// Parse owner/repo
		parts := strings.SplitN(pr.Repo, "/", 2)
		if len(parts) != 2 {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invalid repo format %q for PR #%d", pr.Repo, pr.Number))
			prLogger.Error("pr review invalid repo format")
			continue
		}
		owner, repoName := parts[0], parts[1]

		// Fetch diff
		diff, err := s.fetchPRDiff(ctx, owner, repoName, int(pr.Number), maxDiffSize)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("fetch diff %s#%d: %v", pr.Repo, pr.Number, err))
			prLogger.Error("pr review fetch diff failed", "error", err)
			continue
		}

		if strings.TrimSpace(diff) == "" {
			result.Skipped++
			prLogger.Info("pr review skipped empty diff")
			continue
		}

		// Generate review with timeout
		callCtx := ctx
		cancel := func() {}
		if timeout := time.Duration(s.cfg.RequestTimeoutSeconds) * time.Second; timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, timeout)
		}

		reviewResult, err := s.reviewer.ReviewPR(callCtx, pr.Title, pr.Body, diff)
		cancel()
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("review %s#%d: %v", pr.Repo, pr.Number, err))
			prLogger.Error("pr review generation failed", "error", err)
			continue
		}

		// Persist review
		review := dao.PRReview{
			Repo:          pr.Repo,
			IssueID:       pr.IssueID,
			Number:        pr.Number,
			ReviewSummary: reviewResult.ReviewSummary,
			RiskAreas:     reviewResult.RiskAreas,
			Suggestions:   reviewResult.Suggestions,
			RawDiffSize:   len(diff),
			ModelUsed:     s.cfg.Model,
		}

		if err := s.prStore.UpsertPRReview(ctx, review); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("persist review %s#%d: %v", pr.Repo, pr.Number, err))
			prLogger.Error("pr review persist failed", "error", err)
			continue
		}

		result.Reviewed++
		prLogger.Info("pr review completed",
			"reviewed", result.Reviewed,
			"scanned", result.Scanned,
		)
	}

	for _, repo := range repoOrder {
		summary.Results = append(summary.Results, *resultByRepo[repo])
	}

	logger.Info("pr review run completed",
		"started_at", summary.StartedAt,
		"finished_at", summary.FinishedAt,
		"repo_count", len(summary.Results),
	)

	return summary, nil
}

func (s *PRReviewService) fetchPRDiff(ctx context.Context, owner, repo string, number int, maxSize int) (string, error) {
	diff, _, err := s.ghClient.PullRequests.GetRaw(ctx, owner, repo, number, github.RawOptions{
		Type: github.Diff,
	})
	if err != nil {
		return "", fmt.Errorf("get pr diff: %w", err)
	}

	if len(diff) > maxSize {
		logger := corelog.FromContext(ctx).With("component", "datasrv.pr_review")
		logger.Warn("pr diff truncated",
			"repo", fmt.Sprintf("%s/%s", owner, repo),
			"number", number,
			"original_size", len(diff),
			"max_size", maxSize,
		)
		diff = diff[:maxSize]
	}

	return diff, nil
}
