## 1. Database Schema

- [x] 1.1 Add `pr_ai_reviews` table to `db/postgres/init.sql` with fields: id, repo, issue_id, number, review_summary, risk_areas, suggestions, raw_diff_size, model_used, created_at, updated_at, and unique constraint on (repo, issue_id)

## 2. DAO Layer

- [x] 2.1 Define `PRReview` model struct and `PRReviewStore` interface in `dao/` with methods: UpsertReview, GetReview, ListReviews, ListUnreviewedPRs
- [x] 2.2 Implement GORM-based `PRReviewStore` with all interface methods

## 3. Configuration

- [x] 3.1 Add `PRReviewConfig` struct to `conf/conf.go` with fields: enabled, interval_seconds, batch_size, max_prs_per_run, max_diff_size, request_timeout_seconds, provider, model, system_prompt, API keys
- [x] 3.2 Wire `PRReviewConfig` into the main config struct and config loading

## 4. GitHub Diff Fetching

- [x] 4.1 Add a method to fetch PR diff via GitHub API (`PullRequests.GetRaw` with diff media type) in the sync service or a new helper
- [x] 4.2 Implement diff size truncation with configurable limit and warning log

## 5. AI Review Generation

- [x] 5.1 Define `PRReviewer` interface with `ReviewPR(ctx, title, body, diff) (ReviewResult, error)` method
- [x] 5.2 Implement AI-based `PRReviewer` using existing provider pattern (OpenAI/Google AI) with PR-specific system prompt that returns structured review (summary, risk areas, suggestions)

## 6. PR Review Job Service

- [x] 6.1 Create `PRReviewService` with `Run(ctx)` method that: loads managed repos, queries unreviewed PRs, fetches diffs, generates reviews, and persists results
- [x] 6.2 Add batch size limiting, timeout handling, and error-skip logic per PR
- [x] 6.3 Wire `PRReviewService` scheduler into `app.go` following the existing ticker + goroutine pattern

## 7. API Endpoints

- [x] 7.1 Add protobuf messages for PR review and RPC methods for ListPRReviews and GetPRReview
- [x] 7.2 Implement gRPC service handlers for PR review query endpoints
- [x] 7.3 Add REST gateway mappings for `GET /api/v1/pr-reviews` and `GET /api/v1/pr-review`

## 8. Testing & Validation

- [x] 8.1 Test PR review job end-to-end: verify it picks up unreviewed PRs, fetches diffs, generates reviews, and persists them
- [x] 8.2 Verify diff truncation works correctly at the configured limit
- [x] 8.3 Verify API endpoints return correct review data
