# End-to-End Verification Notes

## Common Preconditions
- Ensure `github.token` is set to a valid PAT with repo read permissions.
- Ensure `github_sync.repos` contains valid `owner/repo` identifiers.
- Start service with updated config.
- gRPC server runs on default butterfly gRPC port `:9090`.

## PostgreSQL (GORM) Mode
1. Configure `storage.driver=postgres` and `storage.postgres_dsn`.
2. Start service.
3. Call `issues.v1.IssueSyncAdminService/SyncIssues` with empty `repo` to sync all configured repos.
4. Verify rows in:
   - `github_issues`
   - `github_issue_checkpoints`
5. Call `GetSyncStatus` and confirm checkpoints and last run results.

Expected:
- `github_issues` has upserted records keyed by `(repo, issue_id)`.
- rerunning `SyncIssues` updates existing rows without duplicates.

## MongoDB Mode
1. Configure `storage.driver=mongo`, `storage.mongo_uri`, `storage.mongo_db`.
2. Start service.
3. Call `SyncIssues` with empty `repo`.
4. Verify documents in collections:
   - `github_issues`
   - `github_issue_checkpoints`
5. Call `GetSyncStatus` and confirm checkpoint/state returned.

Expected:
- unique index `(repo, issue_id)` prevents duplicates.
- rerunning sync is idempotent with updates applied to existing docs.

## Scheduler Validation
1. Set `github_sync.enabled=true` and a short `interval_seconds` (e.g. 30).
2. Start service and wait for at least one interval.
3. Confirm data/checkpoints are updated without manual trigger.
4. Trigger `SyncIssues` while scheduler is running and verify no overlapping run error is returned when already in progress.
