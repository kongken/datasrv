package dao

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMongoSyncStore_UpsertAndCheckpoint(t *testing.T) {
	uri := os.Getenv("DATASRV_TEST_MONGO_URI")
	db := os.Getenv("DATASRV_TEST_MONGO_DB")
	if uri == "" || db == "" {
		t.Skip("set DATASRV_TEST_MONGO_URI and DATASRV_TEST_MONGO_DB to run mongo integration test")
	}

	store, err := NewMongoSyncStore(uri, db)
	if err != nil {
		t.Fatalf("NewMongoSyncStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	updated := time.Now().UTC().Round(time.Second)
	persisted, err := store.UpsertIssues(ctx, "owner/repo", []SyncedIssue{{
		Repo:      "owner/repo",
		IssueID:   1001,
		Number:    1,
		Title:     "test issue",
		State:     "open",
		Author:    "bot",
		UpdatedAt: updated,
	}})
	if err != nil {
		t.Fatalf("UpsertIssues() error = %v", err)
	}
	if persisted != 1 {
		t.Fatalf("persisted = %d, want 1", persisted)
	}

	err = store.SaveRepoCheckpoint(ctx, Checkpoint{
		Repo:               "owner/repo",
		LastSyncedAt:       time.Now().UTC(),
		LastIssueUpdatedAt: updated,
		LastRunStatus:      "success",
	})
	if err != nil {
		t.Fatalf("SaveRepoCheckpoint() error = %v", err)
	}

	cp, err := store.GetRepoCheckpoint(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetRepoCheckpoint() error = %v", err)
	}
	if cp.Repo != "owner/repo" {
		t.Fatalf("checkpoint repo = %q, want owner/repo", cp.Repo)
	}
}

func TestGormSyncStore_UpsertAndCheckpoint(t *testing.T) {
	dsn := os.Getenv("DATASRV_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DATASRV_TEST_POSTGRES_DSN to run postgres integration test")
	}

	store, err := NewGormSyncStore(dsn)
	if err != nil {
		t.Fatalf("NewGormSyncStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	updated := time.Now().UTC().Round(time.Second)
	persisted, err := store.UpsertIssues(ctx, "owner/repo", []SyncedIssue{{
		Repo:      "owner/repo",
		IssueID:   2001,
		Number:    2,
		Title:     "test issue",
		State:     "open",
		Author:    "bot",
		UpdatedAt: updated,
	}})
	if err != nil {
		t.Fatalf("UpsertIssues() error = %v", err)
	}
	if persisted != 1 {
		t.Fatalf("persisted = %d, want 1", persisted)
	}

	err = store.SaveRepoCheckpoint(ctx, Checkpoint{
		Repo:               "owner/repo",
		LastSyncedAt:       time.Now().UTC(),
		LastIssueUpdatedAt: updated,
		LastRunStatus:      "success",
	})
	if err != nil {
		t.Fatalf("SaveRepoCheckpoint() error = %v", err)
	}

	cp, err := store.GetRepoCheckpoint(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetRepoCheckpoint() error = %v", err)
	}
	if cp.Repo != "owner/repo" {
		t.Fatalf("checkpoint repo = %q, want owner/repo", cp.Repo)
	}
}

func TestMongoSyncStore_FeedSourceContentAndCheckpoint(t *testing.T) {
	uri := os.Getenv("DATASRV_TEST_MONGO_URI")
	db := os.Getenv("DATASRV_TEST_MONGO_DB")
	if uri == "" || db == "" {
		t.Skip("set DATASRV_TEST_MONGO_URI and DATASRV_TEST_MONGO_DB to run mongo integration test")
	}

	store, err := NewMongoSyncStore(uri, db)
	if err != nil {
		t.Fatalf("NewMongoSyncStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	assertFeedStoreLifecycle(t, store)
}

func TestGormSyncStore_FeedSourceContentAndCheckpoint(t *testing.T) {
	dsn := os.Getenv("DATASRV_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set DATASRV_TEST_POSTGRES_DSN to run postgres integration test")
	}

	store, err := NewGormSyncStore(dsn)
	if err != nil {
		t.Fatalf("NewGormSyncStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	assertFeedStoreLifecycle(t, store)
}

func assertFeedStoreLifecycle(t *testing.T, store FeedStore) {
	t.Helper()

	ctx := context.Background()
	sourceID := uniqueIntegrationID(t, "feed")
	source, err := store.UpsertFeedSource(ctx, FeedSource{
		ID:          sourceID,
		URL:         "https://example.com/feed.xml",
		DisplayName: "Integration Feed",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("UpsertFeedSource() error = %v", err)
	}
	if source.ID != sourceID {
		t.Fatalf("source id = %q, want %q", source.ID, sourceID)
	}

	listed, err := store.ListFeedSources(ctx, FeedSourceFilter{})
	if err != nil {
		t.Fatalf("ListFeedSources() error = %v", err)
	}
	if len(listed) == 0 {
		t.Fatal("ListFeedSources() returned no sources")
	}

	published := time.Now().UTC().Round(time.Second)
	contentID := uniqueIntegrationID(t, "item")
	identity := uniqueIntegrationID(t, "guid")
	persisted, err := store.UpsertFeedContents(ctx, sourceID, []FeedContent{{
		ID:           contentID,
		FeedSourceID: sourceID,
		Identity:     identity,
		GUID:         identity,
		Title:        "Integration Entry",
		PublishedAt:  published,
		FetchedAt:    published,
	}})
	if err != nil {
		t.Fatalf("UpsertFeedContents() error = %v", err)
	}
	if persisted != 1 {
		t.Fatalf("persisted = %d, want 1", persisted)
	}

	if _, err := store.UpsertFeedContents(ctx, sourceID, []FeedContent{{
		ID:           contentID,
		FeedSourceID: sourceID,
		Identity:     identity,
		GUID:         identity,
		Title:        "Integration Entry Updated",
		PublishedAt:  published,
		FetchedAt:    published,
	}}); err != nil {
		t.Fatalf("second UpsertFeedContents() error = %v", err)
	}

	contents, err := store.ListFeedContents(ctx, FeedContentFilter{FeedSourceID: sourceID})
	if err != nil {
		t.Fatalf("ListFeedContents() error = %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(contents))
	}
	if contents[0].Title != "Integration Entry Updated" {
		t.Fatalf("content title = %q, want updated title", contents[0].Title)
	}

	err = store.SaveFeedCheckpoint(ctx, FeedCheckpoint{
		FeedSourceID:  sourceID,
		LastSyncedAt:  published,
		LastSuccessAt: published,
		LastRunStatus: "success",
		ETag:          `"abc"`,
	})
	if err != nil {
		t.Fatalf("SaveFeedCheckpoint() error = %v", err)
	}

	checkpoint, err := store.GetFeedCheckpoint(ctx, sourceID)
	if err != nil {
		t.Fatalf("GetFeedCheckpoint() error = %v", err)
	}
	if checkpoint.FeedSourceID != sourceID {
		t.Fatalf("checkpoint source id = %q, want %q", checkpoint.FeedSourceID, sourceID)
	}

	if err := store.DeleteFeedSource(ctx, sourceID); err != nil {
		t.Fatalf("DeleteFeedSource() error = %v", err)
	}
}

func uniqueIntegrationID(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
