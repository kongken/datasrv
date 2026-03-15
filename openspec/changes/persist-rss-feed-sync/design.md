## Context

`datasrv` already ships GitHub issue synchronization with config-driven storage selection, gRPC admin controls, and query endpoints. The service definition in `docs/main.md` requires the same platform to support RSS/Atom feeds, but the repository currently has no feed proto contracts, no persistence model for feed content, and no sync orchestration outside the issue path.

This change crosses multiple layers at once: protobuf API contracts, generated bindings, config, butterfly app wiring, sync orchestration, and datastore implementations. The existing issue flow provides a useful reference pattern, but RSS introduces different data semantics such as feed source management, item-level idempotency, and HTTP cache validators (`etag` and `last-modified`) that must be tracked separately from GitHub-style `updated_at` checkpoints.

## Goals / Non-Goals

**Goals:**
- Add a first-class RSS sync domain that can manage multiple feed sources, fetch RSS/Atom data, and persist feed metadata plus feed entries.
- Reuse the current butterfly bootstrap and datastore selection model so RSS support fits the existing service shape instead of creating a parallel subsystem.
- Define admin and query gRPC contracts for RSS operations before implementation, following the repository rule that API changes start from `proto/*`.
- Support incremental, idempotent sync behavior with per-feed status, last success metadata, and recoverable failures.
- Keep the design compatible with the current persistence abstraction so Mongo and Postgres implementations can remain behaviorally aligned.

**Non-Goals:**
- Building a generalized ingestion framework for arbitrary future sources beyond RSS/Atom.
- Adding webhooks, push delivery, or full-text search in this change.
- Designing a UI or external gateway contract beyond the gRPC APIs owned by this service.
- Reworking the completed GitHub issue sync flow unless shared abstractions must be widened for RSS support.

## Decisions

### 1. Introduce a dedicated RSS proto package and service surface

RSS contracts will live under a new `proto/feeds/v1/feed.proto` package with two service groups:
- `FeedSyncAdminService` for source management, manual sync, and status inspection.
- `FeedQueryService` for feed and feed content reads.

This mirrors the existing issue split between admin and query surfaces and keeps feed-specific payloads isolated from issue models.

Alternatives considered:
- Extend `issues.v1` with feed messages and RPCs: rejected because it mixes unrelated resource types and makes generated clients harder to reason about.
- Delay proto work until after storage/service code: rejected because repository rules require API-first changes.

### 2. Split persistence into source configuration, content storage, and sync state

The existing `SyncStore` abstraction is issue-centric. RSS support will add feed-specific models and either widen the DAO interface or introduce a composed feed store abstraction with three responsibilities:
- `FeedSourceStore`: CRUD and listing for managed feed sources.
- `FeedContentStore`: upsert and query for persisted feed items.
- `FeedSyncStateStore`: checkpoint/status persistence for each source.

The preferred implementation is a composed interface so issue code can remain stable while RSS code depends only on the narrower contracts it needs.

Alternatives considered:
- Reuse the existing issue `SyncStore` for everything: rejected because it would overload issue-specific types and query filters.
- Create one monolithic `RSSStore` with all methods: acceptable but less explicit; composition keeps ownership clearer and tests smaller.

### 3. Use feed source IDs plus content fingerprints for idempotent writes

Each managed feed source will have a stable internal ID and configuration record. Persisted feed items will use a deterministic uniqueness key based on the source ID plus the best available item identity in this order:
1. canonical entry ID/guid
2. item link
3. title + published timestamp hash

The sync state record will persist the latest successful fetch time, last HTTP validators (`etag`, `last_modified`), last run status, and the most recent error.

Alternatives considered:
- Rely only on published timestamps for deduplication: rejected because many feeds revise content without changing publish time, and some feeds omit timestamps.
- Store raw feed payloads only and derive entries at query time: rejected because frontend queries need stable, indexed content entities.

### 4. Keep sync orchestration source-local and non-blocking

The RSS sync service will iterate configured, enabled feed sources one by one. A failure on one source will record a failed run for that source and continue to others. Manual sync requests may target all sources or a single source. Scheduler execution will remain single-flight at the service level to avoid overlapping runs.

Alternatives considered:
- Parallelize source sync by default: deferred because it complicates rate limiting, datastore contention, and failure reporting without being required for the first change.
- Fail the whole run on first source error: rejected because the product requirement is operational resilience across multiple sources.

### 5. Reuse existing butterfly wiring with additive initialization

RSS components will be wired through the existing `core.New(&app.Config{...})` lifecycle. Initialization will add feed DAO/service/server registration without changing startup conventions. Scheduler registration should either support both issue and feed jobs or evolve into a small coordinator that starts enabled jobs during init and stops them during teardown.

Alternatives considered:
- Introduce a second application entrypoint or a bespoke worker process: rejected because it violates the current service deployment model.

## Risks / Trade-offs

- [Proto and generated code expansion] -> Mitigation: keep RSS messages in a separate package and regenerate bindings in the same change with `buf generate`.
- [DAO abstraction drift between issue and feed domains] -> Mitigation: add feed-specific interfaces instead of forcing issue types to become generic too early.
- [Weak item identity on low-quality feeds] -> Mitigation: define a deterministic fallback key order and persist the raw source identifiers for debugging.
- [Scheduler overlap between issue sync and feed sync] -> Mitigation: use explicit per-job single-flight guards and keep ticker registration centralized.
- [Storage schema growth] -> Mitigation: separate feed source, feed content, and feed checkpoint models so indexes stay targeted to query patterns.

## Migration Plan

1. Add feed protobuf definitions and regenerate code.
2. Extend config types with RSS sync settings and source defaults.
3. Add feed DAO contracts plus Mongo/Postgres implementations and indexes.
4. Implement RSS fetch/parser adapter and sync service with checkpoint/status persistence.
5. Register admin/query gRPC servers and scheduler wiring under the existing app lifecycle.
6. Roll out with RSS sync disabled by default or with no configured sources so deployment is backward compatible.
7. Validate on a small set of feeds, then enable production sources.

Rollback:
- Disable RSS sync in config and redeploy.
- Keep the protobuf/API additions in place; they are additive and can return empty results when no sources are configured.
- If a schema migration causes issues, stop RSS startup wiring while leaving existing issue functionality untouched.

## Open Questions

- Should feed source configuration be runtime-mutable through admin RPCs only, or also persisted back into service config for reboot durability?
- Which Go RSS/Atom parser library best matches the project’s dependency and maintenance preferences?
- Do frontend consumers need source-level filtering beyond feed ID in the first query API revision, such as category or time-window filters?
