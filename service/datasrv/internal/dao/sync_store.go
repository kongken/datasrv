package dao

import (
	"context"
	"time"
)

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
	Raw           string
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

// SyncIssueFilter defines list parameters for admin read APIs.
type SyncIssueFilter struct {
	Repo   string
	Offset int
	Limit  int
}

// SyncStore is the abstraction over persistence backends used by sync logic.
type SyncStore interface {
	UpsertIssues(ctx context.Context, repo string, issues []SyncedIssue) (int, error)
	ListIssues(ctx context.Context, filter SyncIssueFilter) ([]SyncedIssue, error)
	GetRepoCheckpoint(ctx context.Context, repo string) (Checkpoint, error)
	SaveRepoCheckpoint(ctx context.Context, checkpoint Checkpoint) error
	ListCheckpoints(ctx context.Context) ([]Checkpoint, error)
	Close() error
}
