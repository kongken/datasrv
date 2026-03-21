package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	corelog "butterfly.orx.me/core/log"
	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// IssueQueryGRPCServer implements issues.v1.IssueQueryService for user-facing issue queries.
type IssueQueryGRPCServer struct {
	issuesv1.UnimplementedIssueQueryServiceServer
	store        dao.SyncStore
	commentStore IssueCommentStore
	cacheTTL     time.Duration
	mu           sync.RWMutex
	listCache    map[string]cachedListIssuesResponse
	detailCache  map[string]cachedGetIssueResponse
}

type cachedListIssuesResponse struct {
	resp      *issuesv1.ListIssuesResponse
	expiresAt time.Time
}

type cachedGetIssueResponse struct {
	resp      *issuesv1.GetIssueResponse
	expiresAt time.Time
}

const defaultIssueQueryCacheTTL = 15 * time.Second

func NewIssueQueryGRPCServer(store dao.SyncStore, commentStore IssueCommentStore) *IssueQueryGRPCServer {
	return &IssueQueryGRPCServer{
		store:        store,
		commentStore: commentStore,
		cacheTTL:     defaultIssueQueryCacheTTL,
		listCache:    make(map[string]cachedListIssuesResponse),
		detailCache:  make(map[string]cachedGetIssueResponse),
	}
}

func (s *IssueQueryGRPCServer) ListIssues(ctx context.Context, req *issuesv1.ListIssuesRequest) (*issuesv1.ListIssuesResponse, error) {
	logger := issueQueryLogger(ctx).With(
		"operation", "list_issues",
		"repo", req.GetRepo(),
		"state", req.GetState(),
	)
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
	logger = logger.With("page", page, "page_size", pageSize)

	cacheKey := buildListIssuesCacheKey(req.GetRepo(), req.GetState(), page, pageSize)
	if cached, ok := s.getCachedListIssues(cacheKey, logger); ok {
		logger.Debug("issue query list cache hit")
		return cached, nil
	}
	logger.Debug("issue query list cache miss")

	offset := int((page - 1) * pageSize)
	logger.Debug("issue query list loading from store", "offset", offset, "limit", pageSize+1)
	rows, err := s.store.ListIssues(ctx, dao.SyncIssueFilter{
		Repo:   req.GetRepo(),
		State:  req.GetState(),
		Offset: offset,
		Limit:  int(pageSize + 1),
	})
	if err != nil {
		logger.Error("issue query list store query failed", "error", err)
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

	resp := &issuesv1.ListIssuesResponse{
		Issues:   issues,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}
	s.setCachedListIssues(cacheKey, resp)
	logger.Debug("issue query list completed", "issue_count", len(resp.GetIssues()), "has_next", hasNext)
	return resp, nil
}

func (s *IssueQueryGRPCServer) GetIssue(ctx context.Context, req *issuesv1.GetIssueRequest) (*issuesv1.GetIssueResponse, error) {
	logger := issueQueryLogger(ctx).With("operation", "get_issue", "repo", req.GetRepo())
	cacheKey := buildGetIssueCacheKey(req.GetRepo(), req.GetIssueId(), req.GetNumber())
	if cached, ok := s.getCachedGetIssue(cacheKey, logger); ok {
		logger.Debug("issue query detail cache hit", "issue_id", req.GetIssueId(), "number", req.GetNumber())
		return cached, nil
	}
	logger.Debug("issue query detail cache miss", "issue_id", req.GetIssueId(), "number", req.GetNumber())

	filter := dao.SyncIssueFilter{Repo: req.GetRepo(), Limit: 1}
	switch {
	case req.GetIssueId() > 0:
		filter.IssueID = req.GetIssueId()
		logger = logger.With("issue_id", req.GetIssueId())
	case req.GetNumber() > 0:
		if req.GetRepo() == "" {
			return nil, status.Error(codes.InvalidArgument, "repo is required when querying by number")
		}
		filter.Number = req.GetNumber()
		logger = logger.With("number", req.GetNumber())
	default:
		return nil, status.Error(codes.InvalidArgument, "either issue_id or number is required")
	}

	logger.Debug("issue query detail loading from store")
	rows, err := s.store.ListIssues(ctx, filter)
	if err != nil {
		logger.Error("issue query detail store query failed", "error", err)
		return nil, status.Errorf(codes.Internal, "get issue: %v", err)
	}
	if len(rows) == 0 {
		logger.Info("issue query detail not found")
		return nil, status.Error(codes.NotFound, "issue not found")
	}

	issue := toProtoIssue(rows[0])
	if s.commentStore != nil && rows[0].Comments > 0 {
		logger.Debug("issue query detail loading comments", "comment_count", rows[0].Comments)
		comments, err := s.commentStore.LoadComments(ctx, rows[0].Repo, rows[0].IssueID, rows[0].Number)
		if err == nil {
			issue.CommentsDetail = toProtoIssueComments(comments)
			logger.Debug("issue query detail comments loaded", "loaded_count", len(issue.GetCommentsDetail()))
		} else {
			logger.Warn("issue query detail comments load failed; continue without comments_detail", "error", err)
		}
	}

	resp := &issuesv1.GetIssueResponse{Issue: issue}
	s.setCachedGetIssue(cacheKey, resp)
	logger.Debug("issue query detail completed")
	return resp, nil
}

func buildListIssuesCacheKey(repo, state string, page, pageSize int32) string {
	return fmt.Sprintf("repo=%s|state=%s|page=%d|page_size=%d", repo, state, page, pageSize)
}

func buildGetIssueCacheKey(repo string, issueID int64, number int32) string {
	return fmt.Sprintf("repo=%s|issue_id=%d|number=%d", repo, issueID, number)
}

func (s *IssueQueryGRPCServer) getCachedListIssues(key string, logger *slog.Logger) (*issuesv1.ListIssuesResponse, bool) {
	if s.cacheTTL <= 0 {
		logger.Debug("issue query list cache disabled")
		return nil, false
	}
	now := time.Now().UTC()
	s.mu.RLock()
	item, ok := s.listCache[key]
	if !ok || now.After(item.expiresAt) {
		s.mu.RUnlock()
		if ok {
			s.mu.Lock()
			delete(s.listCache, key)
			s.mu.Unlock()
			logger.Debug("issue query list cache entry expired")
		}
		return nil, false
	}
	resp := proto.Clone(item.resp).(*issuesv1.ListIssuesResponse)
	s.mu.RUnlock()
	return resp, true
}

func (s *IssueQueryGRPCServer) setCachedListIssues(key string, resp *issuesv1.ListIssuesResponse) {
	if s.cacheTTL <= 0 {
		return
	}
	s.mu.Lock()
	s.listCache[key] = cachedListIssuesResponse{
		resp:      proto.Clone(resp).(*issuesv1.ListIssuesResponse),
		expiresAt: time.Now().UTC().Add(s.cacheTTL),
	}
	s.mu.Unlock()
}

func (s *IssueQueryGRPCServer) getCachedGetIssue(key string, logger *slog.Logger) (*issuesv1.GetIssueResponse, bool) {
	if s.cacheTTL <= 0 {
		logger.Debug("issue query detail cache disabled")
		return nil, false
	}
	now := time.Now().UTC()
	s.mu.RLock()
	item, ok := s.detailCache[key]
	if !ok || now.After(item.expiresAt) {
		s.mu.RUnlock()
		if ok {
			s.mu.Lock()
			delete(s.detailCache, key)
			s.mu.Unlock()
			logger.Debug("issue query detail cache entry expired")
		}
		return nil, false
	}
	resp := proto.Clone(item.resp).(*issuesv1.GetIssueResponse)
	s.mu.RUnlock()
	return resp, true
}

func (s *IssueQueryGRPCServer) setCachedGetIssue(key string, resp *issuesv1.GetIssueResponse) {
	if s.cacheTTL <= 0 {
		return
	}
	s.mu.Lock()
	s.detailCache[key] = cachedGetIssueResponse{
		resp:      proto.Clone(resp).(*issuesv1.GetIssueResponse),
		expiresAt: time.Now().UTC().Add(s.cacheTTL),
	}
	s.mu.Unlock()
}

func issueQueryLogger(ctx context.Context) *slog.Logger {
	return corelog.FromContext(ctx).With("component", "datasrv.issue_query")
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
