package service

import (
	"context"
	"testing"

	blogv1 "github.com/kongken/datasrv/pkg/proto/blog/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func TestBlogQueryGRPCServerCreateAndListComments(t *testing.T) {
	t.Parallel()

	store := dao.NewMemoryBlogStore()
	admin := NewBlogAdminGRPCServer(store)
	query := NewBlogQueryGRPCServer(store)
	ctx := context.Background()

	createdPost, err := admin.CreatePost(ctx, &blogv1.CreateBlogPostRequest{
		Post: &blogv1.BlogPost{
			Title:   "Hello",
			Slug:    "hello",
			Content: "world",
			Status:  "published",
		},
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}

	createdComment, err := query.CreateComment(ctx, &blogv1.CreateBlogCommentRequest{
		PostSlug: createdPost.GetSlug(),
		Comment: &blogv1.BlogComment{
			AuthorName: "tester",
			Content:    "first",
		},
	})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if createdComment.GetStatus() != "pending" {
		t.Fatalf("CreateComment() status = %q, want pending", createdComment.GetStatus())
	}

	_, err = admin.UpdateComment(ctx, &blogv1.UpdateBlogCommentRequest{
		Comment: &blogv1.BlogComment{
			Id:          createdComment.GetId(),
			AuthorName:  createdComment.GetAuthorName(),
			AuthorEmail: createdComment.GetAuthorEmail(),
			Content:     createdComment.GetContent(),
			Status:      "approved",
		},
	})
	if err != nil {
		t.Fatalf("UpdateComment() error = %v", err)
	}

	listResp, err := query.ListComments(ctx, &blogv1.ListBlogCommentsRequest{
		PostSlug: createdPost.GetSlug(),
	})
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	if len(listResp.GetComments()) != 1 {
		t.Fatalf("ListComments() len = %d, want 1", len(listResp.GetComments()))
	}
	if listResp.GetComments()[0].GetStatus() != "approved" {
		t.Fatalf("ListComments()[0].status = %q, want approved", listResp.GetComments()[0].GetStatus())
	}
}
