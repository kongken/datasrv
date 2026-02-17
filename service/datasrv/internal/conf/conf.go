package conf

import (
	"fmt"
	"os"
)

// Config holds all configuration for the datasrv service
type Config struct {
	// Database configuration
	Database DatabaseConfig `json:"database"`
	
	// GitHub configuration
	GitHub GitHubConfig `json:"github"`
	
	// Server configuration
	Server ServerConfig `json:"server"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Driver specifies the database driver (postgres, mongodb, etc.)
	Driver string `json:"driver"`
	
	// DSN is the data source name for the database connection
	DSN string `json:"dsn"`
	
	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int `json:"max_open_conns"`
	
	// MaxIdleConns is the maximum number of connections in the idle connection pool
	MaxIdleConns int `json:"max_idle_conns"`
}

// GitHubConfig holds GitHub API configuration
type GitHubConfig struct {
	// Token is the GitHub personal access token for API authentication
	Token string `json:"token"`
	
	// BaseURL is the GitHub API base URL (for GitHub Enterprise)
	BaseURL string `json:"base_url"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	// Host is the server host address
	Host string `json:"host"`
	
	// Port is the server port
	Port int `json:"port"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Database: DatabaseConfig{
			Driver:       getEnvOrDefault("DB_DRIVER", "postgres"),
			DSN:          getEnvOrDefault("DATABASE_DSN", ""),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
		},
		GitHub: GitHubConfig{
			Token:   getEnvOrDefault("GITHUB_TOKEN", ""),
			BaseURL: getEnvOrDefault("GITHUB_BASE_URL", ""),
		},
		Server: ServerConfig{
			Host: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
	}

	// Validate required configuration
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("DATABASE_DSN is required")
	}

	return cfg, nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as int or returns default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// NewDefaultConfig creates a new default configuration
func NewDefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Driver:       "postgres",
			DSN:          "host=localhost port=5432 user=postgres password=postgres dbname=github_issues sslmode=disable",
			MaxOpenConns: 25,
			MaxIdleConns: 10,
		},
		GitHub: GitHubConfig{
			Token:   "",
			BaseURL: "",
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}
}
