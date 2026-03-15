package service

import (
	"context"
	"testing"
	"time"

	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestFeedSyncAdminGRPCServer_CreateListAndDeleteSource(t *testing.T) {
	store := newFakeFeedStore()
	syncSvc := NewFeedSyncService(store, conf.FeedSyncConfig{Enabled: true}, &fakeFeedFetcher{results: map[string]FeedFetchResult{}})
	srv := NewFeedSyncAdminGRPCServer(store, syncSvc, &conf.Config{Storage: conf.StorageConfig{Driver: "postgres"}})

	created, err := srv.CreateFeedSource(context.Background(), &feedsv1.CreateFeedSourceRequest{
		Source: &feedsv1.FeedSource{Url: "https://example.com/feed.xml", DisplayName: "Example", Enabled: true},
	})
	if err != nil {
		t.Fatalf("CreateFeedSource() error = %v", err)
	}
	if created.GetId() == "" {
		t.Fatal("expected created source id")
	}

	listed, err := srv.ListFeedSources(context.Background(), &feedsv1.ListFeedSourcesRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListFeedSources() error = %v", err)
	}
	if len(listed.GetSources()) != 1 {
		t.Fatalf("sources len = %d, want 1", len(listed.GetSources()))
	}

	if _, err := srv.DeleteFeedSource(context.Background(), &feedsv1.DeleteFeedSourceRequest{Id: created.GetId()}); err != nil {
		t.Fatalf("DeleteFeedSource() error = %v", err)
	}
}

func TestFeedSyncAdminGRPCServer_SyncAndStatus(t *testing.T) {
	store := newFakeFeedStore()
	_, _ = store.UpsertFeedSource(context.Background(), dao.FeedSource{
		ID:          "feed-1",
		URL:         "https://example.com/feed.xml",
		DisplayName: "Example",
		Enabled:     true,
	})
	syncSvc := NewFeedSyncService(store, conf.FeedSyncConfig{Enabled: true}, &fakeFeedFetcher{
		results: map[string]FeedFetchResult{
			"feed-1": {
				Source:    dao.FeedSource{ID: "feed-1", URL: "https://example.com/feed.xml", DisplayName: "Example", Enabled: true},
				Contents:  []dao.FeedContent{{ID: "item-1", FeedSourceID: "feed-1", Identity: "guid-1", Title: "hello", PublishedAt: time.Now().UTC()}},
				FetchedAt: time.Now().UTC(),
			},
		},
	})
	srv := NewFeedSyncAdminGRPCServer(store, syncSvc, &conf.Config{})

	resp, err := srv.SyncFeeds(context.Background(), &feedsv1.SyncFeedsRequest{FeedSourceId: "feed-1"})
	if err != nil {
		t.Fatalf("SyncFeeds() error = %v", err)
	}
	if len(resp.GetResults()) != 1 {
		t.Fatalf("results len = %d, want 1", len(resp.GetResults()))
	}

	status, err := srv.GetFeedSyncStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetFeedSyncStatus() error = %v", err)
	}
	if len(status.GetStatuses()) != 1 {
		t.Fatalf("statuses len = %d, want 1", len(status.GetStatuses()))
	}
}
