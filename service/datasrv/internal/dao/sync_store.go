package dao

import (
	"context"
	"errors"
	"time"
)

var ErrIssueNotFound = errors.New("issue not found")

// SyncedIssue is the normalized persistence model used by sync workers.
type SyncedIssue struct {
	Repo          string
	IssueID       int64
	Number        int32
	Title         string
	Body          string
	State         string
	Author        string
	Assignees     []string
	Labels        []string
	Comments      int32
	IsPullRequest bool
	HTMLURL       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
	AISummary     string
	Raw           string
}

type IssueComment struct {
	ID            int64
	Body          string
	UserLogin     string
	UserURL       string
	UserAvatarURL string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	HTMLURL       string
}

// Checkpoint tracks last sync state per repository.
type Checkpoint struct {
	Repo               string
	LastSyncedAt       time.Time
	LastIssueUpdatedAt time.Time
	LastRunStatus      string
	LastError          string
	UpdatedAt          time.Time
}

type ManagedRepo struct {
	Repo      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SyncIssueFilter defines list parameters for admin read APIs.
type SyncIssueFilter struct {
	Repo    string
	State   string
	IssueID int64
	Number  int32
	Offset  int
	Limit   int
}

// SyncStore is the abstraction over persistence backends used by sync logic.
type SyncStore interface {
	UpsertIssues(ctx context.Context, repo string, issues []SyncedIssue) (int, error)
	ListIssues(ctx context.Context, filter SyncIssueFilter) ([]SyncedIssue, error)
	UpdateIssueAISummary(ctx context.Context, repo string, issueID int64, number int32, summary string) (SyncedIssue, error)
	ClearIssueAISummaries(ctx context.Context, repo string) (int, error)
	ListManagedRepos(ctx context.Context) ([]ManagedRepo, error)
	ReplaceManagedRepos(ctx context.Context, repos []string) ([]ManagedRepo, error)
	GetRepoCheckpoint(ctx context.Context, repo string) (Checkpoint, error)
	SaveRepoCheckpoint(ctx context.Context, checkpoint Checkpoint) error
	ListCheckpoints(ctx context.Context) ([]Checkpoint, error)
	Close() error
}
