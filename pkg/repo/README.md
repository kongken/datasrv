# GitHub Issue MongoDB DAO

这是一个用于将 GitHub Issue 持久化存储到 MongoDB 的数据访问对象（DAO）实现。

## 特性

- 使用 MongoDB Go Driver v2
- 完整的 CRUD 操作
- 支持 Upsert 操作
- 多种查询方法（按状态、标签等）
- 自动创建索引

## 使用示例

### 初始化

```go
import (
    "context"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
    "github.com/kongken/monkey/pkg/repo"
)

// 连接 MongoDB
client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
if err != nil {
    panic(err)
}
defer client.Disconnect(context.Background())

// 获取数据库
db := client.Database("github_data")

// 创建 DAO
issueDAO := repo.NewIssueDAO(db)

// 创建索引（首次使用时）
err = issueDAO.CreateIndexes(context.Background())
if err != nil {
    panic(err)
}
```

### 创建 Issue

```go
issue := &repo.Issue{
    ID:     123456,
    Number: 1,
    Title:  "Bug fix",
    Body:   "Fix the login issue",
    State:  "open",
    User: &repo.User{
        ID:    1001,
        Login: "developer",
    },
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
}

err := issueDAO.Create(context.Background(), issue)
```

### Upsert Issue

```go
// 如果存在则更新，不存在则创建
err := issueDAO.Upsert(context.Background(), issue)
```

### 查询 Issue

```go
// 根据 ID 查找
issue, err := issueDAO.FindByID(context.Background(), 123456)

// 根据 Number 查找
issue, err := issueDAO.FindByNumber(context.Background(), 1)

// 查找所有 issues
issues, err := issueDAO.FindAll(context.Background())

// 根据状态查找
openIssues, err := issueDAO.FindByState(context.Background(), "open")

// 根据标签查找
issues, err := issueDAO.FindByLabels(context.Background(), []string{"bug", "urgent"})
```

### 更新 Issue

```go
issue.State = "closed"
issue.ClosedAt = &now
err := issueDAO.Update(context.Background(), issue)
```

### 删除 Issue

```go
err := issueDAO.Delete(context.Background(), 123456)
```

### 统计

```go
// 统计所有 issues
count, err := issueDAO.Count(context.Background(), bson.D{})

// 统计 open 状态的 issues
count, err := issueDAO.Count(context.Background(), bson.D{{"state", "open"}})
```

## 数据结构

### Issue
- `ID`: GitHub issue ID（作为 MongoDB _id）
- `Number`: Issue 编号
- `Title`: 标题
- `Body`: 内容
- `State`: 状态（open/closed）
- `User`: 创建者
- `Labels`: 标签列表
- `Assignees`: 受理人列表
- `Comments`: 评论数量
- `CreatedAt`: 创建时间
- `UpdatedAt`: 更新时间
- `ClosedAt`: 关闭时间
- `HTMLURL`: GitHub URL
- `Milestone`: 里程碑
- `Locked`: 是否锁定

## 索引

DAO 会自动创建以下索引：
- `number`: 唯一索引
- `state`: 普通索引
- `created_at`: 降序索引
- `updated_at`: 降序索引
- `labels.name`: 普通索引
