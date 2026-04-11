## Why

The system already syncs GitHub issues and pull requests into `github_issues` with an `is_pull_request` flag, and generates AI summaries for issues. However, pull requests contain rich diff/change information that issues don't have. We need a dedicated job that fetches PR diffs and uses AI to generate code review summaries, giving maintainers quick insight into what each PR changes without reading the full diff.

## What Changes

- Add a new `pr_ai_reviews` database table to store AI-generated review summaries for pull requests
- Add a new scheduled job (`PRReviewService`) that:
  - Queries `github_issues` for entries where `is_pull_request = true` and no review exists yet
  - Fetches the PR diff from GitHub API
  - Sends the diff to an AI provider to generate a review summary (change description, risk areas, suggestions)
  - Persists the review to the new table
- Add DAO interface and GORM implementation for the review store
- Add configuration for the PR review job (interval, batch size, AI provider/model, enabled flag)
- Add API endpoints to query PR reviews
- Wire the new scheduler into `app.go`

## Capabilities

### New Capabilities
- `pr-ai-review`: AI-powered pull request review generation job, including diff fetching, AI summarization, persistence, and query API

### Modified Capabilities

## Impact

- **Database**: New `pr_ai_reviews` table with foreign key to `github_issues`
- **GitHub API**: Additional API calls to fetch PR diffs (rate limit consideration)
- **AI Provider**: Additional AI API calls for review generation
- **Config**: New `pr_review` section in `config.yml`
- **API**: New endpoints for listing/getting PR reviews
- **Scheduler**: New goroutine in `app.go` for the review job
