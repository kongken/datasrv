# datasrv 服务功能设计

## 服务定位

`datasrv` 是一个面向内容聚合与展示的数据同步服务，负责从外部系统抓取结构化内容并持久化到数据库，对内提供统一的同步与查询能力，对外通过 gRPC 提供后台管理接口和前端展示接口。

首期服务覆盖两类数据源：

- GitHub Issues：从指定仓库同步 issue 数据到数据库。
- RSS Feeds：抓取 feed 及其条目内容并持久化到数据库。

服务整体以“同步编排 + DAO Service 抽象 + 可配置存储后端”的方式组织，业务逻辑不直接依赖具体数据库实现，部署时可根据配置选择 `mongo` 或 `postgres` 作为底层存储。

## 总体架构

服务建议分为五层：

1. `source client`
负责对接外部数据源，包括 GitHub API client 和 RSS fetch/parser。该层只负责拉取外部数据，不处理业务存储逻辑。

2. `sync service`
负责同步任务编排，是整个服务的核心。该层负责源配置读取、定时与手动触发、增量同步、失败重试、标准化转换、checkpoint 推进，并将标准化后的数据交给 DAO Service 持久化。

3. `dao service`
负责定义服务内部统一依赖的数据访问契约。上层同步逻辑和接口逻辑只依赖 DAO 抽象，不直接感知 Mongo 或 Postgres 的存储细节。

4. `storage implementation`
负责具体数据库驱动实现。系统支持 `mongo` 与 `postgres` 两种后端，通过统一接口对上层暴露一致能力。

5. `api layer`
负责对外暴露 gRPC 接口，划分为后台管理使用的 `Admin API` 和面向前端展示使用的 `Query API`。

## 核心目标

### 1. GitHub Issues 同步到数据库

服务需要支持从一个或多个 GitHub 仓库抓取 issue，并持久化到配置的数据库中。同步逻辑应支持：

- 配置多个 `owner/repo` 仓库源。
- 支持手动触发单仓库或全量仓库同步。
- 支持基于 `updated_at` 或 checkpoint 的增量同步。
- 支持将 issue 基础字段、作者、标签、指派人、状态等信息标准化并落库。
- 支持 upsert，保证重复同步不会产生脏数据。
- 支持记录每个仓库的最近同步状态、最后成功时间、失败原因和最近处理结果。

GitHub issue 是当前服务的首个同步对象，现有 proto 与 gRPC service 已围绕该能力建立基础接口。

### 2. RSS Feed 抓取与内容持久化

服务需要支持配置多个 RSS/Atom Feed 源，并将抓取到的 feed 内容写入数据库。这里的重点不是只保存 feed 源配置，而是要持久化每次抓取到的内容实体，供后续查询和前端展示使用。

RSS 同步逻辑应支持：

- 管理多个 feed 源地址及启停状态。
- 拉取并解析 RSS/Atom 数据。
- 持久化 feed 元信息，如标题、链接、描述、更新时间等。
- 持久化 entry/item 内容，如标题、摘要、正文、链接、作者、发布时间、分类等。
- 支持基于发布时间、etag、last-modified 或内容唯一键的增量抓取。
- 支持条目级幂等写入，避免重复持久化。
- 支持记录每个 feed 的同步 checkpoint、最近成功时间、失败原因和运行统计。

### 3. 提供后台管理能力

所有同步相关功能都需要通过 `Admin API` 对外暴露，支持后台系统进行配置和运营管理。后台能力覆盖：

- 源管理：配置 GitHub 仓库列表、RSS feed 列表、启停状态和同步参数。
- 任务控制：手动触发同步、按源执行、失败任务重试。
- 运行状态查看：查看最近运行结果、checkpoint、失败原因、最近成功时间和处理条数。
- 运营能力：查看 issue/feed/entry 总量、最近新增量、失败源列表、指定时间窗口内的同步状态。

### 4. 提供前端展示查询能力

服务还需要提供给前端使用的查询接口，通过网关或其他转换层消费 gRPC query 接口。查询接口保持只读，重点面向展示场景。

Issue 查询侧至少支持：

- 按仓库分页查询 issue 列表。
- 按状态筛选 issue。
- 查询单个 issue 详情。

RSS 查询侧至少支持：

- 查询 feed 列表。
- 查询某个 feed 下的内容列表。
- 查询单条 feed 内容详情。
- 支持按发布时间倒序分页查询。

## 数据流设计

### GitHub Issue 数据流

GitHub issue 同步链路如下：

1. 从配置中读取待同步仓库列表。
2. `sync service` 调用 GitHub API client 拉取 issue。
3. 根据每个仓库的 checkpoint 执行增量同步。
4. 将外部 issue 数据转换为内部标准化模型。
5. 调用 Issue 相关 DAO Service 完成 upsert。
6. 更新同步 checkpoint、运行结果和错误状态。
7. 通过 Query API 向前端或网关提供 issue 查询能力。
8. 通过 Admin API 向后台提供同步配置和状态查询能力。

### RSS Feed 数据流

RSS feed 抓取链路如下：

1. 从配置中读取待抓取 feed 源列表。
2. `sync service` 调用 RSS fetch/parser 拉取并解析源内容。
3. 将 feed 元信息与 entry 内容分别转换为内部模型。
4. 调用 Feed 相关 DAO Service 持久化 feed 和内容条目。
5. 更新 feed 的 checkpoint、抓取时间、运行状态和错误信息。
6. 通过 Query API 对前端提供 feed 和条目内容查询能力。
7. 通过 Admin API 提供 feed 源管理、抓取状态和统计能力。

两条数据流复用同一套同步框架，只是在 source client 和 DAO 实体类型上有所区别。

## DAO Service 抽象

为了支持可切换的数据库后端，服务需要抽象统一的 DAO Service 层。推荐按职责拆分为以下几个接口域：

### 1. `SourceDAOService`

负责管理同步源配置，包括：

- GitHub 仓库源配置
- RSS feed 源配置
- 源启停状态
- 同步参数与调度配置

### 2. `SyncStateDAOService`

负责管理同步状态和运行信息，包括：

- checkpoint
- 最近同步时间
- 最近成功时间
- 最近失败原因
- 最近一次运行结果
- 统计信息和失败列表

### 3. `IssueDAOService`

负责 issue 相关数据读写，包括：

- issue upsert
- issue 列表查询
- issue 详情查询
- issue 状态筛选和分页查询

### 4. `FeedDAOService`

负责 RSS 相关数据读写，包括：

- feed 源元信息持久化
- feed 条目内容 upsert
- feed 列表查询
- feed 条目列表查询
- feed 条目详情查询

通过这一层抽象，上层 `sync service` 与 `api layer` 不需要依赖具体的 Mongo 或 Postgres 实现，只依赖稳定的领域接口。

## 数据模型建议

### GitHub Issue 相关模型

- `Issue`
- `IssueSyncCheckpoint`
- `IssueSyncRunRecord`

其中：

- `Issue` 用于前端展示与查询。
- `IssueSyncCheckpoint` 用于增量同步推进。
- `IssueSyncRunRecord` 用于后台查看任务运行情况与排障。

### RSS Feed 相关模型

- `FeedSource`
- `FeedContent`
- `FeedSyncCheckpoint`
- `FeedSyncRunRecord`

其中：

- `FeedSource` 表示被管理的 RSS 源配置。
- `FeedContent` 表示抓取后持久化的 feed 内容实体，是前端展示的核心数据。
- `FeedSyncCheckpoint` 表示 feed 增量抓取状态。
- `FeedSyncRunRecord` 表示 feed 抓取运行记录、错误信息和处理统计。

## API 分组设计

### Admin API

Admin API 面向后台运营与管理场景，建议提供以下能力：

- 获取和更新 GitHub issue 同步配置
- 获取和更新 RSS feed 抓取配置
- 手动触发 GitHub issue 同步
- 手动触发 RSS feed 抓取
- 查询最近运行状态和 checkpoint
- 查询失败明细和运行统计
- 启停单个源或整个同步任务

当前仓库中已经存在 issue 相关的 admin gRPC 接口，可以作为后续 RSS admin 能力扩展的基础模式。

### Query API

Query API 面向前端只读查询，建议提供以下能力：

- `ListIssues`
- `GetIssue`
- `ListFeeds`
- `GetFeed`
- `ListFeedContents`
- `GetFeedContent`

对前端来说，所有查询都通过稳定的查询模型暴露，不直接暴露同步任务内部状态或底层存储结构。

## 异常处理与运行约束

同步任务需要具备以下运行特性：

- 单个源失败不应阻塞其他源继续执行。
- 区分“外部抓取失败”和“数据库持久化失败”两类错误。
- 保存最近错误信息、最后成功时间和 checkpoint，便于后台排查。
- 依赖 upsert 实现幂等写入，支持安全重试。
- 在重复抓取、网络抖动或临时故障下保持可恢复性。

## 测试建议

建议测试分为三层：

1. `sync service` 测试
验证增量同步、checkpoint 推进、失败重试、手动触发和状态更新逻辑。

2. `dao service` 测试
分别验证 Mongo 与 Postgres 后端下的 upsert、分页查询、状态读写和统计行为一致性。

3. `api layer` 测试
验证 Admin API 与 Query API 的参数校验、分页返回、详情查询和错误处理。

## 演进原则

- 新增同步对象时，优先扩展 source adapter 和 DAO Service，不直接堆积到既有 service 中。
- 涉及 API 合同变更时，优先更新 `proto/*`，再生成代码并实现服务逻辑。
- 保持 butterfly 当前服务 wiring 与生命周期约定不变。
- 保持数据库后端实现对上层透明，避免业务层直接依赖具体存储细节。

## 总结

`datasrv` 的核心职责是将 GitHub issue 和 RSS feed 内容从外部源同步到数据库，并通过统一的后台管理接口和前端查询接口提供可运营、可展示的数据服务。

在架构上，系统需要明确抽象 `dao service`，将同步编排、接口暴露与底层存储实现解耦，使服务可以在 `mongo` 与 `postgres` 间按配置切换，并为未来扩展更多内容源保留统一的演进路径。
