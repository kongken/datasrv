package dao

import (
	"context"
	"fmt"

	"github.com/kongken/datasrv/service/datasrv/internal/dao/ent"
	"github.com/kongken/datasrv/service/datasrv/internal/dao/ent/issue"
	"github.com/kongken/datasrv/service/datasrv/internal/dao/ent/label"
	"github.com/kongken/datasrv/service/datasrv/internal/dao/ent/milestone"
	"github.com/kongken/datasrv/service/datasrv/internal/dao/ent/user"

	_ "github.com/lib/pq"
)

// PostgresDAO implements DAO interface using PostgreSQL and ent ORM
type PostgresDAO struct {
	client *ent.Client
}

// NewPostgresDAO creates a new PostgreSQL DAO instance
func NewPostgresDAO(dsn string) (*PostgresDAO, error) {
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	return &PostgresDAO{
		client: client,
	}, nil
}

// Migrate runs the database migrations
func (d *PostgresDAO) Migrate(ctx context.Context) error {
	if err := d.client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("failed creating schema resources: %w", err)
	}
	return nil
}

// Close closes the database connection
func (d *PostgresDAO) Close() error {
	return d.client.Close()
}

// CreateIssue creates a new issue
func (d *PostgresDAO) CreateIssue(ctx context.Context, issueModel *IssueModel) error {
	tx, err := d.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// Create or get user
	if issueModel.UserID != 0 {
		_, err = tx.User.Query().Where(user.ID(issueModel.UserID)).Only(ctx)
		if err != nil && ent.IsNotFound(err) {
			// User doesn't exist, skip for now (should be created separately)
		}
	}

	// Create or get milestone
	var milestoneExists bool
	if issueModel.MilestoneID != nil && *issueModel.MilestoneID != 0 {
		_, err = tx.Milestone.Query().Where(milestone.ID(*issueModel.MilestoneID)).Only(ctx)
		if err == nil {
			milestoneExists = true
		} else if !ent.IsNotFound(err) {
			tx.Rollback()
			return fmt.Errorf("failed to query milestone: %w", err)
		}
	}

	// Create issue
	creator := tx.Issue.Create().
		SetID(issueModel.ID).
		SetNumber(issueModel.Number).
		SetTitle(issueModel.Title).
		SetBody(issueModel.Body).
		SetState(issueModel.State).
		SetComments(issueModel.Comments).
		SetHTMLURL(issueModel.HTMLURL).
		SetLocked(issueModel.Locked).
		SetCreatedAt(issueModel.CreatedAt).
		SetUpdatedAt(issueModel.UpdatedAt)

	if issueModel.UserID != 0 {
		creator.SetUserID(issueModel.UserID)
	}

	if issueModel.ClosedAt != nil {
		creator.SetClosedAt(*issueModel.ClosedAt)
	}

	if issueModel.MilestoneID != nil && milestoneExists {
		creator.SetMilestoneID(*issueModel.MilestoneID)
	}

	// Add labels if they exist
	if len(issueModel.Labels) > 0 {
		creator.AddLabelIDs(issueModel.Labels...)
	}

	// Add assignees if they exist
	if len(issueModel.Assignees) > 0 {
		creator.AddAssigneeIDs(issueModel.Assignees...)
	}

	if _, err := creator.Save(ctx); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create issue: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchCreateIssues creates multiple issues in a transaction
func (d *PostgresDAO) BatchCreateIssues(ctx context.Context, issues []*IssueModel) error {
	tx, err := d.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	for _, issueModel := range issues {
		// Create or get user
		if issueModel.UserID != 0 {
			_, err = tx.User.Query().Where(user.ID(issueModel.UserID)).Only(ctx)
			if err != nil && ent.IsNotFound(err) {
				// User doesn't exist, skip for now
			}
		}

		// Create or get milestone
		var milestoneExists bool
		if issueModel.MilestoneID != nil && *issueModel.MilestoneID != 0 {
			_, err = tx.Milestone.Query().Where(milestone.ID(*issueModel.MilestoneID)).Only(ctx)
			if err == nil {
				milestoneExists = true
			} else if !ent.IsNotFound(err) {
				tx.Rollback()
				return fmt.Errorf("failed to query milestone: %w", err)
			}
		}

		creator := tx.Issue.Create().
			SetID(issueModel.ID).
			SetNumber(issueModel.Number).
			SetTitle(issueModel.Title).
			SetBody(issueModel.Body).
			SetState(issueModel.State).
			SetComments(issueModel.Comments).
			SetHTMLURL(issueModel.HTMLURL).
			SetLocked(issueModel.Locked).
			SetCreatedAt(issueModel.CreatedAt).
			SetUpdatedAt(issueModel.UpdatedAt)

		if issueModel.UserID != 0 {
			creator.SetUserID(issueModel.UserID)
		}

		if issueModel.ClosedAt != nil {
			creator.SetClosedAt(*issueModel.ClosedAt)
		}

		if issueModel.MilestoneID != nil && milestoneExists {
			creator.SetMilestoneID(*issueModel.MilestoneID)
		}

		if len(issueModel.Labels) > 0 {
			creator.AddLabelIDs(issueModel.Labels...)
		}

		if len(issueModel.Assignees) > 0 {
			creator.AddAssigneeIDs(issueModel.Assignees...)
		}

		if _, err := creator.Save(ctx); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create issue %d: %w", issueModel.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetIssueByID retrieves an issue by its GitHub ID
func (d *PostgresDAO) GetIssueByID(ctx context.Context, id int64) (*IssueModel, error) {
	iss, err := d.client.Issue.Query().
		Where(issue.ID(id)).
		WithUser().
		WithLabels().
		WithAssignees().
		WithMilestone().
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("issue not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query issue: %w", err)
	}

	return d.entIssueToModel(iss), nil
}

// GetIssueByNumber retrieves an issue by its number
func (d *PostgresDAO) GetIssueByNumber(ctx context.Context, number int32) (*IssueModel, error) {
	iss, err := d.client.Issue.Query().
		Where(issue.Number(number)).
		WithUser().
		WithLabels().
		WithAssignees().
		WithMilestone().
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("issue not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query issue: %w", err)
	}

	return d.entIssueToModel(iss), nil
}

// ListIssues retrieves a list of issues with pagination
func (d *PostgresDAO) ListIssues(ctx context.Context, opts *ListOptions) ([]*IssueModel, error) {
	query := d.client.Issue.Query().
		WithUser().
		WithLabels().
		WithAssignees().
		WithMilestone()

	if opts.State != "" && opts.State != "all" {
		query = query.Where(issue.State(opts.State))
	}

	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	issues, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}

	result := make([]*IssueModel, len(issues))
	for i, iss := range issues {
		result[i] = d.entIssueToModel(iss)
	}

	return result, nil
}

// UpdateIssue updates an existing issue
func (d *PostgresDAO) UpdateIssue(ctx context.Context, issueModel *IssueModel) error {
	updater := d.client.Issue.UpdateOneID(issueModel.ID).
		SetNumber(issueModel.Number).
		SetTitle(issueModel.Title).
		SetBody(issueModel.Body).
		SetState(issueModel.State).
		SetComments(issueModel.Comments).
		SetHTMLURL(issueModel.HTMLURL).
		SetLocked(issueModel.Locked).
		SetUpdatedAt(issueModel.UpdatedAt)

	if issueModel.ClosedAt != nil {
		updater.SetClosedAt(*issueModel.ClosedAt)
	} else {
		updater.ClearClosedAt()
	}

	if issueModel.UserID != 0 {
		updater.SetUserID(issueModel.UserID)
	}

	if issueModel.MilestoneID != nil {
		updater.SetMilestoneID(*issueModel.MilestoneID)
	} else {
		updater.ClearMilestoneID()
	}

	if _, err := updater.Save(ctx); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return nil
}

// DeleteIssue deletes an issue by ID
func (d *PostgresDAO) DeleteIssue(ctx context.Context, id int64) error {
	if err := d.client.Issue.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	return nil
}

// CreateUser creates a new user
func (d *PostgresDAO) CreateUser(ctx context.Context, userModel *UserModel) error {
	_, err := d.client.User.Create().
		SetID(userModel.ID).
		SetLogin(userModel.Login).
		SetAvatarURL(userModel.AvatarURL).
		SetHTMLURL(userModel.HTMLURL).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (d *PostgresDAO) GetUserByID(ctx context.Context, id int64) (*UserModel, error) {
	u, err := d.client.User.Query().Where(user.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &UserModel{
		ID:        u.ID,
		Login:     u.Login,
		AvatarURL: u.AvatarURL,
		HTMLURL:   u.HTMLURL,
	}, nil
}

// UpsertUser creates or updates a user
func (d *PostgresDAO) UpsertUser(ctx context.Context, userModel *UserModel) error {
	_, err := d.client.User.Query().Where(user.ID(userModel.ID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return d.CreateUser(ctx, userModel)
		}
		return fmt.Errorf("failed to query user: %w", err)
	}

	// Update existing user
	_, err = d.client.User.UpdateOneID(userModel.ID).
		SetLogin(userModel.Login).
		SetAvatarURL(userModel.AvatarURL).
		SetHTMLURL(userModel.HTMLURL).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// CreateLabel creates a new label
func (d *PostgresDAO) CreateLabel(ctx context.Context, labelModel *LabelModel) error {
	_, err := d.client.Label.Create().
		SetID(labelModel.ID).
		SetName(labelModel.Name).
		SetColor(labelModel.Color).
		SetDescription(labelModel.Description).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create label: %w", err)
	}

	return nil
}

// GetLabelByID retrieves a label by ID
func (d *PostgresDAO) GetLabelByID(ctx context.Context, id int64) (*LabelModel, error) {
	l, err := d.client.Label.Query().Where(label.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("label not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query label: %w", err)
	}

	return &LabelModel{
		ID:          l.ID,
		Name:        l.Name,
		Color:       l.Color,
		Description: l.Description,
	}, nil
}

// UpsertLabel creates or updates a label
func (d *PostgresDAO) UpsertLabel(ctx context.Context, labelModel *LabelModel) error {
	_, err := d.client.Label.Query().Where(label.ID(labelModel.ID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return d.CreateLabel(ctx, labelModel)
		}
		return fmt.Errorf("failed to query label: %w", err)
	}

	// Update existing label
	_, err = d.client.Label.UpdateOneID(labelModel.ID).
		SetName(labelModel.Name).
		SetColor(labelModel.Color).
		SetDescription(labelModel.Description).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update label: %w", err)
	}

	return nil
}

// CreateMilestone creates a new milestone
func (d *PostgresDAO) CreateMilestone(ctx context.Context, milestoneModel *MilestoneModel) error {
	creator := d.client.Milestone.Create().
		SetID(milestoneModel.ID).
		SetNumber(milestoneModel.Number).
		SetTitle(milestoneModel.Title).
		SetDescription(milestoneModel.Description).
		SetState(milestoneModel.State).
		SetCreatedAt(milestoneModel.CreatedAt).
		SetUpdatedAt(milestoneModel.UpdatedAt)

	if milestoneModel.DueOn != nil {
		creator.SetDueOn(*milestoneModel.DueOn)
	}

	if _, err := creator.Save(ctx); err != nil {
		return fmt.Errorf("failed to create milestone: %w", err)
	}

	return nil
}

// GetMilestoneByID retrieves a milestone by ID
func (d *PostgresDAO) GetMilestoneByID(ctx context.Context, id int64) (*MilestoneModel, error) {
	m, err := d.client.Milestone.Query().Where(milestone.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("milestone not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query milestone: %w", err)
	}

	return d.entMilestoneToModel(m), nil
}

// UpsertMilestone creates or updates a milestone
func (d *PostgresDAO) UpsertMilestone(ctx context.Context, milestoneModel *MilestoneModel) error {
	_, err := d.client.Milestone.Query().Where(milestone.ID(milestoneModel.ID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return d.CreateMilestone(ctx, milestoneModel)
		}
		return fmt.Errorf("failed to query milestone: %w", err)
	}

	// Update existing milestone
	updater := d.client.Milestone.UpdateOneID(milestoneModel.ID).
		SetNumber(milestoneModel.Number).
		SetTitle(milestoneModel.Title).
		SetDescription(milestoneModel.Description).
		SetState(milestoneModel.State).
		SetUpdatedAt(milestoneModel.UpdatedAt)

	if milestoneModel.DueOn != nil {
		updater.SetDueOn(*milestoneModel.DueOn)
	} else {
		updater.ClearDueOn()
	}

	if _, err := updater.Save(ctx); err != nil {
		return fmt.Errorf("failed to update milestone: %w", err)
	}

	return nil
}

// Helper functions to convert ent entities to models

func (d *PostgresDAO) entIssueToModel(iss *ent.Issue) *IssueModel {
	model := &IssueModel{
		ID:        iss.ID,
		Number:    iss.Number,
		Title:     iss.Title,
		Body:      iss.Body,
		State:     iss.State,
		Comments:  iss.Comments,
		HTMLURL:   iss.HTMLURL,
		Locked:    iss.Locked,
		CreatedAt: iss.CreatedAt,
		UpdatedAt: iss.UpdatedAt,
		UserID:    iss.UserID,
	}

	if iss.ClosedAt != nil {
		model.ClosedAt = iss.ClosedAt
	}

	if iss.MilestoneID != nil {
		model.MilestoneID = iss.MilestoneID
	}

	if iss.Edges.Labels != nil {
		model.Labels = make([]int64, len(iss.Edges.Labels))
		for i, l := range iss.Edges.Labels {
			model.Labels[i] = l.ID
		}
	}

	if iss.Edges.Assignees != nil {
		model.Assignees = make([]int64, len(iss.Edges.Assignees))
		for i, a := range iss.Edges.Assignees {
			model.Assignees[i] = a.ID
		}
	}

	return model
}

func (d *PostgresDAO) entMilestoneToModel(m *ent.Milestone) *MilestoneModel {
	model := &MilestoneModel{
		ID:          m.ID,
		Number:      m.Number,
		Title:       m.Title,
		Description: m.Description,
		State:       m.State,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	if m.DueOn != nil {
		model.DueOn = m.DueOn
	}

	return model
}
