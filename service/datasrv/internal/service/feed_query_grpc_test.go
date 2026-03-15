package service

import (
	"context"
	"testing"
	"time"

	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func TestFeedQueryGRPCServer_ListFeedsAndContents(t *testing.T) {
	store := newFakeFeedStore()
	_, _ = store.UpsertFeedSource(context.Background(), dao.FeedSource{
		ID:          "feed-1",
		URL:         "https://example.com/feed.xml",
		DisplayName: "Example",
		Enabled:     true,
	})
	_, _ = store.UpsertFeedContents(context.Background(), "feed-1", []dao.FeedContent{
		{ID: "item-1", FeedSourceID: "feed-1", Identity: "guid-1", Title: "one", PublishedAt: time.Now().UTC().Add(time.Hour)},
		{ID: "item-2", FeedSourceID: "feed-1", Identity: "guid-2", Title: "two", PublishedAt: time.Now().UTC()},
	})

	srv := NewFeedQueryGRPCServer(store)
	feedsResp, err := srv.ListFeeds(context.Background(), &feedsv1.ListFeedSourcesRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListFeeds() error = %v", err)
	}
	if len(feedsResp.GetSources()) != 1 {
		t.Fatalf("sources len = %d, want 1", len(feedsResp.GetSources()))
	}

	contentsResp, err := srv.ListFeedContents(context.Background(), &feedsv1.ListFeedContentsRequest{FeedSourceId: "feed-1", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListFeedContents() error = %v", err)
	}
	if len(contentsResp.GetContents()) != 1 {
		t.Fatalf("contents len = %d, want 1", len(contentsResp.GetContents()))
	}
	if !contentsResp.GetHasNext() {
		t.Fatal("expected has_next for paginated contents")
	}
}

func TestFeedQueryGRPCServer_GetFeedContentValidation(t *testing.T) {
	srv := NewFeedQueryGRPCServer(newFakeFeedStore())
	if _, err := srv.ListFeedContents(context.Background(), &feedsv1.ListFeedContentsRequest{}); err == nil {
		t.Fatal("ListFeedContents() should fail when feed_source_id is missing")
	}
	if _, err := srv.GetFeedContent(context.Background(), &feedsv1.GetFeedContentRequest{}); err == nil {
		t.Fatal("GetFeedContent() should fail when id is missing")
	}
}
