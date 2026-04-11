package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormPRReview struct {
	ID            uint      `gorm:"primaryKey"`
	Repo          string    `gorm:"index:idx_pr_review_repo_issue,unique;size:255;not null"`
	IssueID       int64     `gorm:"index:idx_pr_review_repo_issue,unique;not null"`
	Number        int32     `gorm:"not null"`
	ReviewSummary string    `gorm:"type:text"`
	RiskAreas     string    `gorm:"type:text"`
	Suggestions   string    `gorm:"type:text"`
	RawDiffSize   int       `gorm:"not null"`
	ModelUsed     string    `gorm:"size:255"`
	CreatedAt     time.Time `gorm:"index"`
	UpdatedAt     time.Time `gorm:"index"`
}

func (gormPRReview) TableName() string { return "pr_ai_reviews" }

func (g *GormSyncStore) UpsertPRReview(ctx context.Context, review PRReview) error {
	now := time.Now().UTC()
	row := gormPRReview{
		Repo:          review.Repo,
		IssueID:       review.IssueID,
		Number:        review.Number,
		ReviewSummary: review.ReviewSummary,
		RiskAreas:     review.RiskAreas,
		Suggestions:   review.Suggestions,
		RawDiffSize:   review.RawDiffSize,
		ModelUsed:     review.ModelUsed,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := g.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "repo"}, {Name: "issue_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"number", "review_summary", "risk_areas", "suggestions",
				"raw_diff_size", "model_used", "updated_at",
			}),
		}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("gorm upsert pr review: %w", err)
	}
	return nil
}

func (g *GormSyncStore) GetPRReview(ctx context.Context, repo string, number int32) (PRReview, error) {
	var row gormPRReview
	err := g.db.WithContext(ctx).
		Where("repo = ? AND number = ?", repo, number).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return PRReview{}, ErrPRReviewNotFound
		}
		return PRReview{}, fmt.Errorf("gorm get pr review: %w", err)
	}
	return toPRReview(row), nil
}

func (g *GormSyncStore) ListPRReviews(ctx context.Context, filter PRReviewFilter) ([]PRReview, error) {
	query := g.db.WithContext(ctx).Model(&gormPRReview{})
	if filter.Repo != "" {
		query = query.Where("repo = ?", filter.Repo)
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

	var rows []gormPRReview
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list pr reviews: %w", err)
	}

	out := make([]PRReview, 0, len(rows))
	for _, row := range rows {
		out = append(out, toPRReview(row))
	}
	return out, nil
}

func (g *GormSyncStore) ListUnreviewedPRs(ctx context.Context, repos []string, limit int) ([]SyncedIssue, error) {
	if len(repos) == 0 {
		return nil, nil
	}

	query := g.db.WithContext(ctx).Model(&gormIssue{}).
		Where("is_pull_request = ? AND repo IN ?", true, repos).
		Where("NOT EXISTS (SELECT 1 FROM pr_ai_reviews r WHERE r.repo = github_issues.repo AND r.issue_id = github_issues.issue_id)")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []gormIssue
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("gorm list unreviewed prs: %w", err)
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

func toPRReview(row gormPRReview) PRReview {
	return PRReview{
		Repo:          row.Repo,
		IssueID:       row.IssueID,
		Number:        row.Number,
		ReviewSummary: row.ReviewSummary,
		RiskAreas:     row.RiskAreas,
		Suggestions:   row.Suggestions,
		RawDiffSize:   row.RawDiffSize,
		ModelUsed:     row.ModelUsed,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
