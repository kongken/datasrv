package service

import (
	"context"

	corelog "butterfly.orx.me/core/log"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PRReviewQueryGRPCServer implements issues.v1.PRReviewQueryService.
type PRReviewQueryGRPCServer struct {
	issuesv1.UnimplementedPRReviewQueryServiceServer
	prStore dao.PRReviewStore
}

func NewPRReviewQueryGRPCServer(prStore dao.PRReviewStore) *PRReviewQueryGRPCServer {
	return &PRReviewQueryGRPCServer{prStore: prStore}
}

func (s *PRReviewQueryGRPCServer) ListPRReviews(ctx context.Context, req *issuesv1.ListPRReviewsRequest) (*issuesv1.ListPRReviewsResponse, error) {
	logger := corelog.FromContext(ctx).With("component", "datasrv.pr_review_query", "operation", "list_pr_reviews")

	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := int((page - 1) * pageSize)
	rows, err := s.prStore.ListPRReviews(ctx, dao.PRReviewFilter{
		Repo:   req.GetRepo(),
		Offset: offset,
		Limit:  int(pageSize + 1),
	})
	if err != nil {
		logger.Error("list pr reviews failed", "error", err)
		return nil, status.Errorf(codes.Internal, "list pr reviews: %v", err)
	}

	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}

	reviews := make([]*issuesv1.PRReview, 0, len(rows))
	for _, row := range rows {
		reviews = append(reviews, toProtoPRReview(row))
	}

	return &issuesv1.ListPRReviewsResponse{
		Reviews:  reviews,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}, nil
}

func (s *PRReviewQueryGRPCServer) GetPRReview(ctx context.Context, req *issuesv1.GetPRReviewRequest) (*issuesv1.GetPRReviewResponse, error) {
	logger := corelog.FromContext(ctx).With("component", "datasrv.pr_review_query", "operation", "get_pr_review")

	if req.GetRepo() == "" || req.GetNumber() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "repo and number are required")
	}

	review, err := s.prStore.GetPRReview(ctx, req.GetRepo(), req.GetNumber())
	if err != nil {
		if err == dao.ErrPRReviewNotFound {
			return nil, status.Error(codes.NotFound, "pr review not found")
		}
		logger.Error("get pr review failed", "error", err)
		return nil, status.Errorf(codes.Internal, "get pr review: %v", err)
	}

	return &issuesv1.GetPRReviewResponse{
		Review: toProtoPRReview(review),
	}, nil
}

func toProtoPRReview(in dao.PRReview) *issuesv1.PRReview {
	out := &issuesv1.PRReview{
		Repo:          in.Repo,
		IssueId:       in.IssueID,
		Number:        in.Number,
		ReviewSummary: in.ReviewSummary,
		RiskAreas:     in.RiskAreas,
		Suggestions:   in.Suggestions,
		RawDiffSize:   int32(in.RawDiffSize),
		ModelUsed:     in.ModelUsed,
	}
	if !in.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(in.CreatedAt)
	}
	if !in.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(in.UpdatedAt)
	}
	return out
}
