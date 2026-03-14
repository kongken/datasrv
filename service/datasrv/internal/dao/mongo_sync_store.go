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

// MongoSyncStore stores synced issue data in MongoDB.
type MongoSyncStore struct {
	client      *mongo.Client
	db          *mongo.Database
	issuesCol   *mongo.Collection
	checkpointC *mongo.Collection
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
		client:      client,
		db:          db,
		issuesCol:   db.Collection("github_issues"),
		checkpointC: db.Collection("github_issue_checkpoints"),
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
