package service

import (
	"context"
	"fmt"
	"sync"

	commv1 "github.com/kongken/monkey/pkg/proto/comm/v1"
)

type RepoService struct {
	commv1.UnimplementedRepoServiceServer
	mu    sync.RWMutex
	repos map[string]*commv1.Repo
}

func NewRepoService() *RepoService {
	return &RepoService{
		repos: make(map[string]*commv1.Repo),
	}
}

func (s *RepoService) CreateRepo(ctx context.Context, req *commv1.CreateRepoRequest) (*commv1.CreateRepoResponse, error) {
	if req.Repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if req.Repo.Id == "" {
		return nil, fmt.Errorf("repo id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.repos[req.Repo.Id]; exists {
		return nil, fmt.Errorf("repo with id %s already exists", req.Repo.Id)
	}

	s.repos[req.Repo.Id] = req.Repo

	return &commv1.CreateRepoResponse{
		Repo: req.Repo,
	}, nil
}

func (s *RepoService) GetRepo(ctx context.Context, req *commv1.GetRepoRequest) (*commv1.GetRepoResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("repo id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	repo, exists := s.repos[req.Id]
	if !exists {
		return nil, fmt.Errorf("repo with id %s not found", req.Id)
	}

	return &commv1.GetRepoResponse{
		Repo: repo,
	}, nil
}

func (s *RepoService) UpdateRepo(ctx context.Context, req *commv1.UpdateRepoRequest) (*commv1.UpdateRepoResponse, error) {
	if req.Repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if req.Repo.Id == "" {
		return nil, fmt.Errorf("repo id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.repos[req.Repo.Id]; !exists {
		return nil, fmt.Errorf("repo with id %s not found", req.Repo.Id)
	}

	s.repos[req.Repo.Id] = req.Repo

	return &commv1.UpdateRepoResponse{
		Repo: req.Repo,
	}, nil
}

func (s *RepoService) DeleteRepo(ctx context.Context, req *commv1.DeleteRepoRequest) (*commv1.DeleteRepoResponse, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("repo id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.repos[req.Id]; !exists {
		return nil, fmt.Errorf("repo with id %s not found", req.Id)
	}

	delete(s.repos, req.Id)

	return &commv1.DeleteRepoResponse{
		Success: true,
	}, nil
}

func (s *RepoService) ListRepos(ctx context.Context, req *commv1.ListReposRequest) (*commv1.ListReposResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repos := make([]*commv1.Repo, 0, len(s.repos))
	for _, repo := range s.repos {
		repos = append(repos, repo)
	}

	// 简单的分页实现
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	start := int((page - 1) * pageSize)
	end := int(page * pageSize)

	if start >= len(repos) {
		return &commv1.ListReposResponse{
			Repos: []*commv1.Repo{},
			Total: int32(len(repos)),
		}, nil
	}

	if end > len(repos) {
		end = len(repos)
	}

	return &commv1.ListReposResponse{
		Repos: repos[start:end],
		Total: int32(len(repos)),
	}, nil
}
