# Design: GitHub Issues Sync Persistence

## Overview
Add a synchronization module in `datasrv` that reads a configured repo list, fetches issues from GitHub API, and persists normalized issue records through a DAO interface.

The storage backend is selected by config:
- `mongo`: existing/new Mongo DAO implementation
- `postgres`: new Postgres DAO implementation using GORM

The sync loop supports:
- on-demand trigger (admin API)
- periodic trigger (scheduler)
- per-repository checkpoints to avoid full re-fetch each run

## Architecture

### Components
1. Config (`internal/conf/conf.go`)
- Add `GitHubSyncConfig` with fields:
  - `enabled` (bool)
  - `token` (string, env-bound)
  - `repos` (`[]string`, format: `owner/repo`)
  - `interval` (duration)
  - `page_size` (int)
  - `max_pages_per_run` (int)
  - `request_timeout` (duration)
- Add `StorageConfig` with fields:
  - `driver` (`mongo|postgres`)
  - mongo DSN/db options
  - postgres DSN and gorm options

2. DAO abstraction (`internal/dao`)
- Define interfaces:
  - `IssueStore`
    - `UpsertIssues(ctx, repo string, issues []IssueRecord) error`
    - `ListIssues(ctx, filter IssueFilter) ([]IssueRecord, error)` (optional for admin read)
  - `SyncStateStore`
    - `GetRepoCheckpoint(ctx, repo string) (Checkpoint, error)`
    - `SaveRepoCheckpoint(ctx, repo string, checkpoint Checkpoint) error`
- Optionally compose into `SyncStore` for sync service consumption.

3. Mongo DAO
- Implement interface using existing Mongo stack conventions.
- Unique index on `(repo, issue_id)`.
- Checkpoint collection keyed by repo.

4. Postgres DAO (GORM)
- Add GORM models:
  - `issues`
  - `issue_sync_checkpoints`
- Recommended constraints:
  - unique `(repo, issue_id)` for idempotent upsert
  - index on `(repo, updated_at)`
- Use GORM upsert (`OnConflict`) to update mutable issue fields.

5. GitHub client adapter
- Wrapper around GitHub REST API for issues listing.
- Supports token auth, pagination, and `since` filtering using checkpoint timestamp.

6. Sync service
- Flow for each repo:
  1. load checkpoint
  2. fetch pages from GitHub (bounded by config)
  3. transform to internal `IssueRecord`
  4. upsert batch into DAO
  5. update checkpoint with max observed update timestamp
- Track metrics/logs per run: repos scanned, issues fetched, issues persisted, failures.

7. Admin APIs
- Under admin route group:
  - `POST /admin/issues/sync` trigger immediate sync (all repos or one repo)
  - `GET /admin/issues/sync/config` read current effective sync config (masked token)
  - `PUT /admin/issues/sync/config` update runtime sync config (if service policy allows dynamic update; otherwise reject with guidance)
  - `GET /admin/issues/sync/status` return last run summary/checkpoints

8. Scheduler integration
- Register periodic job during app startup (respect butterfly lifecycle).
- Job guard to prevent overlapping runs (mutex/lease).
- Skip scheduling when sync disabled.

## Data Model

### IssueRecord (logical)
- `repo` string
- `issue_id` int64 (GitHub numeric id)
- `number` int
- `title` string
- `state` string
- `author` string
- `assignees` json/text
- `labels` json/text
- `is_pull_request` bool
- `created_at` time
- `updated_at` time
- `closed_at` nullable time
- `url` string
- `raw` json/blob (optional)

### Checkpoint
- `repo` string
- `last_synced_at` time
- `last_issue_updated_at` time
- `last_run_status` string
- `last_error` string
- `updated_at` time

## Error Handling and Reliability
- Continue sync for remaining repos when one repo fails; aggregate run result.
- Retry transient GitHub errors with bounded backoff.
- Rate-limit awareness:
  - detect quota responses,
  - stop run early with partial success status,
  - keep existing checkpoint unchanged for incomplete repo pass.
- Upsert must be idempotent for repeated runs.

## Security
- GitHub token only from secure config/env.
- Never return raw token in admin responses.
- Admin routes should stay under existing auth/ACL middleware.

## Testing Strategy
- Unit tests for DAO contract using table-driven cases.
- Integration tests:
  - Postgres GORM upsert behavior
  - Mongo upsert behavior
- Sync service tests using mocked GitHub client:
  - pagination
  - checkpoint resume
  - partial failure handling
- API tests for admin endpoints.

## Rollout Plan
1. Add config and DAO abstraction behind feature flag (`github_sync.enabled`).
2. Land Mongo and Postgres implementations.
3. Enable admin trigger endpoint first.
4. Enable scheduler in staging with low-frequency interval.
5. Monitor logs/metrics, then production rollout.
