package dao

import (
	"context"
	"testing"
)

func TestMemoryBlogStoreCreateListAndDeleteComment(t *testing.T) {
	t.Parallel()

	store := NewMemoryBlogStore()
	ctx := context.Background()

	post, err := store.CreateBlogPost(ctx, BlogPost{
		ID:      "post-1",
		Title:   "First Post",
		Slug:    "first-post",
		Content: "hello",
		Status:  "published",
	})
	if err != nil {
		t.Fatalf("CreateBlogPost() error = %v", err)
	}

	comment, err := store.CreateBlogComment(ctx, BlogComment{
		ID:         "comment-1",
		PostID:     post.ID,
		AuthorName: "tester",
		Content:    "nice post",
		Status:     "approved",
	})
	if err != nil {
		t.Fatalf("CreateBlogComment() error = %v", err)
	}
	if comment.PostSlug != post.Slug {
		t.Fatalf("CreateBlogComment() post slug = %q, want %q", comment.PostSlug, post.Slug)
	}

	rows, err := store.ListBlogComments(ctx, BlogCommentFilter{
		PostID: post.ID,
		Status: "approved",
	})
	if err != nil {
		t.Fatalf("ListBlogComments() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("ListBlogComments() len = %d, want 1", len(rows))
	}

	updatedPost, err := store.GetBlogPost(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetBlogPost() error = %v", err)
	}
	if updatedPost.CommentCount != 1 {
		t.Fatalf("GetBlogPost() comment_count = %d, want 1", updatedPost.CommentCount)
	}

	if err := store.DeleteBlogComment(ctx, comment.ID); err != nil {
		t.Fatalf("DeleteBlogComment() error = %v", err)
	}

	updatedPost, err = store.GetBlogPost(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetBlogPost() after delete error = %v", err)
	}
	if updatedPost.CommentCount != 0 {
		t.Fatalf("GetBlogPost() after delete comment_count = %d, want 0", updatedPost.CommentCount)
	}
}
