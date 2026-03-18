BEGIN;

CREATE TABLE IF NOT EXISTS github_issues (
    id BIGSERIAL PRIMARY KEY,
    repo VARCHAR(255) NOT NULL,
    issue_id BIGINT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    state VARCHAR(32) NOT NULL DEFAULT '',
    author VARCHAR(255) NOT NULL DEFAULT '',
    assignees_json TEXT NOT NULL DEFAULT '[]',
    labels_json TEXT NOT NULL DEFAULT '[]',
    comments INTEGER NOT NULL DEFAULT 0,
    is_pull_request BOOLEAN NOT NULL DEFAULT FALSE,
    html_url VARCHAR(1024) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ NULL,
    ai_summary TEXT NOT NULL DEFAULT '',
    raw TEXT NOT NULL DEFAULT '',
    CONSTRAINT uk_github_issues_repo_issue_id UNIQUE (repo, issue_id)
);

CREATE TABLE IF NOT EXISTS github_issue_checkpoints (
    repo VARCHAR(255) PRIMARY KEY,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_issue_updated_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_run_status VARCHAR(32) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS github_sync_repos (
    repo VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rss_feed_sources (
    id VARCHAR(64) PRIMARY KEY,
    url VARCHAR(2048) NOT NULL,
    display_name VARCHAR(512) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    site_url VARCHAR(2048) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    etag VARCHAR(512) NOT NULL DEFAULT '',
    last_modified VARCHAR(512) NOT NULL DEFAULT '',
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_success_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_run_status VARCHAR(32) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_rss_feed_sources_url UNIQUE (url)
);

CREATE TABLE IF NOT EXISTS rss_feed_contents (
    id VARCHAR(80) PRIMARY KEY,
    feed_source_id VARCHAR(64) NOT NULL,
    identity VARCHAR(2048) NOT NULL,
    guid VARCHAR(2048) NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    link VARCHAR(2048) NOT NULL DEFAULT '',
    author VARCHAR(512) NOT NULL DEFAULT '',
    categories_json TEXT NOT NULL DEFAULT '[]',
    published_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uk_rss_feed_contents_source_identity UNIQUE (feed_source_id, identity)
);

CREATE TABLE IF NOT EXISTS rss_feed_checkpoints (
    feed_source_id VARCHAR(64) PRIMARY KEY,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_success_at TIMESTAMPTZ NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    last_run_status VARCHAR(32) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    etag VARCHAR(512) NOT NULL DEFAULT '',
    last_modified VARCHAR(512) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_github_issues_repo_updated_at
    ON github_issues (repo, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_github_issues_state
    ON github_issues (state);

CREATE INDEX IF NOT EXISTS idx_github_issues_created_at
    ON github_issues (created_at);

CREATE INDEX IF NOT EXISTS idx_github_issues_updated_at
    ON github_issues (updated_at);

CREATE INDEX IF NOT EXISTS idx_rss_feed_sources_last_synced_at
    ON rss_feed_sources (last_synced_at);

CREATE INDEX IF NOT EXISTS idx_github_sync_repos_updated_at
    ON github_sync_repos (updated_at);

CREATE INDEX IF NOT EXISTS idx_rss_feed_sources_last_success_at
    ON rss_feed_sources (last_success_at);

CREATE INDEX IF NOT EXISTS idx_rss_feed_sources_updated_at
    ON rss_feed_sources (updated_at);

CREATE INDEX IF NOT EXISTS idx_rss_feed_contents_source_published_at
    ON rss_feed_contents (feed_source_id, published_at DESC, id ASC);

CREATE INDEX IF NOT EXISTS idx_rss_feed_contents_published_at
    ON rss_feed_contents (published_at);

COMMENT ON TABLE github_issues IS 'Synced GitHub issues persisted by datasrv.';
COMMENT ON TABLE github_issue_checkpoints IS 'Per-repository issue sync checkpoints.';
COMMENT ON TABLE github_sync_repos IS 'Managed repository list used by issue sync.';
COMMENT ON TABLE rss_feed_sources IS 'Configured RSS/Atom feed sources.';
COMMENT ON TABLE rss_feed_contents IS 'Normalized feed entries fetched from RSS/Atom sources.';
COMMENT ON TABLE rss_feed_checkpoints IS 'Per-feed sync checkpoints and cache validators.';

COMMIT;
