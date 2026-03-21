package dao

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

type MemoryBlogStore struct {
	mu           sync.RWMutex
	posts        map[string]BlogPost
	postsBySlug  map[string]string
	postOrder    []string
	comments     map[string]BlogComment
	postComments map[string][]string
}

func NewMemoryBlogStore() *MemoryBlogStore {
	return &MemoryBlogStore{
		posts:        make(map[string]BlogPost),
		postsBySlug:  make(map[string]string),
		comments:     make(map[string]BlogComment),
		postComments: make(map[string][]string),
	}
}

func (s *MemoryBlogStore) ListBlogPosts(_ context.Context, filter BlogPostFilter) ([]BlogPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]BlogPost, 0, len(s.postOrder))
	for _, id := range s.postOrder {
		post, ok := s.posts[id]
		if !ok {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(post.Status, filter.Status) {
			continue
		}
		if filter.Tag != "" && !containsFold(post.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !matchesBlogPostQuery(post, filter.Query) {
			continue
		}
		rows = append(rows, cloneBlogPost(post))
	}
	return paginatePosts(rows, filter.Offset, filter.Limit), nil
}

func (s *MemoryBlogStore) GetBlogPost(_ context.Context, id string) (BlogPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	post, ok := s.posts[id]
	if !ok {
		return BlogPost{}, ErrBlogPostNotFound
	}
	return cloneBlogPost(post), nil
}

func (s *MemoryBlogStore) GetBlogPostBySlug(_ context.Context, slug string) (BlogPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.postsBySlug[slug]
	if !ok {
		return BlogPost{}, ErrBlogPostNotFound
	}
	post, ok := s.posts[id]
	if !ok {
		return BlogPost{}, ErrBlogPostNotFound
	}
	return cloneBlogPost(post), nil
}

func (s *MemoryBlogStore) CreateBlogPost(_ context.Context, post BlogPost) (BlogPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.postsBySlug[post.Slug]; exists {
		return BlogPost{}, ErrBlogPostSlugConflict
	}
	now := time.Now().UTC()
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	post.UpdatedAt = now
	post.Tags = cloneStrings(post.Tags)
	s.posts[post.ID] = post
	s.postsBySlug[post.Slug] = post.ID
	s.postOrder = append([]string{post.ID}, s.postOrder...)
	return cloneBlogPost(post), nil
}

func (s *MemoryBlogStore) UpdateBlogPost(_ context.Context, post BlogPost) (BlogPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.posts[post.ID]
	if !ok {
		return BlogPost{}, ErrBlogPostNotFound
	}
	if existingID, exists := s.postsBySlug[post.Slug]; exists && existingID != post.ID {
		return BlogPost{}, ErrBlogPostSlugConflict
	}
	if current.Slug != post.Slug {
		delete(s.postsBySlug, current.Slug)
		s.postsBySlug[post.Slug] = post.ID
	}
	post.CreatedAt = current.CreatedAt
	post.CommentCount = current.CommentCount
	post.UpdatedAt = time.Now().UTC()
	post.Tags = cloneStrings(post.Tags)
	s.posts[post.ID] = post
	return cloneBlogPost(post), nil
}

func (s *MemoryBlogStore) DeleteBlogPost(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, ok := s.posts[id]
	if !ok {
		return ErrBlogPostNotFound
	}
	delete(s.posts, id)
	delete(s.postsBySlug, post.Slug)
	s.postOrder = deleteID(s.postOrder, id)
	for _, commentID := range s.postComments[id] {
		delete(s.comments, commentID)
	}
	delete(s.postComments, id)
	return nil
}

func (s *MemoryBlogStore) ListBlogComments(_ context.Context, filter BlogCommentFilter) ([]BlogComment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commentIDs := s.postComments[filter.PostID]
	rows := make([]BlogComment, 0, len(commentIDs))
	for _, id := range commentIDs {
		comment, ok := s.comments[id]
		if !ok {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(comment.Status, filter.Status) {
			continue
		}
		rows = append(rows, cloneBlogComment(comment))
	}
	return paginateComments(rows, filter.Offset, filter.Limit), nil
}

func (s *MemoryBlogStore) GetBlogComment(_ context.Context, id string) (BlogComment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	comment, ok := s.comments[id]
	if !ok {
		return BlogComment{}, ErrBlogCommentNotFound
	}
	return cloneBlogComment(comment), nil
}

func (s *MemoryBlogStore) CreateBlogComment(_ context.Context, comment BlogComment) (BlogComment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, ok := s.posts[comment.PostID]
	if !ok {
		return BlogComment{}, ErrBlogCommentPostAbsent
	}
	now := time.Now().UTC()
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = now
	}
	comment.UpdatedAt = now
	comment.PostSlug = post.Slug
	s.comments[comment.ID] = comment
	s.postComments[comment.PostID] = append(s.postComments[comment.PostID], comment.ID)
	post.CommentCount++
	post.UpdatedAt = now
	s.posts[post.ID] = post
	return cloneBlogComment(comment), nil
}

func (s *MemoryBlogStore) UpdateBlogComment(_ context.Context, comment BlogComment) (BlogComment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.comments[comment.ID]
	if !ok {
		return BlogComment{}, ErrBlogCommentNotFound
	}
	post, ok := s.posts[current.PostID]
	if !ok {
		return BlogComment{}, ErrBlogCommentPostAbsent
	}
	comment.PostID = current.PostID
	comment.PostSlug = post.Slug
	comment.CreatedAt = current.CreatedAt
	comment.UpdatedAt = time.Now().UTC()
	s.comments[comment.ID] = comment
	return cloneBlogComment(comment), nil
}

func (s *MemoryBlogStore) DeleteBlogComment(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	comment, ok := s.comments[id]
	if !ok {
		return ErrBlogCommentNotFound
	}
	delete(s.comments, id)
	s.postComments[comment.PostID] = deleteID(s.postComments[comment.PostID], id)
	if post, ok := s.posts[comment.PostID]; ok {
		if post.CommentCount > 0 {
			post.CommentCount--
		}
		post.UpdatedAt = time.Now().UTC()
		s.posts[post.ID] = post
	}
	return nil
}

func matchesBlogPostQuery(post BlogPost, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(post.Title), query) ||
		strings.Contains(strings.ToLower(post.Slug), query) ||
		strings.Contains(strings.ToLower(post.Summary), query) ||
		strings.Contains(strings.ToLower(post.Content), query)
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func paginatePosts(rows []BlogPost, offset, limit int) []BlogPost {
	if offset >= len(rows) {
		return []BlogPost{}
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return rows[offset:]
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func paginateComments(rows []BlogComment, offset, limit int) []BlogComment {
	if offset >= len(rows) {
		return []BlogComment{}
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		return rows[offset:]
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

func deleteID(items []string, id string) []string {
	index := slices.Index(items, id)
	if index < 0 {
		return items
	}
	return append(items[:index], items[index+1:]...)
}

func cloneBlogPost(post BlogPost) BlogPost {
	post.Tags = cloneStrings(post.Tags)
	return post
}

func cloneBlogComment(comment BlogComment) BlogComment {
	return comment
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
