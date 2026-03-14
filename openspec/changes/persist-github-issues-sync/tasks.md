# Tasks: GitHub Issues Sync Persistence

## 1. Config and Wiring
- [x] 1.1 Extend `service/datasrv/internal/conf/conf.go` with sync and storage config structures.
- [x] 1.2 Wire config loading defaults/env mapping according to butterfly conventions.
- [x] 1.3 Add backend selector (`mongo|postgres`) in app bootstrap without breaking existing startup sequence.

## 2. DAO Abstraction
- [x] 2.1 Create DAO interfaces for issue persistence and sync checkpoint state.
- [x] 2.2 Add shared domain structs (`IssueRecord`, `Checkpoint`, filter/request types).
- [x] 2.3 Refactor call sites to depend on interfaces instead of concrete storage types.

## 3. Mongo Implementation
- [x] 3.1 Implement `IssueStore` for Mongo with upsert by `(repo, issue_id)`.
- [x] 3.2 Implement `SyncStateStore` for Mongo keyed by repo.
- [x] 3.3 Add/verify indexes and basic integration tests.

## 4. Postgres Implementation (GORM)
- [x] 4.1 Add GORM initialization path for Postgres connection and lifecycle handling.
- [x] 4.2 Define GORM models for issues and checkpoints.
- [x] 4.3 Implement upsert and checkpoint persistence using GORM `OnConflict`.
- [x] 4.4 Add migration/init logic for required tables/indexes.
- [x] 4.5 Add Postgres-focused DAO tests.

## 5. GitHub Sync Service
- [x] 5.1 Add GitHub client adapter (token auth, pagination, timeout, since filter).
- [x] 5.2 Implement per-repo sync orchestration with checkpoint resume.
- [x] 5.3 Implement run summary and structured logging.
- [x] 5.4 Add retry/backoff for transient GitHub API failures.

## 6. Admin API
- [x] 6.1 Add `POST /admin/issues/sync` for manual trigger (all repos or selected repo).
- [x] 6.2 Add `GET /admin/issues/sync/config` with secret masking.
- [x] 6.3 Add `GET /admin/issues/sync/status` for last run/checkpoint summary.
- [x] 6.4 Add `PUT /admin/issues/sync/config` only if runtime update is supported; otherwise return explicit non-support response.

## 7. Scheduler Integration
- [x] 7.1 Register periodic sync job in butterfly lifecycle.
- [x] 7.2 Ensure non-overlapping runs.
- [x] 7.3 Respect `enabled=false` and graceful shutdown behavior.

## 8. Validation
- [x] 8.1 Run `go test ./...` and fix failing tests.
- [x] 8.2 Add end-to-end verification notes for both Mongo and Postgres modes.
- [x] 8.3 Document minimal configuration examples for operations handoff.
