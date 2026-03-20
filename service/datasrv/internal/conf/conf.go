package conf

var (
	Conf = new(Config)
)

// Config holds all configuration for the datasrv service
type Config struct {
	// Database configuration (legacy key, same structure as storage)
	Database StorageConfig `yaml:"database" json:"database"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage" json:"storage"`

	// GitHub configuration
	GitHub GitHubConfig `yaml:"github" json:"github"`

	// GitHub issue sync job configuration
	GitHubSync GitHubSyncConfig `yaml:"github_sync" json:"github_sync"`

	// IssueCommentStorage controls where full GitHub issue comments are persisted.
	IssueCommentStorage IssueCommentStorageConfig `yaml:"issue_comment_storage" json:"issue_comment_storage"`

	// RSS feed sync job configuration
	FeedSync FeedSyncConfig `yaml:"feed_sync" json:"feed_sync"`

	// IssueSummary controls periodic AI summary generation for synced issues.
	IssueSummary IssueSummaryConfig `yaml:"issue_summary" json:"issue_summary"`

	// Server configuration
	Server ServerConfig `yaml:"server" json:"server"`

	// Admin login configuration
	Admin AdminConfig `yaml:"admin" json:"admin"`
}

func (Config) Print() {}

// StorageConfig holds datastore configuration.
type StorageConfig struct {
	// Driver specifies the persistence backend (mongo, postgres).
	Driver string `yaml:"driver" json:"driver"`

	// DSN is the default data source name.
	DSN string `yaml:"dsn" json:"dsn"`

	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`

	// MongoURI is the MongoDB URI.
	MongoURI string `yaml:"mongo_uri" json:"mongo_uri"`

	// MongoDB is the Mongo database name.
	MongoDB string `yaml:"mongo_db" json:"mongo_db"`

	// PostgresDSN is the PostgreSQL DSN.
	PostgresDSN string `yaml:"postgres_dsn" json:"postgres_dsn"`
}

// GitHubConfig holds GitHub API configuration
type GitHubConfig struct {
	// Token is the GitHub personal access token for API authentication
	Token string `yaml:"token" json:"token"`

	// BaseURL is the GitHub API base URL (for GitHub Enterprise)
	BaseURL string `yaml:"base_url" json:"base_url"`
}

// GitHubSyncConfig holds scheduled sync options.
type GitHubSyncConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Repos in owner/repo format.
	Repos []string `yaml:"repos" json:"repos"`

	// IntervalSeconds controls scheduler frequency.
	IntervalSeconds int `yaml:"interval_seconds" json:"interval_seconds"`

	// PageSize controls per-page fetch size.
	PageSize int `yaml:"page_size" json:"page_size"`

	// MaxPagesPerRun bounds per-repo work in one run.
	MaxPagesPerRun int `yaml:"max_pages_per_run" json:"max_pages_per_run"`

	// RequestTimeoutSeconds controls the timeout for each outbound GitHub/S3 request.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds" json:"request_timeout_seconds"`
}

// IssueCommentStorageConfig stores full GitHub issue comments outside the primary database.
type IssueCommentStorageConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Provider currently supports "s3".
	Provider string `yaml:"provider" json:"provider"`

	Bucket string `yaml:"bucket" json:"bucket"`
	Region string `yaml:"region" json:"region"`

	// Endpoint supports S3-compatible object stores such as MinIO.
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key"`
	UsePathStyle    bool   `yaml:"use_path_style" json:"use_path_style"`
	KeyPrefix       string `yaml:"key_prefix" json:"key_prefix"`
}

// FeedSourceConfig defines a configured RSS/Atom source.
type FeedSourceConfig struct {
	ID          string `yaml:"id" json:"id"`
	URL         string `yaml:"url" json:"url"`
	DisplayName string `yaml:"display_name" json:"display_name"`
	Description string `yaml:"description" json:"description"`
	SiteURL     string `yaml:"site_url" json:"site_url"`
	Enabled     bool   `yaml:"enabled" json:"enabled"`
}

// FeedSyncConfig holds scheduled RSS sync options.
type FeedSyncConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// IntervalSeconds controls scheduler frequency.
	IntervalSeconds int `yaml:"interval_seconds" json:"interval_seconds"`

	// RequestTimeoutSeconds controls outbound RSS fetch timeout.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds" json:"request_timeout_seconds"`

	// Sources seeds feed source definitions into the backing store.
	Sources []FeedSourceConfig `yaml:"sources" json:"sources"`
}

// IssueSummaryConfig holds scheduled issue AI summary generation options.
type IssueSummaryConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// IntervalSeconds controls scheduler frequency.
	IntervalSeconds int `yaml:"interval_seconds" json:"interval_seconds"`

	// BatchSize controls how many issues to load per page.
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// MaxIssuesPerRun bounds how many issues are scanned in one run.
	MaxIssuesPerRun int `yaml:"max_issues_per_run" json:"max_issues_per_run"`

	// RequestTimeoutSeconds controls timeout for each model generation request.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds" json:"request_timeout_seconds"`

	// State optionally filters issues by state. Empty means all.
	State string `yaml:"state" json:"state"`

	// OverwriteExisting controls whether non-empty ai_summary values are regenerated.
	OverwriteExisting bool `yaml:"overwrite_existing" json:"overwrite_existing"`

	// Provider selects the Genkit provider, such as "openai" or "googleai".
	Provider string `yaml:"provider" json:"provider"`

	// Model is the provider model id. Short names are expanded with the provider prefix.
	Model string `yaml:"model" json:"model"`

	// SystemPrompt overrides the default summarization system prompt.
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`

	OpenAIAPIKey  string `yaml:"openai_api_key" json:"openai_api_key"`
	OpenAIBaseURL string `yaml:"openai_base_url" json:"openai_base_url"`
	GoogleAPIKey  string `yaml:"google_api_key" json:"google_api_key"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	// Host is the server host address
	Host string `yaml:"host" json:"host"`

	// Port is the server port
	Port int `yaml:"port" json:"port"`
}

// AdminConfig holds admin login credentials for management access.
type AdminConfig struct {
	// User is the admin login username.
	User string `yaml:"user" json:"user"`

	// Password is the admin login password.
	Password string `yaml:"password" json:"password"`

	// RedisName is the named Redis client used to persist admin auth tokens.
	RedisName string `yaml:"redis_name" json:"redis_name"`

	// TokenTTLSeconds controls how long issued admin tokens remain valid.
	TokenTTLSeconds int `yaml:"token_ttl_seconds" json:"token_ttl_seconds"`
}
