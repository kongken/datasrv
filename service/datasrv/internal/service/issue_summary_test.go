package service

import (
	"context"
	"errors"
	"testing"
	"time"

	genpkg "github.com/kongken/datasrv/pkg/gen"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type fakeIssueSummarizer struct {
	calls []fakeIssueSummarizerCall
	err   error
	text  string
}

type fakeIssueSummarizerCall struct {
	Issue   genpkg.Issue
	Replies []genpkg.IssueReply
}

func (f *fakeIssueSummarizer) SummarizeIssue(_ context.Context, issue genpkg.Issue, replies []genpkg.IssueReply) (string, error) {
	f.calls = append(f.calls, fakeIssueSummarizerCall{
		Issue:   issue,
		Replies: append([]genpkg.IssueReply(nil), replies...),
	})
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

func TestIssueSummaryServiceRunUpdatesMissingSummary(t *testing.T) {
	store := newFakeSyncStore()
	commentStore := newFakeIssueCommentStore()
	now := time.Now().UTC()

	_, _ = store.ReplaceManagedRepos(context.Background(), []string{"o/r"})
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{
		{
			Repo:      "o/r",
			IssueID:   10,
			Number:    100,
			Title:     "missing summary",
			Body:      "issue body",
			State:     "open",
			Author:    "alice",
			Comments:  1,
			UpdatedAt: now,
		},
		{
			Repo:      "o/r",
			IssueID:   11,
			Number:    101,
			Title:     "has summary",
			Body:      "issue body 2",
			State:     "open",
			Author:    "bob",
			AISummary: "existing",
			UpdatedAt: now.Add(-time.Minute),
		},
	})
	commentStore.saved["o/r/10-100.json"] = []dao.IssueComment{{
		ID:        1,
		Body:      "first reply",
		UserLogin: "reviewer",
		CreatedAt: now,
		UpdatedAt: now,
	}}

	summarizer := &fakeIssueSummarizer{text: "generated summary"}
	svc := NewIssueSummaryService(store, commentStore, summarizer, conf.IssueSummaryConfig{
		Enabled:         true,
		BatchSize:       10,
		MaxIssuesPerRun: 10,
	})

	summary, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(summary.Results))
	}
	if got := summary.Results[0].Updated; got != 1 {
		t.Fatalf("updated = %d, want 1", got)
	}
	if got := summary.Results[0].Skipped; got != 1 {
		t.Fatalf("skipped = %d, want 1", got)
	}
	if len(summarizer.calls) != 1 {
		t.Fatalf("summarizer calls = %d, want 1", len(summarizer.calls))
	}
	if summarizer.calls[0].Issue.Number != 100 {
		t.Fatalf("summarized issue number = %d, want 100", summarizer.calls[0].Issue.Number)
	}
	if len(summarizer.calls[0].Replies) != 1 || summarizer.calls[0].Replies[0].Body != "first reply" {
		t.Fatalf("replies = %#v, want first reply", summarizer.calls[0].Replies)
	}

	rows, err := store.ListIssues(context.Background(), dao.SyncIssueFilter{Repo: "o/r", Number: 100, Limit: 1})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if rows[0].AISummary != "generated summary" {
		t.Fatalf("ai_summary = %q, want generated summary", rows[0].AISummary)
	}
}

func TestIssueSummaryServiceRunCollectsPerIssueFailures(t *testing.T) {
	store := newFakeSyncStore()
	_, _ = store.ReplaceManagedRepos(context.Background(), []string{"o/r"})
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{{
		Repo:      "o/r",
		IssueID:   10,
		Number:    100,
		Title:     "broken",
		Body:      "issue body",
		State:     "open",
		Author:    "alice",
		UpdatedAt: time.Now().UTC(),
	}})

	svc := NewIssueSummaryService(store, nil, &fakeIssueSummarizer{err: errors.New("boom")}, conf.IssueSummaryConfig{
		Enabled:   true,
		BatchSize: 10,
	})

	summary, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := summary.Results[0].Failed; got != 1 {
		t.Fatalf("failed = %d, want 1", got)
	}
	if len(summary.Results[0].Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(summary.Results[0].Errors))
	}
}

func TestIssueSummaryServiceRunPrioritizesLatestIssuesAcrossRepos(t *testing.T) {
	store := newFakeSyncStore()
	now := time.Now().UTC()

	_, _ = store.ReplaceManagedRepos(context.Background(), []string{"o/old", "o/new"})
	_, _ = store.UpsertIssues(context.Background(), "o/old", []dao.SyncedIssue{{
		Repo:      "o/old",
		IssueID:   10,
		Number:    100,
		Title:     "older issue",
		Body:      "older body",
		State:     "open",
		Author:    "alice",
		UpdatedAt: now.Add(-time.Hour),
	}})
	_, _ = store.UpsertIssues(context.Background(), "o/new", []dao.SyncedIssue{{
		Repo:      "o/new",
		IssueID:   11,
		Number:    101,
		Title:     "newer issue",
		Body:      "newer body",
		State:     "open",
		Author:    "bob",
		UpdatedAt: now,
	}})

	summarizer := &fakeIssueSummarizer{text: "generated summary"}
	svc := NewIssueSummaryService(store, nil, summarizer, conf.IssueSummaryConfig{
		Enabled:         true,
		BatchSize:       10,
		MaxIssuesPerRun: 1,
	})

	summary, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(summarizer.calls) != 1 {
		t.Fatalf("summarizer calls = %d, want 1", len(summarizer.calls))
	}
	if summarizer.calls[0].Issue.Repo != "o/new" {
		t.Fatalf("first summarized repo = %q, want o/new", summarizer.calls[0].Issue.Repo)
	}
	if summary.Results[0].Repo != "o/new" {
		t.Fatalf("first summary result repo = %q, want o/new", summary.Results[0].Repo)
	}
}
