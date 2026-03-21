package service

import (
	"context"
	"testing"

	blogv1 "github.com/kongken/datasrv/pkg/proto/blog/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func TestBlogQueryGRPCServerCreateAndListComments(t *testing.T) {
	t.Parallel()

	store := newStubBlogStore()
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

type stubBlogStore struct {
	postsByID   map[string]dao.BlogPost
	postBySlug  map[string]string
	comments    map[string]dao.BlogComment
	commentList map[string][]string
}

func newStubBlogStore() *stubBlogStore {
	return &stubBlogStore{
		postsByID:   make(map[string]dao.BlogPost),
		postBySlug:  make(map[string]string),
		comments:    make(map[string]dao.BlogComment),
		commentList: make(map[string][]string),
	}
}

func (s *stubBlogStore) ListBlogPosts(_ context.Context, filter dao.BlogPostFilter) ([]dao.BlogPost, error) {
	var posts []dao.BlogPost
	for _, post := range s.postsByID {
		if filter.Status != "" && post.Status != filter.Status {
			continue
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (s *stubBlogStore) GetBlogPost(_ context.Context, id string) (dao.BlogPost, error) {
	post, ok := s.postsByID[id]
	if !ok {
		return dao.BlogPost{}, dao.ErrBlogPostNotFound
	}
	return post, nil
}

func (s *stubBlogStore) GetBlogPostBySlug(_ context.Context, slug string) (dao.BlogPost, error) {
	id, ok := s.postBySlug[slug]
	if !ok {
		return dao.BlogPost{}, dao.ErrBlogPostNotFound
	}
	return s.postsByID[id], nil
}

func (s *stubBlogStore) CreateBlogPost(_ context.Context, post dao.BlogPost) (dao.BlogPost, error) {
	s.postsByID[post.ID] = post
	s.postBySlug[post.Slug] = post.ID
	return post, nil
}

func (s *stubBlogStore) UpdateBlogPost(_ context.Context, post dao.BlogPost) (dao.BlogPost, error) {
	current, ok := s.postsByID[post.ID]
	if !ok {
		return dao.BlogPost{}, dao.ErrBlogPostNotFound
	}
	post.CommentCount = current.CommentCount
	s.postsByID[post.ID] = post
	s.postBySlug[post.Slug] = post.ID
	return post, nil
}

func (s *stubBlogStore) DeleteBlogPost(_ context.Context, id string) error {
	delete(s.postsByID, id)
	return nil
}

func (s *stubBlogStore) ListBlogComments(_ context.Context, filter dao.BlogCommentFilter) ([]dao.BlogComment, error) {
	var comments []dao.BlogComment
	for _, id := range s.commentList[filter.PostID] {
		comment := s.comments[id]
		if filter.Status != "" && comment.Status != filter.Status {
			continue
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (s *stubBlogStore) GetBlogComment(_ context.Context, id string) (dao.BlogComment, error) {
	comment, ok := s.comments[id]
	if !ok {
		return dao.BlogComment{}, dao.ErrBlogCommentNotFound
	}
	return comment, nil
}

func (s *stubBlogStore) CreateBlogComment(_ context.Context, comment dao.BlogComment) (dao.BlogComment, error) {
	post, ok := s.postsByID[comment.PostID]
	if !ok {
		return dao.BlogComment{}, dao.ErrBlogCommentPostAbsent
	}
	post.CommentCount++
	s.postsByID[post.ID] = post
	s.comments[comment.ID] = comment
	s.commentList[comment.PostID] = append(s.commentList[comment.PostID], comment.ID)
	return comment, nil
}

func (s *stubBlogStore) UpdateBlogComment(_ context.Context, comment dao.BlogComment) (dao.BlogComment, error) {
	current, ok := s.comments[comment.ID]
	if !ok {
		return dao.BlogComment{}, dao.ErrBlogCommentNotFound
	}
	comment.PostID = current.PostID
	comment.PostSlug = current.PostSlug
	s.comments[comment.ID] = comment
	return comment, nil
}

func (s *stubBlogStore) DeleteBlogComment(_ context.Context, id string) error {
	delete(s.comments, id)
	return nil
}
