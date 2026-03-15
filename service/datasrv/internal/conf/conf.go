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

	// RSS feed sync job configuration
	FeedSync FeedSyncConfig `yaml:"feed_sync" json:"feed_sync"`

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

	// RequestTimeoutSeconds controls GitHub API timeout.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds" json:"request_timeout_seconds"`
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
}
