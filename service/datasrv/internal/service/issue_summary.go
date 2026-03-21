package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	corelog "butterfly.orx.me/core/log"
	genpkg "github.com/kongken/datasrv/pkg/gen"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type IssueSummarizer interface {
	SummarizeIssue(ctx context.Context, issue genpkg.Issue, replies []genpkg.IssueReply) (string, error)
}

type IssueSummaryService struct {
	store        dao.SyncStore
	commentStore IssueCommentStore
	summarizer   IssueSummarizer
	cfg          conf.IssueSummaryConfig
}

type IssueSummaryRunSummary struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []IssueSummaryRepoResult
}

type IssueSummaryRepoResult struct {
	Repo    string
	Scanned int
	Updated int
	Skipped int
	Failed  int
	Stopped bool
	Errors  []string
}

func NewIssueSummaryService(store dao.SyncStore, commentStore IssueCommentStore, summarizer IssueSummarizer, cfg conf.IssueSummaryConfig) *IssueSummaryService {
	return &IssueSummaryService{
		store:        store,
		commentStore: commentStore,
		summarizer:   summarizer,
		cfg:          cfg,
	}
}

func (s *IssueSummaryService) GetConfig() conf.IssueSummaryConfig {
	return s.cfg
}

func (s *IssueSummaryService) Run(ctx context.Context) (IssueSummaryRunSummary, error) {
	summary := IssueSummaryRunSummary{
		StartedAt: time.Now().UTC(),
	}
	defer func() {
		summary.FinishedAt = time.Now().UTC()
	}()

	if s == nil || s.store == nil {
		return summary, fmt.Errorf("issue summary store is not initialized")
	}
	if s.summarizer == nil {
		return summary, fmt.Errorf("issue summarizer is not initialized")
	}

	repos, err := s.store.ListManagedRepos(ctx)
	if err != nil {
		return summary, fmt.Errorf("list managed repos: %w", err)
	}
	logger := corelog.FromContext(ctx).With("component", "datasrv.issue_summary")
	logger.Info("issue summary run started",
		"repo_count", len(repos),
		"batch_size", s.cfg.BatchSize,
		"max_issues_per_run", s.cfg.MaxIssuesPerRun,
		"state", s.cfg.State,
		"overwrite_existing", s.cfg.OverwriteExisting,
	)

	remaining := s.cfg.MaxIssuesPerRun
	for _, repo := range repos {
		if remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
			logger.Info("issue summary run reached max issues per run before repo",
				"repo", repo.Repo,
				"max_issues_per_run", s.cfg.MaxIssuesPerRun,
			)
			break
		}
		result, err := s.runRepo(ctx, repo.Repo, &remaining)
		if err != nil {
			return summary, err
		}
		summary.Results = append(summary.Results, result)
	}

	logger.Info("issue summary run completed",
		"started_at", summary.StartedAt,
		"finished_at", summary.FinishedAt,
		"repo_count", len(summary.Results),
	)

	return summary, nil
}

func (s *IssueSummaryService) runRepo(ctx context.Context, repo string, remaining *int) (IssueSummaryRepoResult, error) {
	result := IssueSummaryRepoResult{Repo: repo}
	batchSize := s.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 20
	}

	state := strings.TrimSpace(s.cfg.State)
	if state == "" {
		state = "all"
	}

	logger := corelog.FromContext(ctx).With("component", "datasrv.issue_summary", "repo", repo)
	logger.Info("issue summary repo scan started",
		"repo", repo,
		"batch_size", batchSize,
		"state", state,
		"overwrite_existing", s.cfg.OverwriteExisting,
	)

	for offset := 0; ; offset += batchSize {
		if remaining != nil && *remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
			result.Stopped = true
			logger.Info("issue summary repo scan stopped by max issues per run",
				"scanned", result.Scanned,
				"updated", result.Updated,
				"skipped", result.Skipped,
				"failed", result.Failed,
			)
			return result, nil
		}

		logger.Debug("issue summary loading issue page",
			"offset", offset,
			"limit", batchSize,
			"remaining", remainingValue(remaining),
		)
		rows, err := s.store.ListIssues(ctx, dao.SyncIssueFilter{
			Repo:   repo,
			State:  state,
			Offset: offset,
			Limit:  batchSize,
		})
		if err != nil {
			logger.Error("issue summary list issues failed",
				"offset", offset,
				"limit", batchSize,
				"error", err,
			)
			return result, fmt.Errorf("list issues for repo %s: %w", repo, err)
		}
		if len(rows) == 0 {
			logger.Info("issue summary repo scan completed",
				"scanned", result.Scanned,
				"updated", result.Updated,
				"skipped", result.Skipped,
				"failed", result.Failed,
			)
			return result, nil
		}
		logger.Debug("issue summary issue page loaded",
			"offset", offset,
			"count", len(rows),
		)

		for _, row := range rows {
			if remaining != nil && *remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
				result.Stopped = true
				logger.Info("issue summary repo scan stopped mid-page by max issues per run",
					"number", row.Number,
					"scanned", result.Scanned,
					"updated", result.Updated,
					"skipped", result.Skipped,
					"failed", result.Failed,
				)
				return result, nil
			}
			result.Scanned++
			if remaining != nil && s.cfg.MaxIssuesPerRun > 0 {
				*remaining--
			}
			issueLogger := logger.With("issue_id", row.IssueID, "number", row.Number)
			issueLogger.Debug("issue summary processing issue",
				"issue_id", row.IssueID,
				"number", row.Number,
				"title", row.Title,
				"has_existing_summary", strings.TrimSpace(row.AISummary) != "",
				"comment_count", row.Comments,
				"remaining", remainingValue(remaining),
			)

			if !s.cfg.OverwriteExisting && strings.TrimSpace(row.AISummary) != "" {
				result.Skipped++
				issueLogger.Debug("issue summary skipped existing summary")
				continue
			}

			replies, err := s.loadReplies(ctx, row)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("load replies for %s#%d: %v", row.Repo, row.Number, err))
				issueLogger.Error("issue summary load replies failed", "error", err)
				continue
			}
			issueLogger.Debug("issue summary replies loaded",
				"reply_count", len(replies),
			)

			callCtx := ctx
			cancel := func() {}
			if timeout := time.Duration(s.cfg.RequestTimeoutSeconds) * time.Second; timeout > 0 {
				callCtx, cancel = context.WithTimeout(ctx, timeout)
			}

			issueSummary, err := s.summarizer.SummarizeIssue(callCtx, toGenIssue(row), replies)
			cancel()
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("summarize %s#%d: %v", row.Repo, row.Number, err))
				issueLogger.Error("issue summary generation failed", "error", err)
				continue
			}
			issueLogger.Debug("issue summary generated",
				"summary_length", len(strings.TrimSpace(issueSummary)),
			)

			if _, err := s.store.UpdateIssueAISummary(ctx, row.Repo, row.IssueID, row.Number, strings.TrimSpace(issueSummary)); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("update ai summary for %s#%d: %v", row.Repo, row.Number, err))
				issueLogger.Error("issue summary update failed", "error", err)
				continue
			}
			result.Updated++
			issueLogger.Info("issue summary updated",
				"updated", result.Updated,
				"scanned", result.Scanned,
			)
		}

		if len(rows) < batchSize {
			logger.Info("issue summary repo scan completed with partial page",
				"scanned", result.Scanned,
				"updated", result.Updated,
				"skipped", result.Skipped,
				"failed", result.Failed,
			)
			return result, nil
		}
	}
}

func (s *IssueSummaryService) loadReplies(ctx context.Context, issue dao.SyncedIssue) ([]genpkg.IssueReply, error) {
	logger := corelog.FromContext(ctx).With("component", "datasrv.issue_summary", "repo", issue.Repo, "issue_id", issue.IssueID, "number", issue.Number)
	if s.commentStore == nil || issue.Comments <= 0 {
		logger.Debug("issue summary replies skipped",
			"has_comment_store", s.commentStore != nil,
			"comment_count", issue.Comments,
		)
		return nil, nil
	}
	logger.Debug("issue summary loading replies from store",
		"comment_count", issue.Comments,
	)
	comments, err := s.commentStore.LoadComments(ctx, issue.Repo, issue.IssueID, issue.Number)
	if err != nil {
		return nil, err
	}
	replies := make([]genpkg.IssueReply, 0, len(comments))
	for _, comment := range comments {
		replies = append(replies, genpkg.IssueReply{
			ID:        comment.ID,
			Author:    comment.UserLogin,
			Body:      comment.Body,
			HTMLURL:   comment.HTMLURL,
			CreatedAt: comment.CreatedAt,
			UpdatedAt: comment.UpdatedAt,
		})
	}
	return replies, nil
}

func remainingValue(remaining *int) any {
	if remaining == nil {
		return nil
	}
	return *remaining
}

func toGenIssue(issue dao.SyncedIssue) genpkg.Issue {
	return genpkg.Issue{
		Repo:      issue.Repo,
		Number:    issue.Number,
		Title:     issue.Title,
		State:     issue.State,
		Author:    issue.Author,
		Body:      issue.Body,
		Labels:    append([]string(nil), issue.Labels...),
		HTMLURL:   issue.HTMLURL,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
	}
}
