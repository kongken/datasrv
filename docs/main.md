# datasrv service overview

`datasrv` is a content sync service. It pulls data from external systems, stores normalized records in a configurable backend, and exposes those records through gRPC services with grpc-gateway HTTP routes.

## What the service handles

The current runtime includes:

- GitHub issue synchronization and query
- RSS/Atom feed synchronization and query
- Admin login and token validation
- AI issue summary generation
- AI PR review retrieval
- Blog post and comment APIs

## Runtime composition

Service entrypoints:

- bootstrap: `service/datasrv/cmd/main.go`
- app wiring: `service/datasrv/internal/app.go`
- config schema: `service/datasrv/internal/conf/conf.go`

Key layers:

1. Source clients
   - GitHub API access in `pkg/github`
   - feed fetching and parsing in `service/datasrv/internal/service`
2. Sync services
   - issue sync, feed sync, summary jobs, and PR review jobs
3. DAO layer
   - storage interfaces in `service/datasrv/internal/dao`
   - PostgreSQL and MongoDB implementations behind shared interfaces
4. API layer
   - protobuf contracts in `proto/*`
   - generated code in `pkg/proto/*`
   - gRPC registration and HTTP gateway wiring in `service/datasrv/internal`

## Storage model

The service supports:

- PostgreSQL through GORM and Ent-related schema artifacts
- MongoDB through dedicated DAO implementations

The active driver is selected from `storage.driver`. If `storage` is empty, the legacy `database` section is still accepted.

## Main data flows

### GitHub issues

1. Load managed repositories from config and persisted state.
2. Fetch issue pages from GitHub.
3. Resume from stored checkpoints where possible.
4. Normalize issue fields, users, labels, milestones, and comments.
5. Persist issue data through the sync store.
6. Expose admin operations and read APIs over gRPC and HTTP.

Optional additions:

- store full comments in S3-compatible object storage
- generate AI summaries for synced issues
- generate AI reviews for pull requests

### RSS feeds

1. Load configured feed sources.
2. Fetch RSS or Atom payloads.
3. Track `etag` and `last-modified` when available.
4. Normalize feed metadata and content entries.
5. Persist feed sources, sync state, and entries.
6. Expose feed admin and query APIs.

### Blog content

Blog APIs run inside the same service process and share the same HTTP and gRPC surface, which makes `datasrv` usable as both a sync backend and a content API backend.

## Modules worth knowing

- `service/datasrv/internal/service/admin_grpc.go`: issue admin APIs
- `service/datasrv/internal/service/feed_admin_grpc.go`: feed admin APIs
- `service/datasrv/internal/service/issue_query_grpc.go`: issue query APIs
- `service/datasrv/internal/service/feed_query_grpc.go`: feed query APIs
- `service/datasrv/internal/service/blog_grpc.go`: blog APIs
- `service/datasrv/internal/dao/gorm_sync_store.go`: PostgreSQL-backed sync store
- `service/datasrv/internal/dao/mongo_sync_store.go`: MongoDB-backed sync store

## Related docs

- [`setup.md`](setup.md)
- [`http-api.md`](http-api.md)
