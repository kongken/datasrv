package service

import (
	"context"
	"sync"

	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FeedSyncAdminGRPCServer struct {
	feedsv1.UnimplementedFeedSyncAdminServiceServer

	store   dao.FeedStore
	syncSvc *FeedSyncService
	cfg     *conf.Config

	statusMu sync.RWMutex
	lastRun  FeedSyncRunSummary
}

func NewFeedSyncAdminGRPCServer(store dao.FeedStore, syncSvc *FeedSyncService, cfg *conf.Config) *FeedSyncAdminGRPCServer {
	return &FeedSyncAdminGRPCServer{store: store, syncSvc: syncSvc, cfg: cfg}
}

func (s *FeedSyncAdminGRPCServer) ListFeedSources(ctx context.Context, req *feedsv1.ListFeedSourcesRequest) (*feedsv1.ListFeedSourcesResponse, error) {
	page, pageSize, offset := normalizePagination(req.GetPage(), req.GetPageSize())
	rows, err := s.store.ListFeedSources(ctx, dao.FeedSourceFilter{Offset: offset, Limit: int(pageSize + 1)})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list feed sources: %v", err)
	}
	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}
	sources := make([]*feedsv1.FeedSource, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, toProtoFeedSource(row))
	}
	return &feedsv1.ListFeedSourcesResponse{Sources: sources, Page: page, PageSize: pageSize, HasNext: hasNext}, nil
}

func (s *FeedSyncAdminGRPCServer) GetFeedSource(ctx context.Context, req *feedsv1.GetFeedSourceRequest) (*feedsv1.FeedSource, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	source, err := s.store.GetFeedSource(ctx, req.GetId())
	if err != nil {
		if err == dao.ErrFeedSourceNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get feed source: %v", err)
	}
	return toProtoFeedSource(source), nil
}

func (s *FeedSyncAdminGRPCServer) CreateFeedSource(ctx context.Context, req *feedsv1.CreateFeedSourceRequest) (*feedsv1.FeedSource, error) {
	source, err := upsertFeedSourceRequest(ctx, s.store, req.GetSource(), false)
	if err != nil {
		return nil, err
	}
	return toProtoFeedSource(source), nil
}

func (s *FeedSyncAdminGRPCServer) UpdateFeedSource(ctx context.Context, req *feedsv1.UpdateFeedSourceRequest) (*feedsv1.FeedSource, error) {
	source, err := upsertFeedSourceRequest(ctx, s.store, req.GetSource(), true)
	if err != nil {
		return nil, err
	}
	return toProtoFeedSource(source), nil
}

func (s *FeedSyncAdminGRPCServer) DeleteFeedSource(ctx context.Context, req *feedsv1.DeleteFeedSourceRequest) (*feedsv1.DeleteFeedSourceResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.store.DeleteFeedSource(ctx, req.GetId()); err != nil {
		if err == dao.ErrFeedSourceNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "delete feed source: %v", err)
	}
	return &feedsv1.DeleteFeedSourceResponse{Id: req.GetId()}, nil
}

func (s *FeedSyncAdminGRPCServer) SyncFeeds(ctx context.Context, req *feedsv1.SyncFeedsRequest) (*feedsv1.SyncFeedsResponse, error) {
	if req.GetFeedSourceId() != "" {
		if _, err := s.store.GetFeedSource(ctx, req.GetFeedSourceId()); err != nil {
			if err == dao.ErrFeedSourceNotFound {
				return nil, status.Error(codes.NotFound, err.Error())
			}
			return nil, status.Errorf(codes.Internal, "get feed source: %v", err)
		}
	}
	summary, err := s.syncSvc.RunSync(ctx, req.GetFeedSourceId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sync feeds: %v", err)
	}
	s.statusMu.Lock()
	s.lastRun = summary
	s.statusMu.Unlock()

	results := make([]*feedsv1.FeedSyncResult, 0, len(summary.Results))
	for _, result := range summary.Results {
		results = append(results, &feedsv1.FeedSyncResult{
			FeedSourceId: result.FeedSourceID,
			Fetched:      result.Fetched,
			Persisted:    result.Persisted,
			Error:        result.Error,
		})
	}
	return &feedsv1.SyncFeedsResponse{
		StartedAt:  timestamppb.New(summary.StartedAt),
		FinishedAt: timestamppb.New(summary.FinishedAt),
		Results:    results,
	}, nil
}

func (s *FeedSyncAdminGRPCServer) GetFeedSyncStatus(ctx context.Context, _ *emptypb.Empty) (*feedsv1.GetFeedSyncStatusResponse, error) {
	s.statusMu.RLock()
	lastRun := s.lastRun
	s.statusMu.RUnlock()

	sources, err := s.store.ListFeedSources(ctx, dao.FeedSourceFilter{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list feed sources: %v", err)
	}

	statuses := make([]*feedsv1.FeedSyncStatus, 0, len(sources))
	for _, source := range sources {
		statuses = append(statuses, &feedsv1.FeedSyncStatus{
			FeedSourceId:  source.ID,
			LastRunStatus: source.LastRunStatus,
			LastError:     source.LastError,
			Etag:          source.ETag,
			LastModified:  source.LastModified,
			LastSyncedAt:  maybeTimestamp(source.LastSyncedAt),
			LastSuccessAt: maybeTimestamp(source.LastSuccessAt),
		})
	}

	results := make([]*feedsv1.FeedSyncResult, 0, len(lastRun.Results))
	for _, result := range lastRun.Results {
		results = append(results, &feedsv1.FeedSyncResult{
			FeedSourceId: result.FeedSourceID,
			Fetched:      result.Fetched,
			Persisted:    result.Persisted,
			Error:        result.Error,
		})
	}

	resp := &feedsv1.GetFeedSyncStatusResponse{
		Running:     s.syncSvc.IsRunning(),
		LastResults: results,
		Statuses:    statuses,
	}
	if !lastRun.StartedAt.IsZero() {
		resp.LastStartedAt = timestamppb.New(lastRun.StartedAt)
	}
	if !lastRun.FinishedAt.IsZero() {
		resp.LastFinishedAt = timestamppb.New(lastRun.FinishedAt)
	}
	return resp, nil
}

func upsertFeedSourceRequest(ctx context.Context, store dao.FeedStore, in *feedsv1.FeedSource, requireID bool) (dao.FeedSource, error) {
	if in == nil {
		return dao.FeedSource{}, status.Error(codes.InvalidArgument, "source is required")
	}
	if in.GetUrl() == "" {
		return dao.FeedSource{}, status.Error(codes.InvalidArgument, "source.url is required")
	}
	if requireID && in.GetId() == "" {
		return dao.FeedSource{}, status.Error(codes.InvalidArgument, "source.id is required")
	}
	source := dao.FeedSource{
		ID:          normalizeFeedSourceID(in.GetId(), in.GetUrl()),
		URL:         in.GetUrl(),
		DisplayName: in.GetDisplayName(),
		Description: in.GetDescription(),
		SiteURL:     in.GetSiteUrl(),
		Enabled:     in.GetEnabled(),
	}
	if requireID {
		existing, err := store.GetFeedSource(ctx, source.ID)
		if err != nil {
			if err == dao.ErrFeedSourceNotFound {
				return dao.FeedSource{}, status.Error(codes.NotFound, err.Error())
			}
			return dao.FeedSource{}, status.Errorf(codes.Internal, "get feed source: %v", err)
		}
		source.CreatedAt = existing.CreatedAt
		source.ETag = existing.ETag
		source.LastModified = existing.LastModified
		source.LastSyncedAt = existing.LastSyncedAt
		source.LastSuccessAt = existing.LastSuccessAt
		source.LastRunStatus = existing.LastRunStatus
		source.LastError = existing.LastError
	}
	out, err := store.UpsertFeedSource(ctx, source)
	if err != nil {
		return dao.FeedSource{}, status.Errorf(codes.Internal, "upsert feed source: %v", err)
	}
	return out, nil
}
