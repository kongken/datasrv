## ADDED Requirements

### Requirement: PR review storage
The system SHALL store AI-generated PR reviews in a `pr_ai_reviews` table with fields: `id`, `repo`, `issue_id`, `number`, `review_summary`, `risk_areas`, `suggestions`, `raw_diff_size`, `model_used`, `created_at`, `updated_at`. The table SHALL have a unique constraint on `(repo, issue_id)`.

#### Scenario: Review persisted after AI generation
- **WHEN** the PR review job generates a review for a pull request
- **THEN** the review MUST be saved to `pr_ai_reviews` with all fields populated

#### Scenario: Duplicate review prevention
- **WHEN** a review already exists for a given `(repo, issue_id)`
- **THEN** the system SHALL skip that PR unless overwrite mode is enabled

### Requirement: PR review job scheduling
The system SHALL run a `PRReviewService` job on a configurable interval that processes pull requests in batches. The job SHALL only process entries in `github_issues` where `is_pull_request = true` and no corresponding review exists in `pr_ai_reviews`.

#### Scenario: Job processes unreviewed PRs
- **WHEN** the PR review scheduler fires
- **THEN** the job SHALL query for pull requests without reviews, fetch their diffs, generate AI reviews, and persist results

#### Scenario: Job respects batch limits
- **WHEN** there are more unreviewed PRs than `max_prs_per_run`
- **THEN** the job SHALL process at most `max_prs_per_run` PRs and leave the rest for the next run

#### Scenario: Job skips when disabled
- **WHEN** `pr_review.enabled` is `false`
- **THEN** the scheduler SHALL not start the PR review job

### Requirement: PR diff fetching
The system SHALL fetch the unified diff for each pull request from GitHub API using `GET /repos/{owner}/{repo}/pulls/{number}` with diff media type.

#### Scenario: Diff fetched successfully
- **WHEN** the system fetches a PR diff from GitHub
- **THEN** the raw unified diff text SHALL be passed to the AI provider for review

#### Scenario: Diff exceeds size limit
- **WHEN** a PR diff exceeds the configured `max_diff_size` (default 100KB)
- **THEN** the diff SHALL be truncated to the limit and a warning SHALL be logged

#### Scenario: Diff fetch fails
- **WHEN** the GitHub API returns an error for a diff fetch
- **THEN** the system SHALL log the error and skip that PR, continuing with the next one

### Requirement: AI review generation
The system SHALL send the PR diff along with PR metadata (title, body) to a configured AI provider to generate a structured review containing: a summary of changes, identified risk areas, and suggestions.

#### Scenario: Successful review generation
- **WHEN** the AI provider returns a valid response
- **THEN** the system SHALL parse the response into `review_summary`, `risk_areas`, and `suggestions` fields

#### Scenario: AI provider timeout
- **WHEN** the AI provider does not respond within `request_timeout_seconds`
- **THEN** the system SHALL log a timeout error and skip that PR

### Requirement: PR review configuration
The system SHALL support a `pr_review` configuration section with fields: `enabled`, `interval_seconds`, `batch_size`, `max_prs_per_run`, `max_diff_size`, `request_timeout_seconds`, `provider`, `model`, `system_prompt`, and API keys.

#### Scenario: Configuration loaded at startup
- **WHEN** the application starts
- **THEN** the PR review configuration SHALL be loaded from the config file and used to initialize the job

### Requirement: PR review query API
The system SHALL expose API endpoints to list and get PR reviews.

#### Scenario: List reviews by repo
- **WHEN** a client sends `GET /api/v1/pr-reviews?repo={repo}`
- **THEN** the system SHALL return a paginated list of PR reviews for that repository

#### Scenario: Get single review
- **WHEN** a client sends `GET /api/v1/pr-review?repo={repo}&number={number}`
- **THEN** the system SHALL return the review for that specific PR, or 404 if none exists
