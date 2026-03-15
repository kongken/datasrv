# Proposal: Persist GitHub Issues to Configured Datastores

## Summary
Build a datasource sync capability in `datasrv` that periodically pulls GitHub issues from configured repositories and persists them into database backends.

The service must support:
- MongoDB persistence
- PostgreSQL persistence (implemented with GORM)
- DAO layer abstraction for pluggable datastore implementations
- Configurable GitHub repository list
- Admin APIs to manage sync configuration and trigger sync
- Scheduled issue sync jobs

## Motivation
Current service does not provide an integrated way to continuously synchronize GitHub issues into internal storage. Teams need a single service that can:
- ingest issue metadata from multiple repos,
- persist data in different infra choices (Mongo/Postgres),
- offer operational controls via admin endpoints,
- and run on schedule without manual intervention.

## Goals
- Introduce a clean DAO abstraction for issues and sync state.
- Provide Mongo and Postgres implementations behind the same interface.
- Enforce Postgres implementation with GORM.
- Load GitHub repos and sync settings from config.
- Expose admin APIs for operational control.
- Add scheduler-driven periodic synchronization.

## Non-Goals
- Full GitHub webhook ingestion in this change.
- UI dashboard for admin management.
- Data warehouse/analytics schema modeling beyond issue sync needs.

## Scope
- Config model extension for GitHub sync settings and datastore selection.
- DAO interfaces and two concrete storage adapters.
- Sync service handling pagination, upsert semantics, and checkpointing.
- Admin routes for sync trigger and configuration read/update (as permitted by service runtime model).
- Background scheduler integration within existing butterfly app lifecycle.

## Success Criteria
- Service can start with either Mongo or Postgres backend from config.
- Postgres path uses GORM for CRUD and upsert.
- Sync job fetches issues for each configured repo and persists idempotently.
- Admin endpoint can trigger on-demand sync and return execution summary.
- Automated tests cover DAO contract and core sync flow.
