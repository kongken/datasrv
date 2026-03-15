package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type fakeFeedStore struct {
	sources     map[string]dao.FeedSource
	contents    map[string][]dao.FeedContent
	checkpoints map[string]dao.FeedCheckpoint
}

func newFakeFeedStore() *fakeFeedStore {
	return &fakeFeedStore{
		sources:     map[string]dao.FeedSource{},
		contents:    map[string][]dao.FeedContent{},
		checkpoints: map[string]dao.FeedCheckpoint{},
	}
}

func (f *fakeFeedStore) UpsertFeedSource(_ context.Context, source dao.FeedSource) (dao.FeedSource, error) {
	if source.ID == "" {
		source.ID = source.URL
	}
	f.sources[source.ID] = source
	return source, nil
}

func (f *fakeFeedStore) GetFeedSource(_ context.Context, id string) (dao.FeedSource, error) {
	source, ok := f.sources[id]
	if !ok {
		return dao.FeedSource{}, dao.ErrFeedSourceNotFound
	}
	return source, nil
}

func (f *fakeFeedStore) ListFeedSources(_ context.Context, _ dao.FeedSourceFilter) ([]dao.FeedSource, error) {
	out := make([]dao.FeedSource, 0, len(f.sources))
	for _, source := range f.sources {
		out = append(out, source)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeFeedStore) DeleteFeedSource(_ context.Context, id string) error {
	delete(f.sources, id)
	delete(f.contents, id)
	delete(f.checkpoints, id)
	return nil
}

func (f *fakeFeedStore) UpsertFeedContents(_ context.Context, sourceID string, contents []dao.FeedContent) (int, error) {
	f.contents[sourceID] = append([]dao.FeedContent(nil), contents...)
	return len(contents), nil
}

func (f *fakeFeedStore) ListFeedContents(_ context.Context, filter dao.FeedContentFilter) ([]dao.FeedContent, error) {
	contents := append([]dao.FeedContent(nil), f.contents[filter.FeedSourceID]...)
	sort.Slice(contents, func(i, j int) bool {
		if contents[i].PublishedAt.Equal(contents[j].PublishedAt) {
			return contents[i].ID < contents[j].ID
		}
		return contents[i].PublishedAt.After(contents[j].PublishedAt)
	})
	start := filter.Offset
	if start > len(contents) {
		return []dao.FeedContent{}, nil
	}
	end := len(contents)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return contents[start:end], nil
}

func (f *fakeFeedStore) GetFeedContent(_ context.Context, id string) (dao.FeedContent, error) {
	for _, items := range f.contents {
		for _, item := range items {
			if item.ID == id {
				return item, nil
			}
		}
	}
	return dao.FeedContent{}, dao.ErrFeedContentNotFound
}

func (f *fakeFeedStore) GetFeedCheckpoint(_ context.Context, sourceID string) (dao.FeedCheckpoint, error) {
	if cp, ok := f.checkpoints[sourceID]; ok {
		return cp, nil
	}
	return dao.FeedCheckpoint{FeedSourceID: sourceID}, nil
}

func (f *fakeFeedStore) SaveFeedCheckpoint(_ context.Context, checkpoint dao.FeedCheckpoint) error {
	f.checkpoints[checkpoint.FeedSourceID] = checkpoint
	return nil
}

func (f *fakeFeedStore) Close() error { return nil }

type fakeFeedFetcher struct {
	results map[string]FeedFetchResult
	errs    map[string]error
}

func (f *fakeFeedFetcher) Fetch(_ context.Context, source dao.FeedSource, _ dao.FeedCheckpoint) (FeedFetchResult, error) {
	if err := f.errs[source.ID]; err != nil {
		return FeedFetchResult{}, err
	}
	return f.results[source.ID], nil
}

func TestNormalizeFeedSyncConfigDefaults(t *testing.T) {
	cfg := normalizeFeedSyncConfig(conf.FeedSyncConfig{})
	if cfg.IntervalSeconds != 300 || cfg.RequestTimeoutSeconds != 15 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestFeedSyncServiceRunSyncPersistsConfiguredSources(t *testing.T) {
	store := newFakeFeedStore()
	fetcher := &fakeFeedFetcher{
		results: map[string]FeedFetchResult{
			"feed-1": {
				Source: dao.FeedSource{ID: "feed-1", URL: "https://example.com/feed.xml", DisplayName: "Example", Enabled: true},
				Contents: []dao.FeedContent{{
					ID:           "item-1",
					FeedSourceID: "feed-1",
					Identity:     "guid-1",
					Title:        "hello",
					PublishedAt:  time.Now().UTC(),
				}},
				FetchedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewFeedSyncService(store, conf.FeedSyncConfig{
		Enabled: true,
		Sources: []conf.FeedSourceConfig{{
			ID:          "feed-1",
			URL:         "https://example.com/feed.xml",
			DisplayName: "Example",
			Enabled:     true,
		}},
	}, fetcher)

	summary, err := svc.RunSync(context.Background(), "")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(summary.Results))
	}
	if summary.Results[0].Persisted != 1 {
		t.Fatalf("persisted = %d, want 1", summary.Results[0].Persisted)
	}

	contents, err := store.ListFeedContents(context.Background(), dao.FeedContentFilter{FeedSourceID: "feed-1"})
	if err != nil {
		t.Fatalf("ListFeedContents() error = %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(contents))
	}
}

func TestFeedSyncServiceContinuesAfterSourceFailure(t *testing.T) {
	store := newFakeFeedStore()
	fetcher := &fakeFeedFetcher{
		results: map[string]FeedFetchResult{
			"feed-2": {
				Source:    dao.FeedSource{ID: "feed-2", URL: "https://example.com/2.xml", DisplayName: "Two", Enabled: true},
				Contents:  []dao.FeedContent{{ID: "item-2", FeedSourceID: "feed-2", Identity: "guid-2", Title: "second"}},
				FetchedAt: time.Now().UTC(),
			},
		},
		errs: map[string]error{
			"feed-1": errors.New("boom"),
		},
	}
	svc := NewFeedSyncService(store, conf.FeedSyncConfig{
		Enabled: true,
		Sources: []conf.FeedSourceConfig{
			{ID: "feed-1", URL: "https://example.com/1.xml", DisplayName: "One", Enabled: true},
			{ID: "feed-2", URL: "https://example.com/2.xml", DisplayName: "Two", Enabled: true},
		},
	}, fetcher)

	summary, err := svc.RunSync(context.Background(), "")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if len(summary.Results) != 2 {
		t.Fatalf("results len = %d, want 2", len(summary.Results))
	}
	if summary.Results[0].Error == "" {
		t.Fatalf("expected first source error")
	}
	if summary.Results[1].Persisted != 1 {
		t.Fatalf("persisted = %d, want 1", summary.Results[1].Persisted)
	}
}

func TestFeedSyncServiceRunSingleSource(t *testing.T) {
	store := newFakeFeedStore()
	fetcher := &fakeFeedFetcher{
		results: map[string]FeedFetchResult{
			"feed-1": {Source: dao.FeedSource{ID: "feed-1", URL: "https://example.com/1.xml", Enabled: true}, Contents: []dao.FeedContent{{ID: "item-1", Identity: "guid-1"}}},
			"feed-2": {Source: dao.FeedSource{ID: "feed-2", URL: "https://example.com/2.xml", Enabled: true}, Contents: []dao.FeedContent{{ID: "item-2", Identity: "guid-2"}}},
		},
	}
	svc := NewFeedSyncService(store, conf.FeedSyncConfig{
		Enabled: true,
		Sources: []conf.FeedSourceConfig{
			{ID: "feed-1", URL: "https://example.com/1.xml", Enabled: true},
			{ID: "feed-2", URL: "https://example.com/2.xml", Enabled: true},
		},
	}, fetcher)

	summary, err := svc.RunSync(context.Background(), "feed-2")
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(summary.Results))
	}
	if summary.Results[0].FeedSourceID != "feed-2" {
		t.Fatalf("feed source id = %q, want feed-2", summary.Results[0].FeedSourceID)
	}
}
