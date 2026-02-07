package repo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// IssueDAO GitHub issue 数据访问对象
type IssueDAO struct {
	collection *mongo.Collection
}

// NewIssueDAO 创建新的 IssueDAO 实例
func NewIssueDAO(db *mongo.Database) *IssueDAO {
	return &IssueDAO{
		collection: db.Collection("github_issues"),
	}
}

// Issue MongoDB 存储的 issue 结构
type Issue struct {
	ID        int64      `bson:"_id"`
	Number    int32      `bson:"number"`
	Title     string     `bson:"title"`
	Body      string     `bson:"body"`
	State     string     `bson:"state"`
	User      *User      `bson:"user,omitempty"`
	Labels    []*Label   `bson:"labels,omitempty"`
	Assignees []*User    `bson:"assignees,omitempty"`
	Comments  int32      `bson:"comments"`
	CreatedAt time.Time  `bson:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at"`
	ClosedAt  *time.Time `bson:"closed_at,omitempty"`
	HTMLURL   string     `bson:"html_url"`
	Milestone *Milestone `bson:"milestone,omitempty"`
	Locked    bool       `bson:"locked"`
}

// User GitHub 用户
type User struct {
	ID        int64  `bson:"id"`
	Login     string `bson:"login"`
	AvatarURL string `bson:"avatar_url"`
	HTMLURL   string `bson:"html_url"`
}

// Label issue 标签
type Label struct {
	ID          int64  `bson:"id"`
	Name        string `bson:"name"`
	Color       string `bson:"color"`
	Description string `bson:"description"`
}

// Milestone 里程碑
type Milestone struct {
	ID          int64      `bson:"id"`
	Number      int32      `bson:"number"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	State       string     `bson:"state"`
	DueOn       *time.Time `bson:"due_on,omitempty"`
}

// IssueComment issue 评论
type IssueComment struct {
	ID        int64     `bson:"_id"`
	Body      string    `bson:"body"`
	User      *User     `bson:"user,omitempty"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
	HTMLURL   string    `bson:"html_url"`
}

// Create 创建新的 issue
func (dao *IssueDAO) Create(ctx context.Context, issue *Issue) error {
	_, err := dao.collection.InsertOne(ctx, issue)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}
	return nil
}

// Update 更新 issue
func (dao *IssueDAO) Update(ctx context.Context, issue *Issue) error {
	filter := bson.D{{"_id", issue.ID}}
	update := bson.D{{"$set", issue}}

	result, err := dao.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("issue not found: %d", issue.ID)
	}

	return nil
}

// Upsert 创建或更新 issue
func (dao *IssueDAO) Upsert(ctx context.Context, issue *Issue) error {
	filter := bson.D{{"_id", issue.ID}}
	update := bson.D{{"$set", issue}}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := dao.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert issue: %w", err)
	}

	return nil
}

// FindByID 根据 ID 查找 issue
func (dao *IssueDAO) FindByID(ctx context.Context, id int64) (*Issue, error) {
	filter := bson.D{{"_id", id}}

	var issue Issue
	err := dao.collection.FindOne(ctx, filter).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("issue not found: %d", id)
		}
		return nil, fmt.Errorf("failed to find issue: %w", err)
	}

	return &issue, nil
}

// FindByNumber 根据 number 查找 issue
func (dao *IssueDAO) FindByNumber(ctx context.Context, number int32) (*Issue, error) {
	filter := bson.D{{"number", number}}

	var issue Issue
	err := dao.collection.FindOne(ctx, filter).Decode(&issue)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("issue not found: %d", number)
		}
		return nil, fmt.Errorf("failed to find issue: %w", err)
	}

	return &issue, nil
}

// FindAll 查找所有 issues
func (dao *IssueDAO) FindAll(ctx context.Context) ([]*Issue, error) {
	cursor, err := dao.collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to find issues: %w", err)
	}
	defer cursor.Close(ctx)

	var issues []*Issue
	if err := cursor.All(ctx, &issues); err != nil {
		return nil, fmt.Errorf("failed to decode issues: %w", err)
	}

	return issues, nil
}

// FindByState 根据状态查找 issues
func (dao *IssueDAO) FindByState(ctx context.Context, state string) ([]*Issue, error) {
	filter := bson.D{{"state", state}}

	cursor, err := dao.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find issues by state: %w", err)
	}
	defer cursor.Close(ctx)

	var issues []*Issue
	if err := cursor.All(ctx, &issues); err != nil {
		return nil, fmt.Errorf("failed to decode issues: %w", err)
	}

	return issues, nil
}

// FindByLabels 根据标签查找 issues
func (dao *IssueDAO) FindByLabels(ctx context.Context, labelNames []string) ([]*Issue, error) {
	filter := bson.D{{"labels.name", bson.D{{"$in", labelNames}}}}

	cursor, err := dao.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find issues by labels: %w", err)
	}
	defer cursor.Close(ctx)

	var issues []*Issue
	if err := cursor.All(ctx, &issues); err != nil {
		return nil, fmt.Errorf("failed to decode issues: %w", err)
	}

	return issues, nil
}

// Delete 删除 issue
func (dao *IssueDAO) Delete(ctx context.Context, id int64) error {
	filter := bson.D{{"_id", id}}

	result, err := dao.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("issue not found: %d", id)
	}

	return nil
}

// Count 统计 issues 数量
func (dao *IssueDAO) Count(ctx context.Context, filter interface{}) (int64, error) {
	count, err := dao.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count issues: %w", err)
	}
	return count, nil
}

// CreateIndexes 创建索引
func (dao *IssueDAO) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"number", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"state", 1}},
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
		{
			Keys: bson.D{{"updated_at", -1}},
		},
		{
			Keys: bson.D{{"labels.name", 1}},
		},
	}

	_, err := dao.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}
