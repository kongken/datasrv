package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	blogv1 "github.com/kongken/datasrv/pkg/proto/blog/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var blogIDSeed uint64

type BlogQueryGRPCServer struct {
	blogv1.UnimplementedBlogQueryServiceServer
	store dao.BlogStore
}

type BlogAdminGRPCServer struct {
	blogv1.UnimplementedBlogAdminServiceServer
	store dao.BlogStore
}

func NewBlogQueryGRPCServer(store dao.BlogStore) *BlogQueryGRPCServer {
	return &BlogQueryGRPCServer{store: store}
}

func NewBlogAdminGRPCServer(store dao.BlogStore) *BlogAdminGRPCServer {
	return &BlogAdminGRPCServer{store: store}
}

func (s *BlogQueryGRPCServer) ListPosts(ctx context.Context, req *blogv1.ListBlogPostsRequest) (*blogv1.ListBlogPostsResponse, error) {
	page, pageSize, offset := normalizePagination(req.GetPage(), req.GetPageSize())
	statusFilter := strings.TrimSpace(req.GetStatus())
	if statusFilter == "" {
		statusFilter = "published"
	}
	rows, err := s.store.ListBlogPosts(ctx, dao.BlogPostFilter{
		Status: statusFilter,
		Tag:    req.GetTag(),
		Query:  req.GetQuery(),
		Offset: offset,
		Limit:  int(pageSize + 1),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list blog posts: %v", err)
	}

	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}

	posts := make([]*blogv1.BlogPost, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, toProtoBlogPost(row))
	}

	return &blogv1.ListBlogPostsResponse{
		Posts:    posts,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}, nil
}

func (s *BlogQueryGRPCServer) GetPost(ctx context.Context, req *blogv1.GetBlogPostRequest) (*blogv1.GetBlogPostResponse, error) {
	if strings.TrimSpace(req.GetSlug()) == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	post, err := s.store.GetBlogPostBySlug(ctx, strings.TrimSpace(req.GetSlug()))
	if err != nil {
		if err == dao.ErrBlogPostNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get blog post: %v", err)
	}
	if !isPublicPost(post.Status) {
		return nil, status.Error(codes.NotFound, dao.ErrBlogPostNotFound.Error())
	}
	return &blogv1.GetBlogPostResponse{Post: toProtoBlogPost(post)}, nil
}

func (s *BlogQueryGRPCServer) ListComments(ctx context.Context, req *blogv1.ListBlogCommentsRequest) (*blogv1.ListBlogCommentsResponse, error) {
	post, err := s.loadPublicPostBySlug(ctx, req.GetPostSlug())
	if err != nil {
		return nil, err
	}

	page, pageSize, offset := normalizePagination(req.GetPage(), req.GetPageSize())
	statusFilter := req.GetStatus()
	if strings.TrimSpace(statusFilter) == "" {
		statusFilter = "approved"
	}
	rows, err := s.store.ListBlogComments(ctx, dao.BlogCommentFilter{
		PostID: post.ID,
		Status: statusFilter,
		Offset: offset,
		Limit:  int(pageSize + 1),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list blog comments: %v", err)
	}

	hasNext := false
	if len(rows) > int(pageSize) {
		hasNext = true
		rows = rows[:pageSize]
	}
	comments := make([]*blogv1.BlogComment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, toProtoBlogComment(row))
	}
	return &blogv1.ListBlogCommentsResponse{
		Comments: comments,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}, nil
}

func (s *BlogQueryGRPCServer) CreateComment(ctx context.Context, req *blogv1.CreateBlogCommentRequest) (*blogv1.BlogComment, error) {
	post, err := s.loadPublicPostBySlug(ctx, req.GetPostSlug())
	if err != nil {
		return nil, err
	}
	comment, err := newBlogCommentModel(post, req.GetComment(), false)
	if err != nil {
		return nil, err
	}
	created, err := s.store.CreateBlogComment(ctx, comment)
	if err != nil {
		if err == dao.ErrBlogCommentPostAbsent {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create blog comment: %v", err)
	}
	return toProtoBlogComment(created), nil
}

func (s *BlogAdminGRPCServer) CreatePost(ctx context.Context, req *blogv1.CreateBlogPostRequest) (*blogv1.BlogPost, error) {
	post, err := newBlogPostModel(req.GetPost(), false)
	if err != nil {
		return nil, err
	}
	created, err := s.store.CreateBlogPost(ctx, post)
	if err != nil {
		if err == dao.ErrBlogPostSlugConflict {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create blog post: %v", err)
	}
	return toProtoBlogPost(created), nil
}

func (s *BlogAdminGRPCServer) UpdatePost(ctx context.Context, req *blogv1.UpdateBlogPostRequest) (*blogv1.BlogPost, error) {
	post, err := newBlogPostModel(req.GetPost(), true)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateBlogPost(ctx, post)
	if err != nil {
		switch err {
		case dao.ErrBlogPostNotFound:
			return nil, status.Error(codes.NotFound, err.Error())
		case dao.ErrBlogPostSlugConflict:
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "update blog post: %v", err)
		}
	}
	return toProtoBlogPost(updated), nil
}

func (s *BlogAdminGRPCServer) DeletePost(ctx context.Context, req *blogv1.DeleteBlogPostRequest) (*blogv1.DeleteBlogPostResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.store.DeleteBlogPost(ctx, strings.TrimSpace(req.GetId())); err != nil {
		if err == dao.ErrBlogPostNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "delete blog post: %v", err)
	}
	return &blogv1.DeleteBlogPostResponse{Id: strings.TrimSpace(req.GetId())}, nil
}

func (s *BlogAdminGRPCServer) GetComment(ctx context.Context, req *blogv1.GetBlogCommentRequest) (*blogv1.GetBlogCommentResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	comment, err := s.store.GetBlogComment(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		if err == dao.ErrBlogCommentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "get blog comment: %v", err)
	}
	return &blogv1.GetBlogCommentResponse{Comment: toProtoBlogComment(comment)}, nil
}

func (s *BlogAdminGRPCServer) UpdateComment(ctx context.Context, req *blogv1.UpdateBlogCommentRequest) (*blogv1.BlogComment, error) {
	comment, err := newBlogCommentModel(dao.BlogPost{}, req.GetComment(), true)
	if err != nil {
		return nil, err
	}
	updated, err := s.store.UpdateBlogComment(ctx, comment)
	if err != nil {
		switch err {
		case dao.ErrBlogCommentNotFound, dao.ErrBlogCommentPostAbsent:
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "update blog comment: %v", err)
		}
	}
	return toProtoBlogComment(updated), nil
}

func (s *BlogAdminGRPCServer) DeleteComment(ctx context.Context, req *blogv1.DeleteBlogCommentRequest) (*blogv1.DeleteBlogCommentResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.store.DeleteBlogComment(ctx, strings.TrimSpace(req.GetId())); err != nil {
		if err == dao.ErrBlogCommentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "delete blog comment: %v", err)
	}
	return &blogv1.DeleteBlogCommentResponse{Id: strings.TrimSpace(req.GetId())}, nil
}

func (s *BlogQueryGRPCServer) loadPublicPostBySlug(ctx context.Context, slug string) (dao.BlogPost, error) {
	if strings.TrimSpace(slug) == "" {
		return dao.BlogPost{}, status.Error(codes.InvalidArgument, "post_slug is required")
	}
	post, err := s.store.GetBlogPostBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		if err == dao.ErrBlogPostNotFound {
			return dao.BlogPost{}, status.Error(codes.NotFound, err.Error())
		}
		return dao.BlogPost{}, status.Errorf(codes.Internal, "get blog post: %v", err)
	}
	if !isPublicPost(post.Status) {
		return dao.BlogPost{}, status.Error(codes.NotFound, dao.ErrBlogPostNotFound.Error())
	}
	return post, nil
}

func newBlogPostModel(in *blogv1.BlogPost, requireID bool) (dao.BlogPost, error) {
	if in == nil {
		return dao.BlogPost{}, status.Error(codes.InvalidArgument, "post is required")
	}
	id := strings.TrimSpace(in.GetId())
	if requireID && id == "" {
		return dao.BlogPost{}, status.Error(codes.InvalidArgument, "post.id is required")
	}
	if !requireID && id == "" {
		id = nextBlogID("post")
	}

	title := strings.TrimSpace(in.GetTitle())
	if title == "" {
		return dao.BlogPost{}, status.Error(codes.InvalidArgument, "post.title is required")
	}
	slug := strings.TrimSpace(in.GetSlug())
	if slug == "" {
		return dao.BlogPost{}, status.Error(codes.InvalidArgument, "post.slug is required")
	}

	statusValue := strings.TrimSpace(in.GetStatus())
	if statusValue == "" {
		statusValue = "draft"
	}
	publishedAt := time.Time{}
	if in.GetPublishedAt() != nil {
		publishedAt = in.GetPublishedAt().AsTime()
	}
	if publishedAt.IsZero() && strings.EqualFold(statusValue, "published") {
		publishedAt = time.Now().UTC()
	}

	return dao.BlogPost{
		ID:          id,
		Title:       title,
		Slug:        slug,
		Summary:     strings.TrimSpace(in.GetSummary()),
		Content:     in.GetContent(),
		Tags:        append([]string(nil), in.GetTags()...),
		Status:      statusValue,
		PublishedAt: publishedAt,
	}, nil
}

func newBlogCommentModel(post dao.BlogPost, in *blogv1.BlogComment, requireID bool) (dao.BlogComment, error) {
	if in == nil {
		return dao.BlogComment{}, status.Error(codes.InvalidArgument, "comment is required")
	}
	id := strings.TrimSpace(in.GetId())
	if requireID && id == "" {
		return dao.BlogComment{}, status.Error(codes.InvalidArgument, "comment.id is required")
	}
	if !requireID && id == "" {
		id = nextBlogID("comment")
	}

	authorName := strings.TrimSpace(in.GetAuthorName())
	if authorName == "" {
		return dao.BlogComment{}, status.Error(codes.InvalidArgument, "comment.author_name is required")
	}
	content := strings.TrimSpace(in.GetContent())
	if content == "" {
		return dao.BlogComment{}, status.Error(codes.InvalidArgument, "comment.content is required")
	}

	statusValue := strings.TrimSpace(in.GetStatus())
	if statusValue == "" {
		if requireID {
			statusValue = "approved"
		} else {
			statusValue = "pending"
		}
	}

	comment := dao.BlogComment{
		ID:          id,
		PostID:      strings.TrimSpace(in.GetPostId()),
		PostSlug:    strings.TrimSpace(in.GetPostSlug()),
		AuthorName:  authorName,
		AuthorEmail: strings.TrimSpace(in.GetAuthorEmail()),
		Content:     content,
		Status:      statusValue,
	}
	if !requireID {
		comment.PostID = post.ID
		comment.PostSlug = post.Slug
	}
	return comment, nil
}

func nextBlogID(prefix string) string {
	value := atomic.AddUint64(&blogIDSeed, 1)
	return fmt.Sprintf("%s-%d", prefix, value)
}

func isPublicPost(statusValue string) bool {
	return strings.EqualFold(strings.TrimSpace(statusValue), "published")
}

func toProtoBlogPost(post dao.BlogPost) *blogv1.BlogPost {
	return &blogv1.BlogPost{
		Id:           post.ID,
		Title:        post.Title,
		Slug:         post.Slug,
		Summary:      post.Summary,
		Content:      post.Content,
		Tags:         append([]string(nil), post.Tags...),
		Status:       post.Status,
		CommentCount: post.CommentCount,
		CreatedAt:    maybeTimestamp(post.CreatedAt),
		UpdatedAt:    maybeTimestamp(post.UpdatedAt),
		PublishedAt:  maybeTimestamp(post.PublishedAt),
	}
}

func toProtoBlogComment(comment dao.BlogComment) *blogv1.BlogComment {
	return &blogv1.BlogComment{
		Id:          comment.ID,
		PostId:      comment.PostID,
		PostSlug:    comment.PostSlug,
		AuthorName:  comment.AuthorName,
		AuthorEmail: comment.AuthorEmail,
		Content:     comment.Content,
		Status:      comment.Status,
		CreatedAt:   maybeTimestamp(comment.CreatedAt),
		UpdatedAt:   maybeTimestamp(comment.UpdatedAt),
	}
}
