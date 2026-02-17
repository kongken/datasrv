package conf

var (
	Conf = new(Config)
)

// Config holds all configuration for the datasrv service
type Config struct {
	// Database configuration
	Database DatabaseConfig `yaml:"database" json:"database"`

	// GitHub configuration
	GitHub GitHubConfig `yaml:"github" json:"github"`

	// Server configuration
	Server ServerConfig `yaml:"server" json:"server"`
}

func (Config) Print() {}

type GithubConfig struct {
	Token string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Driver specifies the database driver (postgres, mongodb, etc.)
	Driver string `yaml:"driver" json:"driver"`

	// DSN is the data source name for the database connection
	DSN string `yaml:"dsn" json:"dsn"`

	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`

	// MaxIdleConns is the maximum number of connections in the idle connection pool
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`
}

// GitHubConfig holds GitHub API configuration
type GitHubConfig struct {
	// Token is the GitHub personal access token for API authentication
	Token string `yaml:"token" json:"token"`

	// BaseURL is the GitHub API base URL (for GitHub Enterprise)
	BaseURL string `yaml:"base_url" json:"base_url"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	// Host is the server host address
	Host string `yaml:"host" json:"host"`

	// Port is the server port
	Port int `yaml:"port" json:"port"`
}
