package dao

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBlogPostNotFound      = errors.New("blog post not found")
	ErrBlogCommentNotFound   = errors.New("blog comment not found")
	ErrBlogPostSlugConflict  = errors.New("blog post slug already exists")
	ErrBlogCommentPostAbsent = errors.New("blog comment post not found")
)

type BlogPost struct {
	ID           string
	Title        string
	Slug         string
	Summary      string
	Content      string
	Tags         []string
	Status       string
	CommentCount int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
	PublishedAt  time.Time
}

type BlogPostFilter struct {
	Status string
	Tag    string
	Query  string
	Offset int
	Limit  int
}

type BlogComment struct {
	ID          string
	PostID      string
	PostSlug    string
	AuthorName  string
	AuthorEmail string
	Content     string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BlogCommentFilter struct {
	PostID string
	Status string
	Offset int
	Limit  int
}

type BlogStore interface {
	ListBlogPosts(ctx context.Context, filter BlogPostFilter) ([]BlogPost, error)
	GetBlogPost(ctx context.Context, id string) (BlogPost, error)
	GetBlogPostBySlug(ctx context.Context, slug string) (BlogPost, error)
	CreateBlogPost(ctx context.Context, post BlogPost) (BlogPost, error)
	UpdateBlogPost(ctx context.Context, post BlogPost) (BlogPost, error)
	DeleteBlogPost(ctx context.Context, id string) error
	ListBlogComments(ctx context.Context, filter BlogCommentFilter) ([]BlogComment, error)
	GetBlogComment(ctx context.Context, id string) (BlogComment, error)
	CreateBlogComment(ctx context.Context, comment BlogComment) (BlogComment, error)
	UpdateBlogComment(ctx context.Context, comment BlogComment) (BlogComment, error)
	DeleteBlogComment(ctx context.Context, id string) error
}
