package service

import (
	"context"
	"fmt"

	"github.com/google/go-github/v82/github"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

// GitHubService provides operations for fetching and storing GitHub issues
type GitHubService struct {
	client *github.Client
	dao    dao.DAO
}

// NewGitHubService creates a new GitHub service instance
func NewGitHubService(client *github.Client, dao dao.DAO) *GitHubService {
	return &GitHubService{
		client: client,
		dao:    dao,
	}
}

// FetchAndStoreIssues fetches issues from a GitHub repository and stores them in the database
// owner: repository owner
// repo: repository name
// opts: options for listing issues (state, labels, etc.)
func (s *GitHubService) FetchAndStoreIssues(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) error {
	// Upsert repository metadata first.
	if err := s.SyncRepository(ctx, owner, repo); err != nil {
		return fmt.Errorf("failed to sync repository metadata: %w", err)
	}

	// Fetch issues from GitHub
	issues, _, err := s.client.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch issues from GitHub: %w", err)
	}

	// Convert GitHub issues to DAO models and persist them
	return s.persistIssues(ctx, issues)
}

// FetchAndStoreAllIssues fetches all issues from a GitHub repository with pagination
func (s *GitHubService) FetchAndStoreAllIssues(ctx context.Context, owner, repo string, state string) error {
	// Upsert repository metadata first.
	if err := s.SyncRepository(ctx, owner, repo); err != nil {
		return fmt.Errorf("failed to sync repository metadata: %w", err)
	}

	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100, // Maximum allowed by GitHub API
		},
	}

	for {
		issues, resp, err := s.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return fmt.Errorf("failed to fetch issues from GitHub (page %d): %w", opts.ListOptions.Page, err)
		}

		if len(issues) == 0 {
			break
		}

		// Persist the batch of issues
		if err := s.persistIssues(ctx, issues); err != nil {
			return fmt.Errorf("failed to persist issues (page %d): %w", opts.ListOptions.Page, err)
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	return nil
}

// SyncRepository fetches repository metadata from GitHub and upserts it into the database.
func (s *GitHubService) SyncRepository(ctx context.Context, owner, repo string) error {
	ghRepo, _, err := s.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch repository %s/%s from GitHub: %w", owner, repo, err)
	}
	return s.dao.UpsertRepository(ctx, s.convertGitHubRepositoryToModel(ghRepo))
}

// persistIssues converts GitHub issues to DAO models and persists them
func (s *GitHubService) persistIssues(ctx context.Context, ghIssues []*github.Issue) error {
	// First, upsert all users, labels, and milestones
	for _, ghIssue := range ghIssues {
		// Upsert user (creator)
		if ghIssue.User != nil {
			userModel := s.convertGitHubUserToModel(ghIssue.User)
			if err := s.dao.UpsertUser(ctx, userModel); err != nil {
				return fmt.Errorf("failed to upsert user %d: %w", userModel.ID, err)
			}
		}

		// Upsert assignees
		for _, assignee := range ghIssue.Assignees {
			userModel := s.convertGitHubUserToModel(assignee)
			if err := s.dao.UpsertUser(ctx, userModel); err != nil {
				return fmt.Errorf("failed to upsert assignee %d: %w", userModel.ID, err)
			}
		}

		// Upsert labels
		for _, label := range ghIssue.Labels {
			labelModel := s.convertGitHubLabelToModel(label)
			if err := s.dao.UpsertLabel(ctx, labelModel); err != nil {
				return fmt.Errorf("failed to upsert label %d: %w", labelModel.ID, err)
			}
		}

		// Upsert milestone
		if ghIssue.Milestone != nil {
			milestoneModel := s.convertGitHubMilestoneToModel(ghIssue.Milestone)
			if err := s.dao.UpsertMilestone(ctx, milestoneModel); err != nil {
				return fmt.Errorf("failed to upsert milestone %d: %w", milestoneModel.ID, err)
			}
		}
	}

	// Convert GitHub issues to DAO models
	issueModels := make([]*dao.IssueModel, len(ghIssues))
	for i, ghIssue := range ghIssues {
		issueModels[i] = s.convertGitHubIssueToModel(ghIssue)
	}

	// Batch create/update issues
	return s.dao.BatchCreateIssues(ctx, issueModels)
}

// GetIssueByNumber retrieves an issue by its number
func (s *GitHubService) GetIssueByNumber(ctx context.Context, number int32) (*dao.IssueModel, error) {
	return s.dao.GetIssueByNumber(ctx, number)
}

// GetIssueByID retrieves an issue by its GitHub ID
func (s *GitHubService) GetIssueByID(ctx context.Context, id int64) (*dao.IssueModel, error) {
	return s.dao.GetIssueByID(ctx, id)
}

// ListIssues retrieves a list of issues from the database
func (s *GitHubService) ListIssues(ctx context.Context, opts *dao.ListOptions) ([]*dao.IssueModel, error) {
	return s.dao.ListIssues(ctx, opts)
}

// GetRepositoryByID retrieves repository metadata by GitHub repository ID.
func (s *GitHubService) GetRepositoryByID(ctx context.Context, id int64) (*dao.RepositoryModel, error) {
	return s.dao.GetRepositoryByID(ctx, id)
}

// GetRepositoryByFullName retrieves repository metadata by full name (owner/name).
func (s *GitHubService) GetRepositoryByFullName(ctx context.Context, fullName string) (*dao.RepositoryModel, error) {
	return s.dao.GetRepositoryByFullName(ctx, fullName)
}

// ListRepositories retrieves repository metadata from the database.
func (s *GitHubService) ListRepositories(ctx context.Context, opts *dao.RepositoryListOptions) ([]*dao.RepositoryModel, error) {
	return s.dao.ListRepositories(ctx, opts)
}

// convertGitHubIssueToModel converts a GitHub issue to a DAO model
func (s *GitHubService) convertGitHubIssueToModel(ghIssue *github.Issue) *dao.IssueModel {
	model := &dao.IssueModel{
		ID:        ghIssue.GetID(),
		Number:    int32(ghIssue.GetNumber()),
		Title:     ghIssue.GetTitle(),
		Body:      ghIssue.GetBody(),
		State:     ghIssue.GetState(),
		Comments:  int32(ghIssue.GetComments()),
		HTMLURL:   ghIssue.GetHTMLURL(),
		Locked:    ghIssue.GetLocked(),
		CreatedAt: ghIssue.GetCreatedAt().Time,
		UpdatedAt: ghIssue.GetUpdatedAt().Time,
	}

	// Set closed_at if available
	if ghIssue.ClosedAt != nil {
		closedAt := ghIssue.GetClosedAt().Time
		model.ClosedAt = &closedAt
	}

	// Set user ID
	if ghIssue.User != nil {
		model.UserID = ghIssue.User.GetID()
	}

	// Set milestone ID
	if ghIssue.Milestone != nil {
		milestoneID := ghIssue.Milestone.GetID()
		model.MilestoneID = &milestoneID
	}

	// Set label IDs
	if len(ghIssue.Labels) > 0 {
		model.Labels = make([]int64, len(ghIssue.Labels))
		for i, label := range ghIssue.Labels {
			model.Labels[i] = label.GetID()
		}
	}

	// Set assignee IDs
	if len(ghIssue.Assignees) > 0 {
		model.Assignees = make([]int64, len(ghIssue.Assignees))
		for i, assignee := range ghIssue.Assignees {
			model.Assignees[i] = assignee.GetID()
		}
	}

	return model
}

// convertGitHubRepositoryToModel converts a GitHub repository to a DAO model.
func (s *GitHubService) convertGitHubRepositoryToModel(ghRepo *github.Repository) *dao.RepositoryModel {
	ownerLogin := ""
	if ghRepo.Owner != nil {
		ownerLogin = ghRepo.Owner.GetLogin()
	}

	model := &dao.RepositoryModel{
		ID:              ghRepo.GetID(),
		Name:            ghRepo.GetName(),
		FullName:        ghRepo.GetFullName(),
		OwnerLogin:      ownerLogin,
		Description:     ghRepo.GetDescription(),
		Private:         ghRepo.GetPrivate(),
		Archived:        ghRepo.GetArchived(),
		Disabled:        ghRepo.GetDisabled(),
		HTMLURL:         ghRepo.GetHTMLURL(),
		DefaultBranch:   ghRepo.GetDefaultBranch(),
		Language:        ghRepo.GetLanguage(),
		StargazersCount: int32(ghRepo.GetStargazersCount()),
		ForksCount:      int32(ghRepo.GetForksCount()),
		OpenIssuesCount: int32(ghRepo.GetOpenIssuesCount()),
		CreatedAt:       ghRepo.GetCreatedAt().Time,
		UpdatedAt:       ghRepo.GetUpdatedAt().Time,
	}

	if model.FullName == "" && model.OwnerLogin != "" && model.Name != "" {
		model.FullName = model.OwnerLogin + "/" + model.Name
	}
	if model.DefaultBranch == "" {
		model.DefaultBranch = "main"
	}

	// Set pushed_at if available.
	if ghRepo.PushedAt != nil {
		pushedAt := ghRepo.GetPushedAt().Time
		model.PushedAt = &pushedAt
	}
	return model
}

// convertGitHubUserToModel converts a GitHub user to a DAO model
func (s *GitHubService) convertGitHubUserToModel(ghUser *github.User) *dao.UserModel {
	return &dao.UserModel{
		ID:        ghUser.GetID(),
		Login:     ghUser.GetLogin(),
		AvatarURL: ghUser.GetAvatarURL(),
		HTMLURL:   ghUser.GetHTMLURL(),
	}
}

// convertGitHubLabelToModel converts a GitHub label to a DAO model
func (s *GitHubService) convertGitHubLabelToModel(ghLabel *github.Label) *dao.LabelModel {
	return &dao.LabelModel{
		ID:          ghLabel.GetID(),
		Name:        ghLabel.GetName(),
		Color:       ghLabel.GetColor(),
		Description: ghLabel.GetDescription(),
	}
}

// convertGitHubMilestoneToModel converts a GitHub milestone to a DAO model
func (s *GitHubService) convertGitHubMilestoneToModel(ghMilestone *github.Milestone) *dao.MilestoneModel {
	model := &dao.MilestoneModel{
		ID:          ghMilestone.GetID(),
		Number:      int32(ghMilestone.GetNumber()),
		Title:       ghMilestone.GetTitle(),
		Description: ghMilestone.GetDescription(),
		State:       ghMilestone.GetState(),
		CreatedAt:   ghMilestone.GetCreatedAt().Time,
		UpdatedAt:   ghMilestone.GetUpdatedAt().Time,
	}

	// Set due_on if available
	if ghMilestone.DueOn != nil {
		dueOn := ghMilestone.GetDueOn().Time
		model.DueOn = &dueOn
	}

	return model
}

// SyncIssue fetches a single issue from GitHub and stores it
func (s *GitHubService) SyncIssue(ctx context.Context, owner, repo string, issueNumber int) error {
	// Fetch the issue from GitHub
	ghIssue, _, err := s.client.Issues.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch issue #%d from GitHub: %w", issueNumber, err)
	}

	// Persist the issue
	return s.persistIssues(ctx, []*github.Issue{ghIssue})
}

// UpdateIssueFromGitHub updates an existing issue in the database from GitHub
func (s *GitHubService) UpdateIssueFromGitHub(ctx context.Context, owner, repo string, issueNumber int) error {
	// Same as SyncIssue - the persistIssues method handles both create and update
	return s.SyncIssue(ctx, owner, repo, issueNumber)
}
