package dao

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoIssueDoc struct {
	Repo          string     `bson:"repo"`
	IssueID       int64      `bson:"issue_id"`
	Number        int32      `bson:"number"`
	Title         string     `bson:"title"`
	Body          string     `bson:"body"`
	State         string     `bson:"state"`
	Author        string     `bson:"author"`
	Assignees     []string   `bson:"assignees"`
	Labels        []string   `bson:"labels"`
	Comments      int32      `bson:"comments"`
	IsPullRequest bool       `bson:"is_pull_request"`
	HTMLURL       string     `bson:"html_url"`
	CreatedAt     time.Time  `bson:"created_at"`
	UpdatedAt     time.Time  `bson:"updated_at"`
	ClosedAt      *time.Time `bson:"closed_at,omitempty"`
	Raw           string     `bson:"raw,omitempty"`
}

type mongoCheckpointDoc struct {
	Repo               string    `bson:"repo"`
	LastSyncedAt       time.Time `bson:"last_synced_at"`
	LastIssueUpdatedAt time.Time `bson:"last_issue_updated_at"`
	LastRunStatus      string    `bson:"last_run_status"`
	LastError          string    `bson:"last_error"`
	UpdatedAt          time.Time `bson:"updated_at"`
}

type mongoFeedSourceDoc struct {
	ID            string    `bson:"id"`
	URL           string    `bson:"url"`
	DisplayName   string    `bson:"display_name"`
	Description   string    `bson:"description"`
	SiteURL       string    `bson:"site_url"`
	Enabled       bool      `bson:"enabled"`
	ETag          string    `bson:"etag"`
	LastModified  string    `bson:"last_modified"`
	LastSyncedAt  time.Time `bson:"last_synced_at"`
	LastSuccessAt time.Time `bson:"last_success_at"`
	LastRunStatus string    `bson:"last_run_status"`
	LastError     string    `bson:"last_error"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

type mongoFeedContentDoc struct {
	ID           string    `bson:"id"`
	FeedSourceID string    `bson:"feed_source_id"`
	Identity     string    `bson:"identity"`
	GUID         string    `bson:"guid"`
	Title        string    `bson:"title"`
	Summary      string    `bson:"summary"`
	Content      string    `bson:"content"`
	Link         string    `bson:"link"`
	Author       string    `bson:"author"`
	Categories   []string  `bson:"categories"`
	PublishedAt  time.Time `bson:"published_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
	FetchedAt    time.Time `bson:"fetched_at"`
}

type mongoFeedCheckpointDoc struct {
	FeedSourceID  string    `bson:"feed_source_id"`
	LastSyncedAt  time.Time `bson:"last_synced_at"`
	LastSuccessAt time.Time `bson:"last_success_at"`
	LastRunStatus string    `bson:"last_run_status"`
	LastError     string    `bson:"last_error"`
	ETag          string    `bson:"etag"`
	LastModified  string    `bson:"last_modified"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

// MongoSyncStore stores synced issue data in MongoDB.
type MongoSyncStore struct {
	client          *mongo.Client
	db              *mongo.Database
	issuesCol       *mongo.Collection
	checkpointC     *mongo.Collection
	feedSourceC     *mongo.Collection
	feedContentC    *mongo.Collection
	feedCheckpointC *mongo.Collection
}

func NewMongoSyncStore(uri, dbName string) (*MongoSyncStore, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongo uri is empty")
	}
	if dbName == "" {
		dbName = "datasrv"
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	db := client.Database(dbName)
	store := &MongoSyncStore{
		client:          client,
		db:              db,
		issuesCol:       db.Collection("github_issues"),
		checkpointC:     db.Collection("github_issue_checkpoints"),
		feedSourceC:     db.Collection("rss_feed_sources"),
		feedContentC:    db.Collection("rss_feed_contents"),
		feedCheckpointC: db.Collection("rss_feed_checkpoints"),
	}

	if err := store.ensureIndexes(context.Background()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return store, nil
}

func (m *MongoSyncStore) ensureIndexes(ctx context.Context) error {
	_, err := m.issuesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "repo", Value: 1}, {Key: "issue_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "repo", Value: 1}, {Key: "updated_at", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("create issues indexes: %w", err)
	}

	_, err = m.checkpointC.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "repo", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create checkpoint indexes: %w", err)
	}

	_, err = m.feedSourceC.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "url", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		return fmt.Errorf("create feed source indexes: %w", err)
	}

	_, err = m.feedContentC.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "feed_source_id", Value: 1}, {Key: "identity", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "feed_source_id", Value: 1}, {Key: "published_at", Value: -1}, {Key: "id", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("create feed content indexes: %w", err)
	}

	_, err = m.feedCheckpointC.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "feed_source_id", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create feed checkpoint indexes: %w", err)
	}
	return nil
}

func (m *MongoSyncStore) UpsertIssues(ctx context.Context, repo string, issues []SyncedIssue) (int, error) {
	if len(issues) == 0 {
		return 0, nil
	}

	models := make([]mongo.WriteModel, 0, len(issues))
	for _, it := range issues {
		filter := bson.M{"repo": repo, "issue_id": it.IssueID}
		set := bson.M{
			"repo":            repo,
			"issue_id":        it.IssueID,
			"number":          it.Number,
			"title":           it.Title,
			"body":            it.Body,
			"state":           it.State,
			"author":          it.Author,
			"assignees":       it.Assignees,
			"labels":          it.Labels,
			"comments":        it.Comments,
			"is_pull_request": it.IsPullRequest,
			"html_url":        it.HTMLURL,
			"created_at":      it.CreatedAt,
			"updated_at":      it.UpdatedAt,
			"closed_at":       it.ClosedAt,
			"raw":             it.Raw,
		}
		models = append(models, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(bson.M{"$set": set}).SetUpsert(true))
	}

	if _, err := m.issuesCol.BulkWrite(ctx, models); err != nil {
		return 0, fmt.Errorf("mongo bulk upsert issues: %w", err)
	}
	return len(issues), nil
}

func (m *MongoSyncStore) ListIssues(ctx context.Context, filter SyncIssueFilter) ([]SyncedIssue, error) {
	q := bson.M{}
	if filter.Repo != "" {
		q["repo"] = filter.Repo
	}
	if filter.State != "" && filter.State != "all" {
		q["state"] = filter.State
	}
	if filter.IssueID > 0 {
		q["issue_id"] = filter.IssueID
	}
	if filter.Number > 0 {
		q["number"] = filter.Number
	}

	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	cursor, err := m.issuesCol.Find(ctx, q, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo list issues: %w", err)
	}
	defer cursor.Close(ctx)

	out := make([]SyncedIssue, 0)
	for cursor.Next(ctx) {
		var doc mongoIssueDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode issue doc: %w", err)
		}
		out = append(out, SyncedIssue{
			Repo:          doc.Repo,
			IssueID:       doc.IssueID,
			Number:        doc.Number,
			Title:         doc.Title,
			Body:          doc.Body,
			State:         doc.State,
			Author:        doc.Author,
			Assignees:     doc.Assignees,
			Labels:        doc.Labels,
			Comments:      doc.Comments,
			IsPullRequest: doc.IsPullRequest,
			HTMLURL:       doc.HTMLURL,
			CreatedAt:     doc.CreatedAt,
			UpdatedAt:     doc.UpdatedAt,
			ClosedAt:      doc.ClosedAt,
			Raw:           doc.Raw,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate issue docs: %w", err)
	}
	return out, nil
}

func (m *MongoSyncStore) GetRepoCheckpoint(ctx context.Context, repo string) (Checkpoint, error) {
	var doc mongoCheckpointDoc
	err := m.checkpointC.FindOne(ctx, bson.M{"repo": repo}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return Checkpoint{Repo: repo}, nil
		}
		return Checkpoint{}, fmt.Errorf("get checkpoint: %w", err)
	}
	return Checkpoint{
		Repo:               doc.Repo,
		LastSyncedAt:       doc.LastSyncedAt,
		LastIssueUpdatedAt: doc.LastIssueUpdatedAt,
		LastRunStatus:      doc.LastRunStatus,
		LastError:          doc.LastError,
		UpdatedAt:          doc.UpdatedAt,
	}, nil
}

func (m *MongoSyncStore) SaveRepoCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	checkpoint.UpdatedAt = time.Now()
	_, err := m.checkpointC.UpdateOne(ctx,
		bson.M{"repo": checkpoint.Repo},
		bson.M{"$set": bson.M{
			"repo":                  checkpoint.Repo,
			"last_synced_at":        checkpoint.LastSyncedAt,
			"last_issue_updated_at": checkpoint.LastIssueUpdatedAt,
			"last_run_status":       checkpoint.LastRunStatus,
			"last_error":            checkpoint.LastError,
			"updated_at":            checkpoint.UpdatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	return nil
}

func (m *MongoSyncStore) ListCheckpoints(ctx context.Context) ([]Checkpoint, error) {
	cursor, err := m.checkpointC.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "repo", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("list checkpoints: %w", err)
	}
	defer cursor.Close(ctx)

	out := make([]Checkpoint, 0)
	for cursor.Next(ctx) {
		var doc mongoCheckpointDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode checkpoint doc: %w", err)
		}
		out = append(out, Checkpoint{
			Repo:               doc.Repo,
			LastSyncedAt:       doc.LastSyncedAt,
			LastIssueUpdatedAt: doc.LastIssueUpdatedAt,
			LastRunStatus:      doc.LastRunStatus,
			LastError:          doc.LastError,
			UpdatedAt:          doc.UpdatedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkpoints: %w", err)
	}
	return out, nil
}

func (m *MongoSyncStore) Close() error {
	return m.client.Disconnect(context.Background())
}

func (m *MongoSyncStore) UpsertFeedSource(ctx context.Context, source FeedSource) (FeedSource, error) {
	if source.ID == "" {
		return FeedSource{}, fmt.Errorf("feed source id is empty")
	}
	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		existing, err := m.GetFeedSource(ctx, source.ID)
		if err == nil {
			source.CreatedAt = existing.CreatedAt
		}
	}
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now
	_, err := m.feedSourceC.UpdateOne(ctx,
		bson.M{"id": source.ID},
		bson.M{"$set": bson.M{
			"id":              source.ID,
			"url":             source.URL,
			"display_name":    source.DisplayName,
			"description":     source.Description,
			"site_url":        source.SiteURL,
			"enabled":         source.Enabled,
			"etag":            source.ETag,
			"last_modified":   source.LastModified,
			"last_synced_at":  source.LastSyncedAt,
			"last_success_at": source.LastSuccessAt,
			"last_run_status": source.LastRunStatus,
			"last_error":      source.LastError,
			"created_at":      source.CreatedAt,
			"updated_at":      source.UpdatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return FeedSource{}, fmt.Errorf("upsert feed source: %w", err)
	}
	return source, nil
}

func (m *MongoSyncStore) GetFeedSource(ctx context.Context, id string) (FeedSource, error) {
	var doc mongoFeedSourceDoc
	err := m.feedSourceC.FindOne(ctx, bson.M{"id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return FeedSource{}, ErrFeedSourceNotFound
		}
		return FeedSource{}, fmt.Errorf("get feed source: %w", err)
	}
	return FeedSource{
		ID:            doc.ID,
		URL:           doc.URL,
		DisplayName:   doc.DisplayName,
		Description:   doc.Description,
		SiteURL:       doc.SiteURL,
		Enabled:       doc.Enabled,
		ETag:          doc.ETag,
		LastModified:  doc.LastModified,
		LastSyncedAt:  doc.LastSyncedAt,
		LastSuccessAt: doc.LastSuccessAt,
		LastRunStatus: doc.LastRunStatus,
		LastError:     doc.LastError,
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
	}, nil
}

func (m *MongoSyncStore) ListFeedSources(ctx context.Context, filter FeedSourceFilter) ([]FeedSource, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "id", Value: 1}})
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	cursor, err := m.feedSourceC.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list feed sources: %w", err)
	}
	defer cursor.Close(ctx)

	out := make([]FeedSource, 0)
	for cursor.Next(ctx) {
		var doc mongoFeedSourceDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode feed source: %w", err)
		}
		out = append(out, FeedSource{
			ID:            doc.ID,
			URL:           doc.URL,
			DisplayName:   doc.DisplayName,
			Description:   doc.Description,
			SiteURL:       doc.SiteURL,
			Enabled:       doc.Enabled,
			ETag:          doc.ETag,
			LastModified:  doc.LastModified,
			LastSyncedAt:  doc.LastSyncedAt,
			LastSuccessAt: doc.LastSuccessAt,
			LastRunStatus: doc.LastRunStatus,
			LastError:     doc.LastError,
			CreatedAt:     doc.CreatedAt,
			UpdatedAt:     doc.UpdatedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed sources: %w", err)
	}
	return out, nil
}

func (m *MongoSyncStore) DeleteFeedSource(ctx context.Context, id string) error {
	if _, err := m.feedContentC.DeleteMany(ctx, bson.M{"feed_source_id": id}); err != nil {
		return fmt.Errorf("delete feed contents: %w", err)
	}
	if _, err := m.feedCheckpointC.DeleteMany(ctx, bson.M{"feed_source_id": id}); err != nil {
		return fmt.Errorf("delete feed checkpoints: %w", err)
	}
	result, err := m.feedSourceC.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("delete feed source: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrFeedSourceNotFound
	}
	return nil
}

func (m *MongoSyncStore) UpsertFeedContents(ctx context.Context, sourceID string, contents []FeedContent) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	models := make([]mongo.WriteModel, 0, len(contents))
	for _, content := range contents {
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"feed_source_id": sourceID, "identity": content.Identity}).
			SetUpdate(bson.M{"$set": bson.M{
				"id":             content.ID,
				"feed_source_id": sourceID,
				"identity":       content.Identity,
				"guid":           content.GUID,
				"title":          content.Title,
				"summary":        content.Summary,
				"content":        content.Content,
				"link":           content.Link,
				"author":         content.Author,
				"categories":     content.Categories,
				"published_at":   content.PublishedAt,
				"updated_at":     content.UpdatedAt,
				"fetched_at":     content.FetchedAt,
			}}).
			SetUpsert(true))
	}
	if _, err := m.feedContentC.BulkWrite(ctx, models); err != nil {
		return 0, fmt.Errorf("upsert feed contents: %w", err)
	}
	return len(contents), nil
}

func (m *MongoSyncStore) ListFeedContents(ctx context.Context, filter FeedContentFilter) ([]FeedContent, error) {
	query := bson.M{}
	if filter.FeedSourceID != "" {
		query["feed_source_id"] = filter.FeedSourceID
	}
	if filter.ContentID != "" {
		query["id"] = filter.ContentID
	}
	opts := options.Find().SetSort(bson.D{{Key: "published_at", Value: -1}, {Key: "id", Value: 1}})
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	cursor, err := m.feedContentC.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("list feed contents: %w", err)
	}
	defer cursor.Close(ctx)

	out := make([]FeedContent, 0)
	for cursor.Next(ctx) {
		var doc mongoFeedContentDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode feed content: %w", err)
		}
		out = append(out, FeedContent{
			ID:           doc.ID,
			FeedSourceID: doc.FeedSourceID,
			Identity:     doc.Identity,
			GUID:         doc.GUID,
			Title:        doc.Title,
			Summary:      doc.Summary,
			Content:      doc.Content,
			Link:         doc.Link,
			Author:       doc.Author,
			Categories:   doc.Categories,
			PublishedAt:  doc.PublishedAt,
			UpdatedAt:    doc.UpdatedAt,
			FetchedAt:    doc.FetchedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed contents: %w", err)
	}
	return out, nil
}

func (m *MongoSyncStore) GetFeedContent(ctx context.Context, id string) (FeedContent, error) {
	rows, err := m.ListFeedContents(ctx, FeedContentFilter{ContentID: id, Limit: 1})
	if err != nil {
		return FeedContent{}, err
	}
	if len(rows) == 0 {
		return FeedContent{}, ErrFeedContentNotFound
	}
	return rows[0], nil
}

func (m *MongoSyncStore) GetFeedCheckpoint(ctx context.Context, sourceID string) (FeedCheckpoint, error) {
	var doc mongoFeedCheckpointDoc
	err := m.feedCheckpointC.FindOne(ctx, bson.M{"feed_source_id": sourceID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return FeedCheckpoint{FeedSourceID: sourceID}, nil
		}
		return FeedCheckpoint{}, fmt.Errorf("get feed checkpoint: %w", err)
	}
	return FeedCheckpoint{
		FeedSourceID:  doc.FeedSourceID,
		LastSyncedAt:  doc.LastSyncedAt,
		LastSuccessAt: doc.LastSuccessAt,
		LastRunStatus: doc.LastRunStatus,
		LastError:     doc.LastError,
		ETag:          doc.ETag,
		LastModified:  doc.LastModified,
		UpdatedAt:     doc.UpdatedAt,
	}, nil
}

func (m *MongoSyncStore) SaveFeedCheckpoint(ctx context.Context, checkpoint FeedCheckpoint) error {
	checkpoint.UpdatedAt = time.Now().UTC()
	_, err := m.feedCheckpointC.UpdateOne(ctx,
		bson.M{"feed_source_id": checkpoint.FeedSourceID},
		bson.M{"$set": bson.M{
			"feed_source_id":  checkpoint.FeedSourceID,
			"last_synced_at":  checkpoint.LastSyncedAt,
			"last_success_at": checkpoint.LastSuccessAt,
			"last_run_status": checkpoint.LastRunStatus,
			"last_error":      checkpoint.LastError,
			"etag":            checkpoint.ETag,
			"last_modified":   checkpoint.LastModified,
			"updated_at":      checkpoint.UpdatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("save feed checkpoint: %w", err)
	}
	return nil
}
