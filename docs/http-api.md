# Datasrv HTTP API

本文档整理了 `datasrv` 通过 grpc-gateway 暴露的 HTTP 接口，方便前端直接对接。

## 基础说明

- HTTP 服务默认监听 `:8080`
- gRPC 服务默认监听 `:9090`
- HTTP 接口由 grpc-gateway 转发到本地 gRPC service
- 响应字段遵循 protobuf JSON 映射规则，通常为 `camelCase`
- `GET` 接口的请求字段通过 query string 传递
- `POST` / `PATCH` 接口按 JSON body 传递
- `/api/v1/admin/` 下的 HTTP 接口默认需要 `Authorization: Bearer <token>`
- `POST /api/v1/admin/auth:login` 不需要 token，用于换取登录 token

## Admin Auth

### `POST /api/v1/admin/auth:login`

管理员登录鉴权。

请求体示例：

```json
{
  "user": "admin",
  "password": "secret"
}
```

响应体示例：

```json
{
  "success": true,
  "message": "ok",
  "token": "your-admin-token",
  "expiresAt": "2026-03-17T12:00:00Z"
}
```

登录成功后，前端需要把返回的 `token` 放进请求头：

```text
Authorization: Bearer your-admin-token
```

### `POST /api/v1/admin/auth:logout`

管理员登出，删除当前 token。

请求头：

```text
Authorization: Bearer your-admin-token
```

请求体示例：

```json
{
  "token": "your-admin-token"
}
```

响应体示例：

```json
{
  "success": true,
  "message": "ok"
}
```

### `GET /api/v1/admin/auth:me`

读取当前 Bearer token 对应的管理员登录态。

请求头：

```text
Authorization: Bearer your-admin-token
```

响应体示例：

```json
{
  "user": "admin",
  "expiresAt": "2026-03-17T12:00:00Z"
}
```

## Issue Admin

### `POST /api/v1/admin/issues:sync`

触发 issue 同步。

请求体示例：

```json
{
  "repo": "owner/repo"
}
```

`repo` 可为空，表示同步配置里的全部仓库。

### `GET /api/v1/admin/issues/sync-config`

获取当前 issue 同步配置。

### `PATCH /api/v1/admin/issues/sync-config`

更新当前 issue 同步配置。

请求体示例：

```json
{
  "enabled": true,
  "repos": ["owner/repo"],
  "intervalSeconds": 300,
  "pageSize": 50,
  "maxPagesPerRun": 5,
  "requestTimeoutSeconds": 10
}
```

### `GET /api/v1/admin/issues/sync-status`

获取最近一次同步状态和 checkpoint 信息。

### `PATCH /api/v1/admin/issues/ai-summary`

更新某个 issue 的 AI 摘要。

请求体示例：

```json
{
  "repo": "owner/repo",
  "number": 123,
  "aiSummary": "这是新的摘要"
}
```

也可以传 `issueId` 代替 `number`。

## Issue Query

### `GET /api/v1/issues`

分页查询 issue 列表。

常用 query 参数：

- `repo`: 仓库，格式 `owner/repo`
- `state`: `open`、`closed`、`all`
- `page`: 1 开始
- `pageSize`: 页大小

示例：

```text
GET /api/v1/issues?repo=owner/repo&state=open&page=1&pageSize=20
```

### `GET /api/v1/issue`

查询单个 issue。

常用 query 参数：

- `repo`: 仓库，格式 `owner/repo`
- `issueId`: issue ID
- `number`: issue 编号

示例：

```text
GET /api/v1/issue?repo=owner/repo&number=123
```

`issueId` 和 `number` 二选一。

## Feed Admin

### `GET /api/v1/admin/feed-sources`

分页查询 feed source 列表。

常用 query 参数：

- `page`
- `pageSize`

### `GET /api/v1/admin/feed-sources/{id}`

获取单个 feed source。

### `POST /api/v1/admin/feed-sources`

创建 feed source。

请求体示例：

```json
{
  "source": {
    "url": "https://example.com/feed.xml",
    "displayName": "Example Feed",
    "description": "Example feed source",
    "siteUrl": "https://example.com",
    "enabled": true
  }
}
```

### `PATCH /api/v1/admin/feed-sources`

更新 feed source。

请求体示例：

```json
{
  "source": {
    "id": "feed-source-id",
    "url": "https://example.com/feed.xml",
    "displayName": "Example Feed",
    "description": "Updated description",
    "siteUrl": "https://example.com",
    "enabled": true
  }
}
```

### `DELETE /api/v1/admin/feed-sources/{id}`

删除 feed source。

### `POST /api/v1/admin/feeds:sync`

触发 feed 同步。

请求体示例：

```json
{
  "feedSourceId": "feed-source-id"
}
```

`feedSourceId` 可为空，表示同步全部已启用 source。

### `GET /api/v1/admin/feeds/sync-status`

获取最近一次 feed 同步状态。

## Feed Query

### `GET /api/v1/feeds`

分页查询 feed source 列表。

常用 query 参数：

- `page`
- `pageSize`

### `GET /api/v1/feed-contents`

分页查询 feed 内容列表。

常用 query 参数：

- `feedSourceId`
- `page`
- `pageSize`

示例：

```text
GET /api/v1/feed-contents?feedSourceId=feed-source-id&page=1&pageSize=20
```

### `GET /api/v1/feed-contents/{id}`

获取单条 feed 内容详情。

## 备注

- 当前文档基于 proto 中的 grpc-gateway 路由注解整理。
- 如果后续 proto 路径或字段名变化，需要同步更新本文档。

## Admin Auth Errors

admin HTTP 鉴权失败时，响应体统一为：

```json
{
  "code": "admin_auth_missing_token",
  "message": "missing bearer token"
}
```

当前会返回的常见错误码：

- `admin_auth_missing_token`: 缺少 `Authorization: Bearer <token>`
- `admin_auth_invalid_token`: token 不存在或已失效
- `admin_auth_validation_failed`: Redis 校验失败或服务异常
- `admin_auth_store_unavailable`: token store 未初始化
