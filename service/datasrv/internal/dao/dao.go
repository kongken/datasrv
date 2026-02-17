package dao

import (
	"context"
	"time"
)

// IssueDAO defines the interface for issue data access operations
// This abstraction allows for different implementations (PostgreSQL, MongoDB, etc.)
type IssueDAO interface {
	// CreateIssue creates a new issue in the database
	CreateIssue(ctx context.Context, issue *IssueModel) error

	// BatchCreateIssues creates multiple issues in a single transaction
	BatchCreateIssues(ctx context.Context, issues []*IssueModel) error

	// GetIssueByID retrieves an issue by its GitHub ID
	GetIssueByID(ctx context.Context, id int64) (*IssueModel, error)

	// GetIssueByNumber retrieves an issue by its number
	GetIssueByNumber(ctx context.Context, number int32) (*IssueModel, error)

	// ListIssues retrieves a list of issues with pagination
	ListIssues(ctx context.Context, opts *ListOptions) ([]*IssueModel, error)

	// UpdateIssue updates an existing issue
	UpdateIssue(ctx context.Context, issue *IssueModel) error

	// DeleteIssue deletes an issue by ID
	DeleteIssue(ctx context.Context, id int64) error

	// Close closes the DAO connection
	Close() error
}

// UserDAO defines the interface for user data access operations
type UserDAO interface {
	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *UserModel) error

	// GetUserByID retrieves a user by ID
	GetUserByID(ctx context.Context, id int64) (*UserModel, error)

	// UpsertUser creates or updates a user
	UpsertUser(ctx context.Context, user *UserModel) error
}

// LabelDAO defines the interface for label data access operations
type LabelDAO interface {
	// CreateLabel creates a new label
	CreateLabel(ctx context.Context, label *LabelModel) error

	// GetLabelByID retrieves a label by ID
	GetLabelByID(ctx context.Context, id int64) (*LabelModel, error)

	// UpsertLabel creates or updates a label
	UpsertLabel(ctx context.Context, label *LabelModel) error
}

// MilestoneDAO defines the interface for milestone data access operations
type MilestoneDAO interface {
	// CreateMilestone creates a new milestone
	CreateMilestone(ctx context.Context, milestone *MilestoneModel) error

	// GetMilestoneByID retrieves a milestone by ID
	GetMilestoneByID(ctx context.Context, id int64) (*MilestoneModel, error)

	// UpsertMilestone creates or updates a milestone
	UpsertMilestone(ctx context.Context, milestone *MilestoneModel) error
}

// DAO aggregates all DAO interfaces
type DAO interface {
	IssueDAO
	UserDAO
	LabelDAO
	MilestoneDAO
}

// ListOptions defines options for listing issues
type ListOptions struct {
	Offset int
	Limit  int
	State  string // "open", "closed", or "all"
}

// IssueModel represents the issue data model
type IssueModel struct {
	ID         int64
	Number     int32
	Title      string
	Body       string
	State      string
	Comments   int32
	HTMLURL    string
	Locked     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ClosedAt   *time.Time
	UserID     int64
	MilestoneID *int64
	Labels     []int64 // Label IDs
	Assignees  []int64 // User IDs
}

// UserModel represents the user data model
type UserModel struct {
	ID        int64
	Login     string
	AvatarURL string
	HTMLURL   string
}

// LabelModel represents the label data model
type LabelModel struct {
	ID          int64
	Name        string
	Color       string
	Description string
}

// MilestoneModel represents the milestone data model
type MilestoneModel struct {
	ID          int64
	Number      int32
	Title       string
	Description string
	State       string
	DueOn       *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
