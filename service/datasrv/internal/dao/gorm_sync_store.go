package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	AISummary     string `gorm:"type:text"`
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

type gormManagedRepo struct {
	Repo      string `gorm:"primaryKey;size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time `gorm:"index"`
}

func (gormManagedRepo) TableName() string { return "github_sync_repos" }

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

type gormBlogPost struct {
	ID           string    `gorm:"primaryKey;size:64"`
	Title        string    `gorm:"type:text;not null"`
	Slug         string    `gorm:"size:255;not null;uniqueIndex"`
	Summary      string    `gorm:"type:text"`
	Content      string    `gorm:"type:text"`
	TagsJSON     string    `gorm:"type:text"`
	Status       string    `gorm:"size:32;index;not null"`
	CommentCount int32     `gorm:"not null"`
	CreatedAt    time.Time `gorm:"index"`
	UpdatedAt    time.Time `gorm:"index"`
	PublishedAt  time.Time `gorm:"index"`
}

func (gormBlogPost) TableName() string { return "blog_posts" }

type gormBlogComment struct {
	ID          string    `gorm:"primaryKey;size:64"`
	PostID      string    `gorm:"index:idx_blog_comments_post_created;size:64;not null"`
	AuthorName  string    `gorm:"size:255;not null"`
	AuthorEmail string    `gorm:"size:255"`
	Content     string    `gorm:"type:text;not null"`
	Status      string    `gorm:"size:32;index;not null"`
	CreatedAt   time.Time `gorm:"index:idx_blog_comments_post_created"`
	UpdatedAt   time.Time `gorm:"index"`
}

func (gormBlogComment) TableName() string { return "blog_comments" }

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

	if err := db.AutoMigrate(&gormIssue{}, &gormCheckpoint{}, &gormManagedRepo{}, &gormFeedSource{}, &gormFeedContent{}, &gormFeedCheckpoint{}, &gormBlogPost{}, &gormBlogComment{}); err != nil {
		return nil, fmt.Errorf("gorm automigrate: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_github_issues_repo_updated_at ON github_issues (repo, updated_at DESC)").Error; err != nil {
		return nil, fmt.Errorf("create repo_updated_at index: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_rss_feed_contents_source_published_at ON rss_feed_contents (feed_source_id, published_at DESC, id ASC)").Error; err != nil {
		return nil, fmt.Errorf("create feed content index: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_blog_comments_post_status_created_at ON blog_comments (post_id, status, created_at ASC, id ASC)").Error; err != nil {
		return nil, fmt.Errorf("create blog comment index: %w", err)
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
			AISummary:     it.AISummary,
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

func (g *GormSyncStore) ListManagedRepos(ctx context.Context) ([]ManagedRepo, error) {
	var rows []gormManagedRepo
	if err := g.db.WithContext(ctx).Order("repo ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list managed repos: %w", err)
	}

	out := make([]ManagedRepo, 0, len(rows))
	for _, row := range rows {
		out = append(out, ManagedRepo{
			Repo:      row.Repo,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (g *GormSyncStore) ReplaceManagedRepos(ctx context.Context, repos []string) ([]ManagedRepo, error) {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("gorm begin replace managed repos: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	now := time.Now().UTC()
	if len(repos) > 0 {
		rows := make([]gormManagedRepo, 0, len(repos))
		for _, repo := range repos {
			rows = append(rows, gormManagedRepo{
				Repo:      repo,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repo"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).Create(&rows).Error; err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("gorm upsert managed repos: %w", err)
		}

		if err := tx.Where("repo NOT IN ?", repos).Delete(&gormManagedRepo{}).Error; err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("gorm delete stale managed repos: %w", err)
		}
	} else {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&gormManagedRepo{}).Error; err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("gorm clear managed repos: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("gorm commit managed repos: %w", err)
	}
	return g.ListManagedRepos(ctx)
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
	if err := query.
		Order("CASE WHEN ai_summary IS NOT NULL AND ai_summary <> '' THEN 0 ELSE 1 END ASC").
		Order("updated_at DESC").
		Find(&rows).Error; err != nil {
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
			AISummary:     row.AISummary,
			Raw:           row.Raw,
		})
	}
	return out, nil
}

func (g *GormSyncStore) UpdateIssueAISummary(ctx context.Context, repo string, issueID int64, number int32, summary string) (SyncedIssue, error) {
	query := g.db.WithContext(ctx).Model(&gormIssue{}).Where("repo = ?", repo)
	switch {
	case issueID > 0:
		query = query.Where("issue_id = ?", issueID)
	case number > 0:
		query = query.Where("number = ?", number)
	default:
		return SyncedIssue{}, ErrIssueNotFound
	}

	result := query.Update("ai_summary", summary)
	if result.Error != nil {
		return SyncedIssue{}, fmt.Errorf("gorm update ai_summary: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return SyncedIssue{}, ErrIssueNotFound
	}

	rows, err := g.ListIssues(ctx, SyncIssueFilter{Repo: repo, IssueID: issueID, Number: number, Limit: 1})
	if err != nil {
		return SyncedIssue{}, err
	}
	if len(rows) == 0 {
		return SyncedIssue{}, ErrIssueNotFound
	}
	return rows[0], nil
}

func (g *GormSyncStore) ClearIssueAISummaries(ctx context.Context, repo string) (int, error) {
	query := g.db.WithContext(ctx).Model(&gormIssue{})
	if repo = strings.TrimSpace(repo); repo != "" {
		query = query.Where("repo = ?", repo)
	}

	result := query.Update("ai_summary", "")
	if result.Error != nil {
		return 0, fmt.Errorf("gorm clear ai_summary: %w", result.Error)
	}
	return int(result.RowsAffected), nil
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

func (g *GormSyncStore) ListBlogPosts(ctx context.Context, filter BlogPostFilter) ([]BlogPost, error) {
	query := g.db.WithContext(ctx).Model(&gormBlogPost{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Tag != "" {
		tagPattern, err := json.Marshal(filter.Tag)
		if err != nil {
			return nil, fmt.Errorf("marshal blog tag: %w", err)
		}
		query = query.Where("tags_json LIKE ?", "%"+strings.Trim(string(tagPattern), "\"")+"%")
	}
	if queryText := strings.TrimSpace(filter.Query); queryText != "" {
		like := "%" + queryText + "%"
		query = query.Where("title ILIKE ? OR slug ILIKE ? OR summary ILIKE ? OR content ILIKE ?", like, like, like, like)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	var rows []gormBlogPost
	if err := query.Order("published_at DESC").Order("created_at DESC").Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list blog posts: %w", err)
	}

	out := make([]BlogPost, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBlogPost(row))
	}
	return out, nil
}

func (g *GormSyncStore) GetBlogPost(ctx context.Context, id string) (BlogPost, error) {
	var row gormBlogPost
	if err := g.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return BlogPost{}, ErrBlogPostNotFound
		}
		return BlogPost{}, fmt.Errorf("gorm get blog post: %w", err)
	}
	return toBlogPost(row), nil
}

func (g *GormSyncStore) GetBlogPostBySlug(ctx context.Context, slug string) (BlogPost, error) {
	var row gormBlogPost
	if err := g.db.WithContext(ctx).Where("slug = ?", slug).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return BlogPost{}, ErrBlogPostNotFound
		}
		return BlogPost{}, fmt.Errorf("gorm get blog post by slug: %w", err)
	}
	return toBlogPost(row), nil
}

func (g *GormSyncStore) CreateBlogPost(ctx context.Context, post BlogPost) (BlogPost, error) {
	now := time.Now().UTC()
	if post.ID == "" {
		return BlogPost{}, fmt.Errorf("blog post id is empty")
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	post.UpdatedAt = now
	row, err := toGormBlogPost(post)
	if err != nil {
		return BlogPost{}, err
	}
	if err := g.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return BlogPost{}, ErrBlogPostSlugConflict
		}
		return BlogPost{}, fmt.Errorf("gorm create blog post: %w", err)
	}
	return toBlogPost(row), nil
}

func (g *GormSyncStore) UpdateBlogPost(ctx context.Context, post BlogPost) (BlogPost, error) {
	var existing gormBlogPost
	if err := g.db.WithContext(ctx).Where("id = ?", post.ID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return BlogPost{}, ErrBlogPostNotFound
		}
		return BlogPost{}, fmt.Errorf("gorm load blog post before update: %w", err)
	}

	post.CreatedAt = existing.CreatedAt
	post.CommentCount = existing.CommentCount
	post.UpdatedAt = time.Now().UTC()
	row, err := toGormBlogPost(post)
	if err != nil {
		return BlogPost{}, err
	}
	if err := g.db.WithContext(ctx).Model(&gormBlogPost{}).Where("id = ?", post.ID).Updates(map[string]any{
		"title":         row.Title,
		"slug":          row.Slug,
		"summary":       row.Summary,
		"content":       row.Content,
		"tags_json":     row.TagsJSON,
		"status":        row.Status,
		"updated_at":    row.UpdatedAt,
		"published_at":  row.PublishedAt,
		"comment_count": row.CommentCount,
	}).Error; err != nil {
		if isUniqueViolation(err) {
			return BlogPost{}, ErrBlogPostSlugConflict
		}
		return BlogPost{}, fmt.Errorf("gorm update blog post: %w", err)
	}
	return g.GetBlogPost(ctx, post.ID)
}

func (g *GormSyncStore) DeleteBlogPost(ctx context.Context, id string) error {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("gorm begin delete blog post: %w", tx.Error)
	}
	if err := tx.Where("post_id = ?", id).Delete(&gormBlogComment{}).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("gorm delete blog comments: %w", err)
	}
	result := tx.Where("id = ?", id).Delete(&gormBlogPost{})
	if result.Error != nil {
		_ = tx.Rollback()
		return fmt.Errorf("gorm delete blog post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		_ = tx.Rollback()
		return ErrBlogPostNotFound
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("gorm commit delete blog post: %w", err)
	}
	return nil
}

func (g *GormSyncStore) ListBlogComments(ctx context.Context, filter BlogCommentFilter) ([]BlogComment, error) {
	query := g.db.WithContext(ctx).Model(&gormBlogComment{})
	if filter.PostID != "" {
		query = query.Where("post_id = ?", filter.PostID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	var rows []gormBlogComment
	if err := query.Order("created_at ASC").Order("id ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list blog comments: %w", err)
	}
	out := make([]BlogComment, 0, len(rows))
	for _, row := range rows {
		comment, err := g.toBlogComment(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, comment)
	}
	return out, nil
}

func (g *GormSyncStore) GetBlogComment(ctx context.Context, id string) (BlogComment, error) {
	var row gormBlogComment
	if err := g.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return BlogComment{}, ErrBlogCommentNotFound
		}
		return BlogComment{}, fmt.Errorf("gorm get blog comment: %w", err)
	}
	return g.toBlogComment(ctx, row)
}

func (g *GormSyncStore) CreateBlogComment(ctx context.Context, comment BlogComment) (BlogComment, error) {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return BlogComment{}, fmt.Errorf("gorm begin create blog comment: %w", tx.Error)
	}

	var post gormBlogPost
	if err := tx.Where("id = ?", comment.PostID).First(&post).Error; err != nil {
		_ = tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return BlogComment{}, ErrBlogCommentPostAbsent
		}
		return BlogComment{}, fmt.Errorf("gorm load blog post before create comment: %w", err)
	}

	now := time.Now().UTC()
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = now
	}
	comment.UpdatedAt = now
	row := gormBlogComment{
		ID:          comment.ID,
		PostID:      comment.PostID,
		AuthorName:  comment.AuthorName,
		AuthorEmail: comment.AuthorEmail,
		Content:     comment.Content,
		Status:      comment.Status,
		CreatedAt:   comment.CreatedAt,
		UpdatedAt:   comment.UpdatedAt,
	}
	if err := tx.Create(&row).Error; err != nil {
		_ = tx.Rollback()
		return BlogComment{}, fmt.Errorf("gorm create blog comment: %w", err)
	}
	if err := tx.Model(&gormBlogPost{}).Where("id = ?", post.ID).Updates(map[string]any{
		"comment_count": gorm.Expr("comment_count + ?", 1),
		"updated_at":    now,
	}).Error; err != nil {
		_ = tx.Rollback()
		return BlogComment{}, fmt.Errorf("gorm update blog post comment count: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return BlogComment{}, fmt.Errorf("gorm commit create blog comment: %w", err)
	}
	comment.PostSlug = post.Slug
	return comment, nil
}

func (g *GormSyncStore) UpdateBlogComment(ctx context.Context, comment BlogComment) (BlogComment, error) {
	var existing gormBlogComment
	if err := g.db.WithContext(ctx).Where("id = ?", comment.ID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return BlogComment{}, ErrBlogCommentNotFound
		}
		return BlogComment{}, fmt.Errorf("gorm load blog comment before update: %w", err)
	}
	comment.PostID = existing.PostID
	comment.CreatedAt = existing.CreatedAt
	comment.UpdatedAt = time.Now().UTC()
	if err := g.db.WithContext(ctx).Model(&gormBlogComment{}).Where("id = ?", comment.ID).Updates(map[string]any{
		"author_name":  comment.AuthorName,
		"author_email": comment.AuthorEmail,
		"content":      comment.Content,
		"status":       comment.Status,
		"updated_at":   comment.UpdatedAt,
	}).Error; err != nil {
		return BlogComment{}, fmt.Errorf("gorm update blog comment: %w", err)
	}
	return g.GetBlogComment(ctx, comment.ID)
}

func (g *GormSyncStore) DeleteBlogComment(ctx context.Context, id string) error {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("gorm begin delete blog comment: %w", tx.Error)
	}
	var row gormBlogComment
	if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
		_ = tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return ErrBlogCommentNotFound
		}
		return fmt.Errorf("gorm load blog comment before delete: %w", err)
	}
	if err := tx.Where("id = ?", id).Delete(&gormBlogComment{}).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("gorm delete blog comment: %w", err)
	}
	if err := tx.Model(&gormBlogPost{}).Where("id = ?", row.PostID).Updates(map[string]any{
		"comment_count": gorm.Expr("GREATEST(comment_count - ?, 0)", 1),
		"updated_at":    time.Now().UTC(),
	}).Error; err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("gorm decrement blog post comment count: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("gorm commit delete blog comment: %w", err)
	}
	return nil
}

func toGormBlogPost(post BlogPost) (gormBlogPost, error) {
	tagsJSON, err := json.Marshal(post.Tags)
	if err != nil {
		return gormBlogPost{}, fmt.Errorf("marshal blog tags: %w", err)
	}
	return gormBlogPost{
		ID:           post.ID,
		Title:        post.Title,
		Slug:         post.Slug,
		Summary:      post.Summary,
		Content:      post.Content,
		TagsJSON:     string(tagsJSON),
		Status:       post.Status,
		CommentCount: post.CommentCount,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
		PublishedAt:  post.PublishedAt,
	}, nil
}

func toBlogPost(row gormBlogPost) BlogPost {
	var tags []string
	_ = json.Unmarshal([]byte(row.TagsJSON), &tags)
	return BlogPost{
		ID:           row.ID,
		Title:        row.Title,
		Slug:         row.Slug,
		Summary:      row.Summary,
		Content:      row.Content,
		Tags:         tags,
		Status:       row.Status,
		CommentCount: row.CommentCount,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		PublishedAt:  row.PublishedAt,
	}
}

func (g *GormSyncStore) toBlogComment(ctx context.Context, row gormBlogComment) (BlogComment, error) {
	post, err := g.GetBlogPost(ctx, row.PostID)
	if err != nil {
		if err == ErrBlogPostNotFound {
			return BlogComment{}, ErrBlogCommentPostAbsent
		}
		return BlogComment{}, err
	}
	return BlogComment{
		ID:          row.ID,
		PostID:      row.PostID,
		PostSlug:    post.Slug,
		AuthorName:  row.AuthorName,
		AuthorEmail: row.AuthorEmail,
		Content:     row.Content,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value") ||
		strings.Contains(strings.ToLower(err.Error()), "unique constraint")
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
