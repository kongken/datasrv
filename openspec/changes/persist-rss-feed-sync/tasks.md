## 1. Protobuf Contracts

- [x] 1.1 Add `proto/feeds/v1/feed.proto` with RSS admin and query services, feed source models, feed content models, sync status messages, and request/response types.
- [x] 1.2 Update protobuf generation config if needed and run `buf generate` so the new Go bindings under `pkg/proto/*` are committed with the change.
- [x] 1.3 Add or update protobuf-focused validation/tests to ensure the generated contracts compile cleanly with the existing issue APIs.

## 2. Configuration and App Wiring

- [x] 2.1 Extend `service/datasrv/internal/conf/conf.go` with RSS sync configuration and feed source settings while preserving existing butterfly config conventions.
- [x] 2.2 Refactor `service/datasrv/internal/app.go` initialization so issue and feed components can be wired together without changing `core.New(&app.Config{...})` startup conventions.
- [x] 2.3 Add scheduler coordination and teardown handling for RSS sync jobs with single-flight protection and `enabled=false` behavior.

## 3. Feed Persistence Layer

- [x] 3.1 Define feed-specific DAO interfaces and domain models for feed sources, feed contents, and feed sync checkpoints.
- [x] 3.2 Implement the Postgres path for feed source/content/status persistence, including indexes and upsert behavior for deterministic item identity.
- [x] 3.3 Implement the Mongo path for the same feed persistence contract and ensure behavior matches the Postgres implementation.
- [x] 3.4 Add DAO tests covering source CRUD, idempotent content upsert, reverse-chronological listing, and checkpoint persistence.

## 4. RSS Sync Service

- [x] 4.1 Add an RSS/Atom fetch-and-parse adapter with timeout handling and support for `etag` / `last-modified` request metadata.
- [x] 4.2 Implement feed sync orchestration that loads enabled sources, fetches content, normalizes entries, persists source/content updates, and records per-source run results.
- [x] 4.3 Implement failure handling that preserves the last successful checkpoint, records source-specific errors, and continues processing remaining sources.
- [x] 4.4 Add sync service tests for all-source runs, single-source runs, duplicate entry upserts, and partial failure behavior.

## 5. gRPC Services

- [x] 5.1 Implement `FeedSyncAdminService` handlers for feed source management, manual sync triggers, and sync status inspection using the generated protobuf contracts.
- [x] 5.2 Implement `FeedQueryService` handlers for feed listing, feed content pagination, and single-item detail reads with proper validation and gRPC error codes.
- [x] 5.3 Register the new feed gRPC servers alongside the existing issue services and add handler tests for admin and query flows.

## 6. Verification

- [x] 6.1 Run `go test ./...` and fix any regressions introduced by the RSS change.
- [ ] 6.2 Validate protobuf generation and service startup expectations for both configured storage backends.
- [x] 6.3 Document minimal RSS configuration and operational verification notes for manual sync, scheduled sync, and query usage.
