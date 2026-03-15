## Why

`datasrv` already has a completed change for GitHub issue synchronization, but the service definition in `docs/main.md` also requires RSS/Atom ingestion as a first-class data source. We need this change now so the service can persist feed content, expose it through stable APIs, and complete the second half of its core product scope.

## What Changes

- Add RSS/Atom source management, sync orchestration, and checkpoint tracking so the service can fetch multiple feeds incrementally.
- Persist feed source metadata and feed entry content through the DAO abstraction, with support for the configured datastore backend.
- Add admin gRPC capabilities for managing feed sources, triggering syncs, and inspecting recent sync status and failure details.
- Add query gRPC capabilities for listing feeds, browsing feed contents, and reading a single persisted feed item for frontend use cases.
- Extend service configuration and app wiring so RSS sync can be enabled, scheduled, and observed without disrupting existing butterfly startup conventions.

## Capabilities

### New Capabilities
- `rss-feed-sync`: Synchronize configured RSS/Atom feeds into persistent storage with checkpointed, idempotent upsert behavior.
- `rss-feed-admin-api`: Manage feed sources and operate RSS synchronization from the admin gRPC surface.
- `rss-feed-query-api`: Expose persisted feeds and feed contents through read-only query gRPC endpoints.

### Modified Capabilities

None.

## Impact

- `proto/*` and generated gRPC bindings for new admin/query contracts.
- `service/datasrv/internal/conf/conf.go` and app wiring for RSS sync configuration and scheduling.
- DAO interfaces and concrete datastore implementations for feed sources, feed contents, and sync state.
- New RSS source client, sync service, and tests covering incremental sync, idempotency, and query behavior.
