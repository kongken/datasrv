package service

import (
	"context"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IssueQueryGRPCServer implements issues.v1.IssueQueryService for user-facing issue queries.
type IssueQueryGRPCServer struct {
	issuesv1.UnimplementedIssueQueryServiceServer
	store        dao.SyncStore
	commentStore IssueCommentStore
}

func NewIssueQueryGRPCServer(store dao.SyncStore, commentStore IssueCommentStore) *IssueQueryGRPCServer {
	return &IssueQueryGRPCServer{store: store, commentStore: commentStore}
}

func (s *IssueQueryGRPCServer) ListIssues(ctx context.Context, req *issuesv1.ListIssuesRequest) (*issuesv1.ListIssuesResponse, error) {
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
	rows, err := s.store.ListIssues(ctx, dao.SyncIssueFilter{
		Repo:   req.GetRepo(),
		State:  req.GetState(),
		Offset: offset,
		Limit:  int(pageSize + 1),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list issues: %v", err)
	}

	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}

	issues := make([]*issuesv1.Issue, 0, len(rows))
	for _, row := range rows {
		issues = append(issues, toProtoIssue(row))
	}

	return &issuesv1.ListIssuesResponse{
		Issues:   issues,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}, nil
}

func (s *IssueQueryGRPCServer) GetIssue(ctx context.Context, req *issuesv1.GetIssueRequest) (*issuesv1.GetIssueResponse, error) {
	if req.GetRepo() == "" {
		return nil, status.Error(codes.InvalidArgument, "repo is required")
	}

	filter := dao.SyncIssueFilter{Repo: req.GetRepo(), Limit: 1}
	switch {
	case req.GetIssueId() > 0:
		filter.IssueID = req.GetIssueId()
	case req.GetNumber() > 0:
		filter.Number = req.GetNumber()
	default:
		return nil, status.Error(codes.InvalidArgument, "either issue_id or number is required")
	}

	rows, err := s.store.ListIssues(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get issue: %v", err)
	}
	if len(rows) == 0 {
		return nil, status.Error(codes.NotFound, "issue not found")
	}

	issue := toProtoIssue(rows[0])
	if s.commentStore != nil && rows[0].Comments > 0 {
		comments, err := s.commentStore.LoadComments(ctx, rows[0].Repo, rows[0].IssueID, rows[0].Number)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load issue comments: %v", err)
		}
		issue.CommentsDetail = toProtoIssueComments(comments)
	}

	return &issuesv1.GetIssueResponse{Issue: issue}, nil
}

func toProtoIssue(in dao.SyncedIssue) *issuesv1.Issue {
	labels := make([]*issuesv1.Label, 0, len(in.Labels))
	for _, name := range in.Labels {
		labels = append(labels, &issuesv1.Label{Name: name})
	}

	assignees := make([]*issuesv1.User, 0, len(in.Assignees))
	for _, login := range in.Assignees {
		assignees = append(assignees, &issuesv1.User{Login: login})
	}

	out := &issuesv1.Issue{
		Id:        in.IssueID,
		Number:    in.Number,
		Title:     in.Title,
		Repo:      in.Repo,
		Body:      in.Body,
		State:     in.State,
		User:      &issuesv1.User{Login: in.Author},
		Labels:    labels,
		Assignees: assignees,
		Comments:  in.Comments,
		HtmlUrl:   in.HTMLURL,
		Locked:    false,
		AiSummary: in.AISummary,
	}
	if !in.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(in.CreatedAt)
	}
	if !in.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(in.UpdatedAt)
	}
	if in.ClosedAt != nil {
		out.ClosedAt = timestamppb.New(*in.ClosedAt)
	}
	return out
}

func toProtoIssueComments(in []dao.IssueComment) []*issuesv1.IssueComment {
	out := make([]*issuesv1.IssueComment, 0, len(in))
	for _, comment := range in {
		item := &issuesv1.IssueComment{
			Id:      comment.ID,
			Body:    comment.Body,
			HtmlUrl: comment.HTMLURL,
			User: &issuesv1.User{
				Login:     comment.UserLogin,
				AvatarUrl: comment.UserAvatarURL,
				HtmlUrl:   comment.UserURL,
			},
		}
		if !comment.CreatedAt.IsZero() {
			item.CreatedAt = timestamppb.New(comment.CreatedAt)
		}
		if !comment.UpdatedAt.IsZero() {
			item.UpdatedAt = timestamppb.New(comment.UpdatedAt)
		}
		out = append(out, item)
	}
	return out
}
