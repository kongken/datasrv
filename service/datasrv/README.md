# GitHub Issues Data Service

GitHub Issues 数据服务，用于从 GitHub 抓取 issues 并持久化到数据库。

## 架构设计

### 项目结构

```
service/datasrv/
├── internal/
│   ├── app.go              # 应用初始化
│   ├── conf/
│   │   └── conf.go         # 配置管理
│   ├── dao/
│   │   ├── dao.go          # DAO 接口定义
│   │   ├── postgres.go     # PostgreSQL 实现（使用 ent ORM）
│   │   └── ent/            # ent 生成的代码
│   │       └── schema/     # 数据库 schema 定义
│   └── service/
│       └── github.go       # GitHub service 业务逻辑
└── README.md
```

### 配置管理

配置通过 `internal/conf/conf.go` 统一管理，使用 YAML 格式。

配置文件示例：`service/datasrv/config.yaml.example`

配置结构：

```yaml
database:
  driver: string          # 数据库驱动（postgres, mongodb）
  dsn: string            # 数据库连接字符串
  max_open_conns: int    # 最大连接数
  max_idle_conns: int    # 最大空闲连接数

github:
  token: string          # GitHub 访问令牌
  base_url: string       # GitHub API Base URL（用于 GitHub Enterprise）

server:
  host: string           # 服务器主机
  port: int              # 服务器端口
```

> **注意**：配置由框架自动加载（如使用 viper、go-micro config 等），`conf.Config` 结构体定义了配置格式，框架会自动将 YAML 文件映射到该结构体。

### 应用初始化

`internal/app.go` 负责应用的初始化和资源管理：

- 加载配置
- 初始化 DAO 层（根据配置选择数据库驱动）
- 初始化 GitHub 客户端
- 创建 GitHub Service
- 提供资源清理方法

### DAO 层抽象

为了支持未来切换到 MongoDB 或其他数据库，DAO 层采用了接口抽象设计：

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

### 2. 配置文件

复制示例配置文件并根据需要修改：

```bash
cp service/datasrv/config.yaml.example config.yaml
```

编辑 `config.yaml` 配置文件：

```yaml
# Database configuration
database:
  driver: postgres
  dsn: "host=localhost port=5432 user=postgres password=postgres dbname=github_issues sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 10

# GitHub API configuration
github:
  token: "your_github_token"  # 可选，提高 API 限制
  base_url: ""                # 留空使用 github.com，或设置为 GitHub Enterprise URL

# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
```

> **注意**：框架会自动加载配置文件，无需手动初始化配置。

### 3. 在代码中使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/kongken/datasrv/service/datasrv/internal"
    "github.com/kongken/datasrv/service/datasrv/internal/conf"
    "github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func main() {
    ctx := context.Background()
    
    // 配置由框架自动加载（如 viper 等）
    // 这里假设配置已经通过框架加载到 cfg 变量中
    var cfg *conf.Config
    // cfg = loadConfigByFramework() // 框架自动加载配置
    
    // 使用配置创建应用
    app, err := internal.NewApp(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer app.Close()
    
    // 使用 GitHubService 获取并存储 issues
    err = app.GitHubService.FetchAndStoreAllIssues(ctx, "golang", "go", "open")
    if err != nil {
        log.Fatal(err)
    }
    
    // 列出数据库中的 issues
    issues, err := app.GitHubService.ListIssues(ctx, &dao.ListOptions{
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
3. 在 `app.go` 的 `initDAO` 方法中添加 MongoDB 分支
4. Service 层和配置层代码无需修改

示例：

```go
// dao/mongo.go
package dao

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
    client *mongo.Client
    db     *mongo.Database
}

func NewMongoDB(uri string) (*MongoDB, error) {
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }
    
    return &MongoDB{
        client: client,
        db:     client.Database("github_issues"),
    }, nil
}

func (m *MongoDB) CreateIssue(ctx context.Context, issue *IssueModel) error {
    collection := m.db.Collection("issues")
    _, err := collection.InsertOne(ctx, issue)
    return err
}

// ... 实现其他接口方法
```

然后在 `app.go` 中添加 MongoDB 支持：

```go
func (a *App) initDAO(ctx context.Context) error {
    switch a.Config.Database.Driver {
    case "postgres", "postgresql":
        // ... 现有代码
    
    case "mongodb", "mongo":
        mongoDB, err := dao.NewMongoDB(a.Config.Database.DSN)
        if err != nil {
            return fmt.Errorf("failed to create MongoDB DAO: %w", err)
        }
        a.DAO = mongoDB
        log.Println("MongoDB DAO initialized successfully")
        return nil
    
    default:
        return fmt.Errorf("unsupported database driver: %s", a.Config.Database.Driver)
    }
}
```

使用时只需修改配置文件：

```yaml
database:
  driver: mongodb
  dsn: "mongodb://localhost:27017/github_issues"
  max_open_conns: 25
  max_idle_conns: 10
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
