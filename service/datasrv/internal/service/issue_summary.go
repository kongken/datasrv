package service

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	remaining := s.cfg.MaxIssuesPerRun
	for _, repo := range repos {
		if remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
			break
		}
		result, err := s.runRepo(ctx, repo.Repo, &remaining)
		if err != nil {
			return summary, err
		}
		summary.Results = append(summary.Results, result)
	}

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

	for offset := 0; ; offset += batchSize {
		if remaining != nil && *remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
			result.Stopped = true
			return result, nil
		}

		rows, err := s.store.ListIssues(ctx, dao.SyncIssueFilter{
			Repo:   repo,
			State:  state,
			Offset: offset,
			Limit:  batchSize,
		})
		if err != nil {
			return result, fmt.Errorf("list issues for repo %s: %w", repo, err)
		}
		if len(rows) == 0 {
			return result, nil
		}

		for _, row := range rows {
			if remaining != nil && *remaining == 0 && s.cfg.MaxIssuesPerRun > 0 {
				result.Stopped = true
				return result, nil
			}
			result.Scanned++
			if remaining != nil && s.cfg.MaxIssuesPerRun > 0 {
				*remaining--
			}

			if !s.cfg.OverwriteExisting && strings.TrimSpace(row.AISummary) != "" {
				result.Skipped++
				continue
			}

			replies, err := s.loadReplies(ctx, row)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("load replies for %s#%d: %v", row.Repo, row.Number, err))
				continue
			}

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
				continue
			}

			if _, err := s.store.UpdateIssueAISummary(ctx, row.Repo, row.IssueID, row.Number, strings.TrimSpace(issueSummary)); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("update ai summary for %s#%d: %v", row.Repo, row.Number, err))
				continue
			}
			result.Updated++
		}

		if len(rows) < batchSize {
			return result, nil
		}
	}
}

func (s *IssueSummaryService) loadReplies(ctx context.Context, issue dao.SyncedIssue) ([]genpkg.IssueReply, error) {
	if s.commentStore == nil || issue.Comments <= 0 {
		return nil, nil
	}
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
