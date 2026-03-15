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

type gormFeedSource struct {
	ID            string    `gorm:"primaryKey;size:64"`
	URL           string    `gorm:"size:2048;not null;uniqueIndex"`
	DisplayName   string    `gorm:"size:512"`
	Description   string    `gorm:"type:text"`
	SiteURL       string    `gorm:"size:2048"`
	Enabled       bool      `gorm:"not null"`
	ETag          string    `gorm:"size:512"`
	LastModified  string    `gorm:"size:512"`
	LastSyncedAt  time.Time `gorm:"index"`
	LastSuccessAt time.Time `gorm:"index"`
	LastRunStatus string    `gorm:"size:32"`
	LastError     string    `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time `gorm:"index"`
}

func (gormFeedSource) TableName() string { return "rss_feed_sources" }

type gormFeedContent struct {
	ID             string    `gorm:"primaryKey;size:80"`
	FeedSourceID   string    `gorm:"index:idx_feed_content_identity,unique;size:64;not null"`
	Identity       string    `gorm:"index:idx_feed_content_identity,unique;size:2048;not null"`
	GUID           string    `gorm:"size:2048"`
	Title          string    `gorm:"type:text"`
	Summary        string    `gorm:"type:text"`
	Content        string    `gorm:"type:text"`
	Link           string    `gorm:"size:2048"`
	Author         string    `gorm:"size:512"`
	CategoriesJSON string    `gorm:"type:text"`
	PublishedAt    time.Time `gorm:"index:idx_feed_source_published"`
	UpdatedAt      time.Time
	FetchedAt      time.Time
}

func (gormFeedContent) TableName() string { return "rss_feed_contents" }

type gormFeedCheckpoint struct {
	FeedSourceID  string `gorm:"primaryKey;size:64"`
	LastSyncedAt  time.Time
	LastSuccessAt time.Time
	LastRunStatus string `gorm:"size:32"`
	LastError     string `gorm:"type:text"`
	ETag          string `gorm:"size:512"`
	LastModified  string `gorm:"size:512"`
	UpdatedAt     time.Time
}

func (gormFeedCheckpoint) TableName() string { return "rss_feed_checkpoints" }

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

	if err := db.AutoMigrate(&gormIssue{}, &gormCheckpoint{}, &gormFeedSource{}, &gormFeedContent{}, &gormFeedCheckpoint{}); err != nil {
		return nil, fmt.Errorf("gorm automigrate: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_github_issues_repo_updated_at ON github_issues (repo, updated_at DESC)").Error; err != nil {
		return nil, fmt.Errorf("create repo_updated_at index: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_rss_feed_contents_source_published_at ON rss_feed_contents (feed_source_id, published_at DESC, id ASC)").Error; err != nil {
		return nil, fmt.Errorf("create feed content index: %w", err)
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
	if filter.State != "" && filter.State != "all" {
		query = query.Where("state = ?", filter.State)
	}
	if filter.IssueID > 0 {
		query = query.Where("issue_id = ?", filter.IssueID)
	}
	if filter.Number > 0 {
		query = query.Where("number = ?", filter.Number)
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

func (g *GormSyncStore) UpsertFeedSource(ctx context.Context, source FeedSource) (FeedSource, error) {
	now := time.Now().UTC()
	if source.ID == "" {
		return FeedSource{}, fmt.Errorf("feed source id is empty")
	}
	if source.CreatedAt.IsZero() {
		existing, err := g.GetFeedSource(ctx, source.ID)
		if err == nil {
			source.CreatedAt = existing.CreatedAt
		}
	}
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now

	row := gormFeedSource{
		ID:            source.ID,
		URL:           source.URL,
		DisplayName:   source.DisplayName,
		Description:   source.Description,
		SiteURL:       source.SiteURL,
		Enabled:       source.Enabled,
		ETag:          source.ETag,
		LastModified:  source.LastModified,
		LastSyncedAt:  source.LastSyncedAt,
		LastSuccessAt: source.LastSuccessAt,
		LastRunStatus: source.LastRunStatus,
		LastError:     source.LastError,
		CreatedAt:     source.CreatedAt,
		UpdatedAt:     source.UpdatedAt,
	}

	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"url", "display_name", "description", "site_url", "enabled", "etag", "last_modified",
				"last_synced_at", "last_success_at", "last_run_status", "last_error", "updated_at",
			}),
		}).
		Create(&row).Error
	if err != nil {
		return FeedSource{}, fmt.Errorf("gorm upsert feed source: %w", err)
	}
	source.CreatedAt = row.CreatedAt
	return source, nil
}

func (g *GormSyncStore) GetFeedSource(ctx context.Context, id string) (FeedSource, error) {
	var row gormFeedSource
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FeedSource{}, ErrFeedSourceNotFound
		}
		return FeedSource{}, fmt.Errorf("gorm get feed source: %w", err)
	}
	return FeedSource{
		ID:            row.ID,
		URL:           row.URL,
		DisplayName:   row.DisplayName,
		Description:   row.Description,
		SiteURL:       row.SiteURL,
		Enabled:       row.Enabled,
		ETag:          row.ETag,
		LastModified:  row.LastModified,
		LastSyncedAt:  row.LastSyncedAt,
		LastSuccessAt: row.LastSuccessAt,
		LastRunStatus: row.LastRunStatus,
		LastError:     row.LastError,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (g *GormSyncStore) ListFeedSources(ctx context.Context, filter FeedSourceFilter) ([]FeedSource, error) {
	query := g.db.WithContext(ctx).Model(&gormFeedSource{}).Order("created_at ASC").Order("id ASC")
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var rows []gormFeedSource
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list feed sources: %w", err)
	}
	out := make([]FeedSource, 0, len(rows))
	for _, row := range rows {
		out = append(out, FeedSource{
			ID:            row.ID,
			URL:           row.URL,
			DisplayName:   row.DisplayName,
			Description:   row.Description,
			SiteURL:       row.SiteURL,
			Enabled:       row.Enabled,
			ETag:          row.ETag,
			LastModified:  row.LastModified,
			LastSyncedAt:  row.LastSyncedAt,
			LastSuccessAt: row.LastSuccessAt,
			LastRunStatus: row.LastRunStatus,
			LastError:     row.LastError,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	return out, nil
}

func (g *GormSyncStore) DeleteFeedSource(ctx context.Context, id string) error {
	if err := g.db.WithContext(ctx).Where("feed_source_id = ?", id).Delete(&gormFeedContent{}).Error; err != nil {
		return fmt.Errorf("gorm delete feed contents: %w", err)
	}
	if err := g.db.WithContext(ctx).Where("feed_source_id = ?", id).Delete(&gormFeedCheckpoint{}).Error; err != nil {
		return fmt.Errorf("gorm delete feed checkpoint: %w", err)
	}
	result := g.db.WithContext(ctx).Where("id = ?", id).Delete(&gormFeedSource{})
	if result.Error != nil {
		return fmt.Errorf("gorm delete feed source: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrFeedSourceNotFound
	}
	return nil
}

func (g *GormSyncStore) UpsertFeedContents(ctx context.Context, sourceID string, contents []FeedContent) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	rows := make([]gormFeedContent, 0, len(contents))
	for _, content := range contents {
		categoriesJSON, _ := json.Marshal(content.Categories)
		rows = append(rows, gormFeedContent{
			ID:             content.ID,
			FeedSourceID:   sourceID,
			Identity:       content.Identity,
			GUID:           content.GUID,
			Title:          content.Title,
			Summary:        content.Summary,
			Content:        content.Content,
			Link:           content.Link,
			Author:         content.Author,
			CategoriesJSON: string(categoriesJSON),
			PublishedAt:    content.PublishedAt,
			UpdatedAt:      content.UpdatedAt,
			FetchedAt:      content.FetchedAt,
		})
	}
	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "feed_source_id"}, {Name: "identity"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"id", "guid", "title", "summary", "content", "link", "author", "categories_json",
				"published_at", "updated_at", "fetched_at",
			}),
		}).
		Create(&rows).Error
	if err != nil {
		return 0, fmt.Errorf("gorm upsert feed contents: %w", err)
	}
	return len(contents), nil
}

func (g *GormSyncStore) ListFeedContents(ctx context.Context, filter FeedContentFilter) ([]FeedContent, error) {
	query := g.db.WithContext(ctx).Model(&gormFeedContent{}).Order("published_at DESC").Order("id ASC")
	if filter.FeedSourceID != "" {
		query = query.Where("feed_source_id = ?", filter.FeedSourceID)
	}
	if filter.ContentID != "" {
		query = query.Where("id = ?", filter.ContentID)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var rows []gormFeedContent
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list feed contents: %w", err)
	}
	out := make([]FeedContent, 0, len(rows))
	for _, row := range rows {
		var categories []string
		_ = json.Unmarshal([]byte(row.CategoriesJSON), &categories)
		out = append(out, FeedContent{
			ID:           row.ID,
			FeedSourceID: row.FeedSourceID,
			Identity:     row.Identity,
			GUID:         row.GUID,
			Title:        row.Title,
			Summary:      row.Summary,
			Content:      row.Content,
			Link:         row.Link,
			Author:       row.Author,
			Categories:   categories,
			PublishedAt:  row.PublishedAt,
			UpdatedAt:    row.UpdatedAt,
			FetchedAt:    row.FetchedAt,
		})
	}
	return out, nil
}

func (g *GormSyncStore) GetFeedContent(ctx context.Context, id string) (FeedContent, error) {
	rows, err := g.ListFeedContents(ctx, FeedContentFilter{ContentID: id, Limit: 1})
	if err != nil {
		return FeedContent{}, err
	}
	if len(rows) == 0 {
		return FeedContent{}, ErrFeedContentNotFound
	}
	return rows[0], nil
}

func (g *GormSyncStore) GetFeedCheckpoint(ctx context.Context, sourceID string) (FeedCheckpoint, error) {
	var row gormFeedCheckpoint
	err := g.db.WithContext(ctx).Where("feed_source_id = ?", sourceID).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FeedCheckpoint{FeedSourceID: sourceID}, nil
		}
		return FeedCheckpoint{}, fmt.Errorf("gorm get feed checkpoint: %w", err)
	}
	return FeedCheckpoint{
		FeedSourceID:  row.FeedSourceID,
		LastSyncedAt:  row.LastSyncedAt,
		LastSuccessAt: row.LastSuccessAt,
		LastRunStatus: row.LastRunStatus,
		LastError:     row.LastError,
		ETag:          row.ETag,
		LastModified:  row.LastModified,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (g *GormSyncStore) SaveFeedCheckpoint(ctx context.Context, checkpoint FeedCheckpoint) error {
	checkpoint.UpdatedAt = time.Now().UTC()
	row := gormFeedCheckpoint{
		FeedSourceID:  checkpoint.FeedSourceID,
		LastSyncedAt:  checkpoint.LastSyncedAt,
		LastSuccessAt: checkpoint.LastSuccessAt,
		LastRunStatus: checkpoint.LastRunStatus,
		LastError:     checkpoint.LastError,
		ETag:          checkpoint.ETag,
		LastModified:  checkpoint.LastModified,
		UpdatedAt:     checkpoint.UpdatedAt,
	}
	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "feed_source_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"last_synced_at", "last_success_at", "last_run_status", "last_error", "etag", "last_modified", "updated_at",
			}),
		}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("gorm save feed checkpoint: %w", err)
	}
	return nil
}
