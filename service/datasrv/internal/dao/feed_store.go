package dao

import (
	"context"
	"errors"
	"time"
)

var (
	ErrFeedSourceNotFound  = errors.New("feed source not found")
	ErrFeedContentNotFound = errors.New("feed content not found")
)

type FeedSource struct {
	ID            string
	URL           string
	DisplayName   string
	Description   string
	SiteURL       string
	Enabled       bool
	ETag          string
	LastModified  string
	LastSyncedAt  time.Time
	LastSuccessAt time.Time
	LastRunStatus string
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type FeedSourceFilter struct {
	Offset int
	Limit  int
}

type FeedContent struct {
	ID           string
	FeedSourceID string
	Identity     string
	GUID         string
	Title        string
	Summary      string
	Content      string
	Link         string
	Author       string
	Categories   []string
	PublishedAt  time.Time
	UpdatedAt    time.Time
	FetchedAt    time.Time
}

type FeedContentFilter struct {
	FeedSourceID string
	ContentID    string
	Offset       int
	Limit        int
}

type FeedCheckpoint struct {
	FeedSourceID  string
	LastSyncedAt  time.Time
	LastSuccessAt time.Time
	LastRunStatus string
	LastError     string
	ETag          string
	LastModified  string
	UpdatedAt     time.Time
}

type FeedSourceStore interface {
	UpsertFeedSource(ctx context.Context, source FeedSource) (FeedSource, error)
	GetFeedSource(ctx context.Context, id string) (FeedSource, error)
	ListFeedSources(ctx context.Context, filter FeedSourceFilter) ([]FeedSource, error)
	DeleteFeedSource(ctx context.Context, id string) error
}

type FeedContentStore interface {
	UpsertFeedContents(ctx context.Context, sourceID string, contents []FeedContent) (int, error)
	ListFeedContents(ctx context.Context, filter FeedContentFilter) ([]FeedContent, error)
	GetFeedContent(ctx context.Context, id string) (FeedContent, error)
}

type FeedSyncStateStore interface {
	GetFeedCheckpoint(ctx context.Context, sourceID string) (FeedCheckpoint, error)
	SaveFeedCheckpoint(ctx context.Context, checkpoint FeedCheckpoint) error
}

type FeedStore interface {
	FeedSourceStore
	FeedContentStore
	FeedSyncStateStore
	Close() error
}
