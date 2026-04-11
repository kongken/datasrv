## Context

The datasrv service currently syncs GitHub issues and pull requests into `github_issues` table, distinguishing them via `is_pull_request` boolean. An existing `IssueSummaryService` job generates AI summaries for issues. Pull requests contain code diffs that can be reviewed by AI to provide quick insights for maintainers. We need a parallel job that fetches PR diffs and generates structured review summaries.

The existing architecture uses time-based schedulers in `app.go`, GORM-based DAO layer, and pluggable AI providers (OpenAI, Google AI). The new job should follow the same patterns.

## Goals / Non-Goals

**Goals:**
- Generate AI review summaries for synced pull requests automatically
- Persist reviews in a dedicated table for querying
- Expose reviews via API endpoints
- Follow existing job architecture patterns (scheduler, DAO, service layers)

**Non-Goals:**
- Posting reviews back to GitHub as PR comments
- Real-time webhook-triggered reviews (batch job only)
- Reviewing closed/merged PRs that were synced before this feature (only process going forward, unless configured to backfill)
- Inline code annotations (summary-level review only)

## Decisions

### 1. Separate `pr_ai_reviews` table vs extending `ai_summary` field

**Decision**: New `pr_ai_reviews` table.

**Rationale**: PR reviews contain structured data (summary, risk areas, suggestions) that differs from issue summaries. A separate table avoids overloading the `ai_summary` text field and allows independent querying. Foreign key on `(repo, issue_id)` links back to `github_issues`.

**Alternative**: Store in `ai_summary` as JSON — simpler but loses queryability and mixes concerns.

### 2. Fetching PR diffs via GitHub API

**Decision**: Use GitHub's `GET /repos/{owner}/{repo}/pulls/{number}` with `Accept: application/vnd.github.diff` header to get the unified diff.

**Rationale**: The diff format is compact and directly useful for AI review. The go-github library supports this via `PullRequests.GetRaw()`.

**Alternative**: Fetch individual file patches — more granular but more API calls and complex assembly.

### 3. Diff size limit

**Decision**: Truncate diffs exceeding a configurable limit (default 100KB) before sending to AI. Log a warning when truncation occurs.

**Rationale**: Large diffs can exceed AI context windows and produce poor reviews. Truncation with a warning is pragmatic.

### 4. Reuse existing AI provider abstraction

**Decision**: Create a `PRReviewer` interface similar to `IssueSummarizer`, reusing the same provider configuration pattern but with a PR-specific system prompt.

**Rationale**: Consistent architecture, minimal new code. The PR review job gets its own config section to allow different model/prompt than issue summaries.

### 5. Job scheduling pattern

**Decision**: Follow the same goroutine + ticker pattern used by `IssueSummaryService` in `app.go`.

**Rationale**: Consistency with existing codebase. No need to introduce a job queue for a single additional periodic task.

## Risks / Trade-offs

- **GitHub API rate limiting** → The diff fetch adds one API call per PR. Mitigated by batch size limits and configurable intervals.
- **Large diffs producing poor reviews** → Truncation at configurable limit. Future improvement: smart truncation by file importance.
- **AI cost** → Each PR review is an AI API call. Mitigated by `enabled` flag, batch limits, and skip-if-exists logic.
- **Stale reviews after PR updates** → Reviews are generated once. A future enhancement could re-review on PR update by checking `updated_at` changes.
