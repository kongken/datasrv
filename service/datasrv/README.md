# GitHub Issues Data Service

GitHub Issues 数据服务，用于从 GitHub 抓取 issues 并持久化到数据库。

## 架构设计

### DAO 层抽象

为了支持未来切换到 MongoDB 或其他数据库，DAO 层采用了接口抽象设计：

```
service/datasrv/internal/dao/
├── dao.go          # DAO 接口定义
├── postgres.go     # PostgreSQL 实现（使用 ent ORM）
└── ent/           # ent 生成的代码
    └── schema/    # 数据库 schema 定义
```

#### DAO 接口

`dao.go` 定义了以下接口：

- `IssueDAO` - Issue 数据访问操作
- `UserDAO` - User 数据访问操作
- `LabelDAO` - Label 数据访问操作
- `MilestoneDAO` - Milestone 数据访问操作
- `DAO` - 聚合所有 DAO 接口

#### 数据模型

所有数据模型定义在 `dao.go` 中：

- `IssueModel` - Issue 数据模型
- `UserModel` - User 数据模型
- `LabelModel` - Label 数据模型
- `MilestoneModel` - Milestone 数据模型

这些模型独立于具体的 ORM 实现，便于切换数据库。

### PostgreSQL 实现

`postgres.go` 使用 ent ORM 实现了 DAO 接口：

- 支持批量插入
- 自动处理关联关系（User、Label、Milestone）
- 事务支持
- 自动迁移

### Service 层

`service/github.go` 实现了业务逻辑：

- 从 GitHub API 获取 issues
- 转换数据格式
- 调用 DAO 层持久化数据
- 分页处理大量数据

## 使用方法

### 1. 配置数据库

确保 PostgreSQL 已安装并运行：

```bash
# 创建数据库
createdb github_issues

# 或使用 psql
psql -c "CREATE DATABASE github_issues;"
```

### 2. 设置环境变量

```bash
export DATABASE_DSN="host=localhost port=5432 user=postgres password=postgres dbname=github_issues sslmode=disable"
export GITHUB_TOKEN="your_github_token"  # 可选，提高 API 限制
```

### 3. 运行示例程序

```bash
cd service/datasrv/cmd/example
go run main.go
```

### 4. 在代码中使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/kongken/datasrv/service/datasrv/internal/service"
    "github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func main() {
    ctx := context.Background()
    
    // 创建服务
    cfg := &service.Config{
        DatabaseDSN: "host=localhost port=5432 user=postgres dbname=github_issues sslmode=disable",
        GitHubToken: "your_token",
    }
    
    svc, err := service.NewGitHubServiceWithConfig(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取并存储所有 open issues
    err = svc.FetchAndStoreAllIssues(ctx, "golang", "go", "open")
    if err != nil {
        log.Fatal(err)
    }
    
    // 列出数据库中的 issues
    issues, err := svc.ListIssues(ctx, &dao.ListOptions{
        Limit:  10,
        State:  "open",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    for _, issue := range issues {
        log.Printf("Issue #%d: %s\n", issue.Number, issue.Title)
    }
}
```

## 数据库 Schema

使用 ent ORM 定义的 schema：

### Issue

- `id` - GitHub issue ID（主键）
- `number` - Issue 编号
- `title` - 标题
- `body` - 内容
- `state` - 状态（open/closed）
- `comments` - 评论数
- `html_url` - HTML URL
- `locked` - 是否锁定
- `created_at` - 创建时间
- `updated_at` - 更新时间
- `closed_at` - 关闭时间
- `user_id` - 创建者 ID（外键）
- `milestone_id` - 里程碑 ID（外键）

### User

- `id` - GitHub user ID（主键）
- `login` - 用户名
- `avatar_url` - 头像 URL
- `html_url` - 用户主页 URL

### Label

- `id` - GitHub label ID（主键）
- `name` - 标签名
- `color` - 颜色
- `description` - 描述

### Milestone

- `id` - GitHub milestone ID（主键）
- `number` - 里程碑编号
- `title` - 标题
- `description` - 描述
- `state` - 状态（open/closed）
- `due_on` - 截止日期
- `created_at` - 创建时间
- `updated_at` - 更新时间

## 扩展：支持 MongoDB

要添加 MongoDB 支持，只需：

1. 创建 `dao/mongo.go` 实现 `DAO` 接口
2. 使用相同的数据模型（`IssueModel`、`UserModel` 等）
3. Service 层代码无需修改

示例：

```go
// dao/mongo.go
type MongoDB struct {
    client *mongo.Client
}

func NewMongoDB(uri string) (*MongoDB, error) {
    // 实现 MongoDB 连接
}

func (m *MongoDB) CreateIssue(ctx context.Context, issue *IssueModel) error {
    // 实现 MongoDB 插入逻辑
}

// ... 实现其他接口方法
```

然后在创建服务时选择使用 MongoDB：

```go
// 使用 MongoDB
mongoDB, err := dao.NewMongoDB("mongodb://localhost:27017")
svc := service.NewGitHubService(githubClient, mongoDB)

// 或使用 PostgreSQL
postgresDAO, err := dao.NewPostgresDAO(dsnString)
svc := service.NewGitHubService(githubClient, postgresDAO)
```

## API 方法

### GitHubService

- `FetchAndStoreIssues` - 获取并存储指定条件的 issues
- `FetchAndStoreAllIssues` - 获取并存储所有 issues（分页）
- `SyncIssue` - 同步单个 issue
- `GetIssueByID` - 根据 ID 获取 issue
- `GetIssueByNumber` - 根据编号获取 issue
- `ListIssues` - 列出 issues（支持分页和过滤）

## 依赖

- `github.com/google/go-github/v82` - GitHub API 客户端
- `entgo.io/ent` - ORM 框架
- `github.com/lib/pq` - PostgreSQL 驱动

## 测试

```bash
# 运行测试
go test ./service/datasrv/internal/...

# 运行特定测试
go test ./service/datasrv/internal/dao -v
go test ./service/datasrv/internal/service -v
```

## 注意事项

1. **GitHub API 限制**：未认证请求限制为 60 次/小时，建议使用 Personal Access Token
2. **数据库迁移**：首次运行会自动创建表结构
3. **事务处理**：批量操作使用事务保证数据一致性
4. **关联数据**：会自动 upsert 关联的 User、Label、Milestone
