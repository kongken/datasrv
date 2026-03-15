package service

import (
	"context"
	"time"

	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FeedQueryGRPCServer struct {
	feedsv1.UnimplementedFeedQueryServiceServer
	store dao.FeedStore
}

func NewFeedQueryGRPCServer(store dao.FeedStore) *FeedQueryGRPCServer {
	return &FeedQueryGRPCServer{store: store}
}

func (s *FeedQueryGRPCServer) ListFeeds(ctx context.Context, req *feedsv1.ListFeedSourcesRequest) (*feedsv1.ListFeedSourcesResponse, error) {
	page, pageSize, offset := normalizePagination(req.GetPage(), req.GetPageSize())
	rows, err := s.store.ListFeedSources(ctx, dao.FeedSourceFilter{Offset: offset, Limit: int(pageSize + 1)})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list feeds: %v", err)
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

func (s *FeedQueryGRPCServer) ListFeedContents(ctx context.Context, req *feedsv1.ListFeedContentsRequest) (*feedsv1.ListFeedContentsResponse, error) {
	if req.GetFeedSourceId() == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_source_id is required")
	}
	if _, err := s.store.GetFeedSource(ctx, req.GetFeedSourceId()); err != nil {
		if err == dao.ErrFeedSourceNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get feed source: %v", err)
	}
	page, pageSize, offset := normalizePagination(req.GetPage(), req.GetPageSize())
	rows, err := s.store.ListFeedContents(ctx, dao.FeedContentFilter{
		FeedSourceID: req.GetFeedSourceId(),
		Offset:       offset,
		Limit:        int(pageSize + 1),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list feed contents: %v", err)
	}
	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}
	contents := make([]*feedsv1.FeedContent, 0, len(rows))
	for _, row := range rows {
		contents = append(contents, toProtoFeedContent(row))
	}
	return &feedsv1.ListFeedContentsResponse{Contents: contents, Page: page, PageSize: pageSize, HasNext: hasNext}, nil
}

func (s *FeedQueryGRPCServer) GetFeedContent(ctx context.Context, req *feedsv1.GetFeedContentRequest) (*feedsv1.GetFeedContentResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	content, err := s.store.GetFeedContent(ctx, req.GetId())
	if err != nil {
		if err == dao.ErrFeedContentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get feed content: %v", err)
	}
	source, err := s.store.GetFeedSource(ctx, content.FeedSourceID)
	if err != nil {
		if err == dao.ErrFeedSourceNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get feed source: %v", err)
	}
	return &feedsv1.GetFeedContentResponse{
		Content: toProtoFeedContent(content),
		Source:  toProtoFeedSource(source),
	}, nil
}

func normalizePagination(page, pageSize int32) (int32, int32, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, int((page - 1) * pageSize)
}

func toProtoFeedSource(source dao.FeedSource) *feedsv1.FeedSource {
	return &feedsv1.FeedSource{
		Id:            source.ID,
		Url:           source.URL,
		DisplayName:   source.DisplayName,
		Description:   source.Description,
		SiteUrl:       source.SiteURL,
		Enabled:       source.Enabled,
		Etag:          source.ETag,
		LastModified:  source.LastModified,
		LastSyncedAt:  maybeTimestamp(source.LastSyncedAt),
		LastSuccessAt: maybeTimestamp(source.LastSuccessAt),
		LastRunStatus: source.LastRunStatus,
		LastError:     source.LastError,
		CreatedAt:     maybeTimestamp(source.CreatedAt),
		UpdatedAt:     maybeTimestamp(source.UpdatedAt),
	}
}

func toProtoFeedContent(content dao.FeedContent) *feedsv1.FeedContent {
	return &feedsv1.FeedContent{
		Id:           content.ID,
		FeedSourceId: content.FeedSourceID,
		Identity:     content.Identity,
		Guid:         content.GUID,
		Title:        content.Title,
		Summary:      content.Summary,
		Content:      content.Content,
		Link:         content.Link,
		Author:       content.Author,
		Categories:   content.Categories,
		PublishedAt:  maybeTimestamp(content.PublishedAt),
		UpdatedAt:    maybeTimestamp(content.UpdatedAt),
		FetchedAt:    maybeTimestamp(content.FetchedAt),
	}
}

func maybeTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
