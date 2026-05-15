# datasrv HTTP API

This document summarizes the HTTP routes exposed by `datasrv` through grpc-gateway.

## Basics

- Default HTTP port: `:8080`
- Default gRPC port: `:9090`
- Response fields follow protobuf JSON mapping, so Go snake_case fields appear as `camelCase`
- `GET` routes read request fields from the query string or path
- `POST`, `PATCH`, and `PUT` routes read request fields from the JSON body unless the path already binds them
- `/api/v1/admin/` routes require `Authorization: Bearer <token>` unless stated otherwise

## Admin Auth

### `POST /api/v1/admin/auth:login`

Log in as an admin user.

Request:

```json
{
  "user": "admin",
  "password": "secret"
}
```

Response:

```json
{
  "success": true,
  "message": "ok",
  "token": "your-admin-token",
  "expiresAt": "2026-03-17T12:00:00Z"
}
```

### `POST /api/v1/admin/auth:logout`

Log out the current admin token.

Request:

```json
{
  "token": "your-admin-token"
}
```

### `GET /api/v1/admin/auth:me`

Read the current admin session.

Typical response:

```json
{
  "user": "admin",
  "expiresAt": "2026-03-17T12:00:00Z"
}
```

## Issue Admin

### `POST /api/v1/admin/issues:sync`

Trigger a sync for one repository or all managed repositories.

Request:

```json
{
  "repo": "owner/repo"
}
```

`repo` may be omitted or empty to sync all managed repos.

### `GET /api/v1/admin/issues/sync-config`

Read the current issue sync config.

### `PATCH /api/v1/admin/issues/sync-config`

Update the current issue sync config.

Request:

```json
{
  "enabled": true,
  "repos": ["owner/repo"],
  "intervalSeconds": 300,
  "pageSize": 100,
  "maxPagesPerRun": 10,
  "requestTimeoutSeconds": 60
}
```

### `GET /api/v1/admin/issues/repos`

List the managed repositories currently tracked by the service.

### `PUT /api/v1/admin/issues/repos`

Replace the full managed repository set.

Request:

```json
{
  "repos": ["owner-a/repo-a", "owner-b/repo-b"]
}
```

### `GET /api/v1/admin/issues/sync-status`

Read the latest sync run status and per-repo checkpoints.

### `PATCH /api/v1/admin/issues/ai-summary`

Update one issue summary manually.

Request by issue number:

```json
{
  "repo": "owner/repo",
  "number": 123,
  "aiSummary": "Updated summary"
}
```

You can also send `issueId` instead of `number`.

### `POST /api/v1/admin/issues/ai-summary:clear`

Clear issue summaries for one repo or all managed repos.

Request:

```json
{
  "repo": "owner/repo"
}
```

## Issue Query

### `GET /api/v1/issues`

List issues.

Common query parameters:

- `repo`: `owner/repo`
- `state`: `open`, `closed`, or `all`
- `page`: 1-based page number
- `pageSize`: page size

Example:

```text
GET /api/v1/issues?repo=owner/repo&state=open&page=1&pageSize=20
```

### `GET /api/v1/issue`

Get one issue by `issueId` or `number`.

Example:

```text
GET /api/v1/issue?repo=owner/repo&number=123
```

## PR Review Query

### `GET /api/v1/pr-reviews`

List AI-generated PR reviews.

Common query parameters:

- `repo`: `owner/repo`
- `page`
- `pageSize`

### `GET /api/v1/pr-review`

Get one PR review by repository and pull request number.

Example:

```text
GET /api/v1/pr-review?repo=owner/repo&number=456
```

## Feed Admin

### `GET /api/v1/admin/feed-sources`

List feed sources.

Common query parameters:

- `page`
- `pageSize`

### `GET /api/v1/admin/feed-sources/{id}`

Get one feed source by ID.

### `POST /api/v1/admin/feed-sources`

Create a feed source.

Request:

```json
{
  "source": {
    "id": "example-feed",
    "url": "https://example.com/feed.xml",
    "displayName": "Example Feed",
    "description": "Example feed source",
    "siteUrl": "https://example.com",
    "enabled": true
  }
}
```

### `PATCH /api/v1/admin/feed-sources`

Update a feed source.

### `DELETE /api/v1/admin/feed-sources/{id}`

Delete a feed source.

### `POST /api/v1/admin/feeds:sync`

Trigger feed sync.

Request:

```json
{
  "feedSourceId": "example-feed"
}
```

`feedSourceId` may be omitted or empty to sync all enabled sources.

### `GET /api/v1/admin/feeds/sync-status`

Read the latest feed sync status.

## Feed Query

### `GET /api/v1/feeds`

List feed sources for public/query consumers.

### `GET /api/v1/feed-contents`

List feed content entries.

Common query parameters:

- `feedSourceId`
- `page`
- `pageSize`

### `GET /api/v1/feed-contents/{id}`

Get one feed content entry.

## Blog Query

### `GET /api/v1/blog/posts`

List blog posts.

Common query parameters:

- `page`
- `pageSize`
- `status`
- `tag`
- `query`

### `GET /api/v1/blog/posts/{slug}`

Get one blog post by slug.

### `GET /api/v1/blog/posts/{post_slug}/comments`

List comments for one blog post.

Common query parameters:

- `page`
- `pageSize`
- `status`

### `POST /api/v1/blog/posts/{post_slug}/comments`

Create a comment for one blog post.

## Blog Admin

### `POST /api/v1/admin/blog/posts`

Create a blog post.

### `PATCH /api/v1/admin/blog/posts`

Update a blog post.

### `DELETE /api/v1/admin/blog/posts/{id}`

Delete a blog post.

### `GET /api/v1/admin/blog/comments/{id}`

Get one blog comment.

### `PATCH /api/v1/admin/blog/comments`

Update a blog comment.

### `DELETE /api/v1/admin/blog/comments/{id}`

Delete a blog comment.

## Admin auth errors

Admin auth failures are returned as structured JSON errors. Common codes:

- `admin_auth_missing_token`
- `admin_auth_invalid_token`
- `admin_auth_validation_failed`
- `admin_auth_store_unavailable`

## Source of truth

This document is derived from the grpc-gateway annotations in:

- `proto/issues/v1/issue.proto`
- `proto/feeds/v1/feed.proto`
- `proto/blog/v1/post.proto`
