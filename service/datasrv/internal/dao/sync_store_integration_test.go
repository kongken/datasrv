package dao

import (
	"context"
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
