package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormIssue struct {
	ID            uint      `gorm:"primaryKey"`
	Repo          string    `gorm:"index:idx_repo_issue,unique;size:255;not null"`
	IssueID       int64     `gorm:"index:idx_repo_issue,unique;not null"`
	Number        int32     `gorm:"not null"`
	Title         string    `gorm:"type:text"`
	Body          string    `gorm:"type:text"`
	State         string    `gorm:"size:32;index"`
	Author        string    `gorm:"size:255"`
	AssigneesJSON string    `gorm:"type:text"`
	LabelsJSON    string    `gorm:"type:text"`
	Comments      int32     `gorm:"not null"`
	IsPullRequest bool      `gorm:"not null"`
	HTMLURL       string    `gorm:"size:1024"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time `gorm:"index"`
	ClosedAt      *time.Time
	Raw           string `gorm:"type:text"`
}

func (gormIssue) TableName() string { return "github_issues" }

type gormCheckpoint struct {
	Repo               string `gorm:"primaryKey;size:255"`
	LastSyncedAt       time.Time
	LastIssueUpdatedAt time.Time
	LastRunStatus      string `gorm:"size:32"`
	LastError          string `gorm:"type:text"`
	UpdatedAt          time.Time
}

func (gormCheckpoint) TableName() string { return "github_issue_checkpoints" }

// GormSyncStore stores synced issue data in PostgreSQL via GORM.
type GormSyncStore struct {
	db *gorm.DB
}

func NewGormSyncStore(dsn string) (*GormSyncStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is empty")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm postgres: %w", err)
	}

	if err := db.AutoMigrate(&gormIssue{}, &gormCheckpoint{}); err != nil {
		return nil, fmt.Errorf("gorm automigrate: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_github_issues_repo_updated_at ON github_issues (repo, updated_at DESC)").Error; err != nil {
		return nil, fmt.Errorf("create repo_updated_at index: %w", err)
	}

	return &GormSyncStore{db: db}, nil
}

func (g *GormSyncStore) UpsertIssues(ctx context.Context, repo string, issues []SyncedIssue) (int, error) {
	if len(issues) == 0 {
		return 0, nil
	}

	rows := make([]gormIssue, 0, len(issues))
	for _, it := range issues {
		assigneesJSON, _ := json.Marshal(it.Assignees)
		labelsJSON, _ := json.Marshal(it.Labels)
		rows = append(rows, gormIssue{
			Repo:          repo,
			IssueID:       it.IssueID,
			Number:        it.Number,
			Title:         it.Title,
			Body:          it.Body,
			State:         it.State,
			Author:        it.Author,
			AssigneesJSON: string(assigneesJSON),
			LabelsJSON:    string(labelsJSON),
			Comments:      it.Comments,
			IsPullRequest: it.IsPullRequest,
			HTMLURL:       it.HTMLURL,
			CreatedAt:     it.CreatedAt,
			UpdatedAt:     it.UpdatedAt,
			ClosedAt:      it.ClosedAt,
			Raw:           it.Raw,
		})
	}

	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "repo"}, {Name: "issue_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"number", "title", "body", "state", "author", "assignees_json", "labels_json",
				"comments", "is_pull_request", "html_url", "created_at", "updated_at", "closed_at", "raw",
			}),
		}).
		Create(&rows).Error
	if err != nil {
		return 0, fmt.Errorf("gorm upsert issues: %w", err)
	}

	return len(issues), nil
}

func (g *GormSyncStore) ListIssues(ctx context.Context, filter SyncIssueFilter) ([]SyncedIssue, error) {
	query := g.db.WithContext(ctx).Model(&gormIssue{})
	if filter.Repo != "" {
		query = query.Where("repo = ?", filter.Repo)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	var rows []gormIssue
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list issues: %w", err)
	}

	out := make([]SyncedIssue, 0, len(rows))
	for _, row := range rows {
		var assignees []string
		var labels []string
		_ = json.Unmarshal([]byte(row.AssigneesJSON), &assignees)
		_ = json.Unmarshal([]byte(row.LabelsJSON), &labels)
		out = append(out, SyncedIssue{
			Repo:          row.Repo,
			IssueID:       row.IssueID,
			Number:        row.Number,
			Title:         row.Title,
			Body:          row.Body,
			State:         row.State,
			Author:        row.Author,
			Assignees:     assignees,
			Labels:        labels,
			Comments:      row.Comments,
			IsPullRequest: row.IsPullRequest,
			HTMLURL:       row.HTMLURL,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			ClosedAt:      row.ClosedAt,
			Raw:           row.Raw,
		})
	}
	return out, nil
}

func (g *GormSyncStore) GetRepoCheckpoint(ctx context.Context, repo string) (Checkpoint, error) {
	var row gormCheckpoint
	err := g.db.WithContext(ctx).Where("repo = ?", repo).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return Checkpoint{Repo: repo}, nil
		}
		return Checkpoint{}, fmt.Errorf("gorm get checkpoint: %w", err)
	}

	return Checkpoint{
		Repo:               row.Repo,
		LastSyncedAt:       row.LastSyncedAt,
		LastIssueUpdatedAt: row.LastIssueUpdatedAt,
		LastRunStatus:      row.LastRunStatus,
		LastError:          row.LastError,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

func (g *GormSyncStore) SaveRepoCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	checkpoint.UpdatedAt = time.Now()
	row := gormCheckpoint{
		Repo:               checkpoint.Repo,
		LastSyncedAt:       checkpoint.LastSyncedAt,
		LastIssueUpdatedAt: checkpoint.LastIssueUpdatedAt,
		LastRunStatus:      checkpoint.LastRunStatus,
		LastError:          checkpoint.LastError,
		UpdatedAt:          checkpoint.UpdatedAt,
	}

	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repo"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_synced_at", "last_issue_updated_at", "last_run_status", "last_error", "updated_at"}),
		}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("gorm save checkpoint: %w", err)
	}
	return nil
}

func (g *GormSyncStore) ListCheckpoints(ctx context.Context) ([]Checkpoint, error) {
	var rows []gormCheckpoint
	if err := g.db.WithContext(ctx).Order("repo ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list checkpoints: %w", err)
	}

	out := make([]Checkpoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, Checkpoint{
			Repo:               row.Repo,
			LastSyncedAt:       row.LastSyncedAt,
			LastIssueUpdatedAt: row.LastIssueUpdatedAt,
			LastRunStatus:      row.LastRunStatus,
			LastError:          row.LastError,
			UpdatedAt:          row.UpdatedAt,
		})
	}
	return out, nil
}

func (g *GormSyncStore) Close() error {
	sqlDB, err := g.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
