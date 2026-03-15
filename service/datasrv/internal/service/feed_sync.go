package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type FeedFetcher interface {
	Fetch(ctx context.Context, source dao.FeedSource, checkpoint dao.FeedCheckpoint) (FeedFetchResult, error)
}

type FeedFetchResult struct {
	Source       dao.FeedSource
	Contents     []dao.FeedContent
	ETag         string
	LastModified string
	FetchedAt    time.Time
	NotModified  bool
}

type FeedSyncResult struct {
	FeedSourceID string
	Fetched      int32
	Persisted    int32
	Error        string
}

type FeedSyncRunSummary struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Results    []FeedSyncResult
}

type FeedSyncService struct {
	mu      sync.Mutex
	running bool

	cfgMu sync.RWMutex
	cfg   conf.FeedSyncConfig

	store   dao.FeedStore
	fetcher FeedFetcher
}

func NewFeedSyncService(store dao.FeedStore, cfg conf.FeedSyncConfig, fetcher FeedFetcher) *FeedSyncService {
	if fetcher == nil {
		fetcher = NewHTTPFeedFetcher(normalizeFeedSyncConfig(cfg))
	}
	return &FeedSyncService{
		cfg:     normalizeFeedSyncConfig(cfg),
		store:   store,
		fetcher: fetcher,
	}
}

func normalizeFeedSyncConfig(in conf.FeedSyncConfig) conf.FeedSyncConfig {
	if in.IntervalSeconds <= 0 {
		in.IntervalSeconds = 300
	}
	if in.RequestTimeoutSeconds <= 0 {
		in.RequestTimeoutSeconds = 15
	}
	return in
}

func (s *FeedSyncService) GetConfig() conf.FeedSyncConfig {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

func (s *FeedSyncService) UpdateConfig(cfg conf.FeedSyncConfig) conf.FeedSyncConfig {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = normalizeFeedSyncConfig(cfg)
	return s.cfg
}

func (s *FeedSyncService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *FeedSyncService) RunSync(ctx context.Context, onlySourceID string) (FeedSyncRunSummary, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return FeedSyncRunSummary{}, fmt.Errorf("feed sync is already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	cfg := s.GetConfig()
	summary := FeedSyncRunSummary{StartedAt: time.Now()}
	if !cfg.Enabled {
		summary.FinishedAt = time.Now()
		return summary, nil
	}

	if err := s.ensureConfiguredSources(ctx, cfg.Sources); err != nil {
		return summary, err
	}

	sources, err := s.store.ListFeedSources(ctx, dao.FeedSourceFilter{})
	if err != nil {
		return summary, fmt.Errorf("list feed sources: %w", err)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })

	for _, source := range sources {
		if onlySourceID != "" && source.ID != onlySourceID {
			continue
		}
		if !source.Enabled {
			continue
		}
		summary.Results = append(summary.Results, s.syncOneSource(ctx, cfg, source))
	}

	summary.FinishedAt = time.Now()
	return summary, nil
}

func (s *FeedSyncService) ensureConfiguredSources(ctx context.Context, sources []conf.FeedSourceConfig) error {
	for _, sourceCfg := range sources {
		if strings.TrimSpace(sourceCfg.URL) == "" {
			continue
		}
		_, err := s.store.UpsertFeedSource(ctx, dao.FeedSource{
			ID:          normalizeFeedSourceID(sourceCfg.ID, sourceCfg.URL),
			URL:         strings.TrimSpace(sourceCfg.URL),
			DisplayName: strings.TrimSpace(sourceCfg.DisplayName),
			Description: strings.TrimSpace(sourceCfg.Description),
			SiteURL:     strings.TrimSpace(sourceCfg.SiteURL),
			Enabled:     sourceCfg.Enabled,
		})
		if err != nil {
			return fmt.Errorf("seed feed source %q: %w", sourceCfg.URL, err)
		}
	}
	return nil
}

func (s *FeedSyncService) syncOneSource(ctx context.Context, cfg conf.FeedSyncConfig, source dao.FeedSource) FeedSyncResult {
	result := FeedSyncResult{FeedSourceID: source.ID}
	checkpoint, err := s.store.GetFeedCheckpoint(ctx, source.ID)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	fetchResult, err := s.fetcher.Fetch(reqCtx, source, checkpoint)
	if err != nil {
		result.Error = err.Error()
		_ = s.store.SaveFeedCheckpoint(ctx, dao.FeedCheckpoint{
			FeedSourceID:  source.ID,
			LastSyncedAt:  time.Now().UTC(),
			LastSuccessAt: checkpoint.LastSuccessAt,
			LastRunStatus: "failed",
			LastError:     result.Error,
			ETag:          checkpoint.ETag,
			LastModified:  checkpoint.LastModified,
		})
		source.LastSyncedAt = time.Now().UTC()
		source.LastRunStatus = "failed"
		source.LastError = result.Error
		_, _ = s.store.UpsertFeedSource(ctx, source)
		return result
	}

	if fetchResult.FetchedAt.IsZero() {
		fetchResult.FetchedAt = time.Now().UTC()
	}
	if fetchResult.Source.ID == "" {
		fetchResult.Source = source
	}
	fetchResult.Source.ID = source.ID
	fetchResult.Source.URL = source.URL
	fetchResult.Source.Enabled = source.Enabled
	fetchResult.Source.LastSyncedAt = fetchResult.FetchedAt
	fetchResult.Source.LastRunStatus = "success"
	fetchResult.Source.LastError = ""
	fetchResult.Source.ETag = firstNonEmpty(fetchResult.ETag, checkpoint.ETag)
	fetchResult.Source.LastModified = firstNonEmpty(fetchResult.LastModified, checkpoint.LastModified)

	if !fetchResult.NotModified {
		for i := range fetchResult.Contents {
			content := &fetchResult.Contents[i]
			content.FeedSourceID = source.ID
			if content.Identity == "" {
				content.Identity = deriveFeedContentIdentity(*content)
			}
			if content.ID == "" {
				content.ID = makeFeedContentID(source.ID, content.Identity)
			}
			if content.FetchedAt.IsZero() {
				content.FetchedAt = fetchResult.FetchedAt
			}
			result.Fetched++
		}
		persisted, persistErr := s.store.UpsertFeedContents(ctx, source.ID, fetchResult.Contents)
		if persistErr != nil {
			result.Error = persistErr.Error()
			_ = s.store.SaveFeedCheckpoint(ctx, dao.FeedCheckpoint{
				FeedSourceID:  source.ID,
				LastSyncedAt:  fetchResult.FetchedAt,
				LastSuccessAt: checkpoint.LastSuccessAt,
				LastRunStatus: "failed",
				LastError:     result.Error,
				ETag:          checkpoint.ETag,
				LastModified:  checkpoint.LastModified,
			})
			fetchResult.Source.LastRunStatus = "failed"
			fetchResult.Source.LastError = result.Error
			_, _ = s.store.UpsertFeedSource(ctx, fetchResult.Source)
			return result
		}
		result.Persisted = int32(persisted)
	}

	if !fetchResult.NotModified {
		fetchResult.Source.LastSuccessAt = fetchResult.FetchedAt
	}
	_, _ = s.store.UpsertFeedSource(ctx, fetchResult.Source)
	_ = s.store.SaveFeedCheckpoint(ctx, dao.FeedCheckpoint{
		FeedSourceID:  source.ID,
		LastSyncedAt:  fetchResult.FetchedAt,
		LastSuccessAt: chooseTime(fetchResult.FetchedAt, checkpoint.LastSuccessAt, !fetchResult.NotModified),
		LastRunStatus: "success",
		LastError:     "",
		ETag:          firstNonEmpty(fetchResult.ETag, checkpoint.ETag),
		LastModified:  firstNonEmpty(fetchResult.LastModified, checkpoint.LastModified),
	})
	return result
}

func chooseTime(primary, fallback time.Time, usePrimary bool) time.Time {
	if usePrimary {
		return primary
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeFeedSourceID(id, url string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	hash := sha1.Sum([]byte(strings.TrimSpace(url)))
	return "feed-" + hex.EncodeToString(hash[:6])
}

func deriveFeedContentIdentity(content dao.FeedContent) string {
	switch {
	case strings.TrimSpace(content.GUID) != "":
		return strings.TrimSpace(content.GUID)
	case strings.TrimSpace(content.Link) != "":
		return strings.TrimSpace(content.Link)
	default:
		published := content.PublishedAt.UTC().Format(time.RFC3339)
		return strings.TrimSpace(content.Title) + "|" + published
	}
}

func makeFeedContentID(sourceID, identity string) string {
	hash := sha1.Sum([]byte(sourceID + "::" + identity))
	return "item-" + hex.EncodeToString(hash[:8])
}
