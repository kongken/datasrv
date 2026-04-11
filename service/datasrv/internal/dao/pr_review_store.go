package dao

import (
	"context"
	"errors"
	"time"
)

var ErrPRReviewNotFound = errors.New("pr review not found")

// PRReview is the persistence model for AI-generated pull request reviews.
type PRReview struct {
	Repo          string
	IssueID       int64
	Number        int32
	ReviewSummary string
	RiskAreas     string
	Suggestions   string
	RawDiffSize   int
	ModelUsed     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PRReviewFilter defines list parameters for PR review queries.
type PRReviewFilter struct {
	Repo   string
	Number int32
	Offset int
	Limit  int
}

// PRReviewStore is the abstraction over persistence for PR AI reviews.
type PRReviewStore interface {
	UpsertPRReview(ctx context.Context, review PRReview) error
	GetPRReview(ctx context.Context, repo string, number int32) (PRReview, error)
	ListPRReviews(ctx context.Context, filter PRReviewFilter) ([]PRReview, error)
	ListUnreviewedPRs(ctx context.Context, repos []string, limit int) ([]SyncedIssue, error)
}
