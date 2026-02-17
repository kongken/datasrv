package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v82/github"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
)

// App represents the main application
type App struct {
	Config         *conf.Config
	DAO            dao.DAO
	GitHubService  *service.GitHubService
	GitHubClient   *github.Client
}

// NewApp creates and initializes a new application instance
func NewApp(ctx context.Context, cfg *conf.Config) (*App, error) {
	app := &App{
		Config: cfg,
	}

	// Initialize DAO based on driver type
	if err := app.initDAO(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize DAO: %w", err)
	}

	// Initialize GitHub client
	app.initGitHubClient()

	// Initialize GitHub service
	app.GitHubService = service.NewGitHubService(app.GitHubClient, app.DAO)

	log.Println("Application initialized successfully")
	return app, nil
}

// initDAO initializes the data access layer
func (a *App) initDAO(ctx context.Context) error {
	switch a.Config.Database.Driver {
	case "postgres", "postgresql":
		pgDAO, err := dao.NewPostgresDAO(a.Config.Database.DSN)
		if err != nil {
			return fmt.Errorf("failed to create PostgreSQL DAO: %w", err)
		}

		// Run database migrations
		if err := pgDAO.Migrate(ctx); err != nil {
			return fmt.Errorf("failed to run database migrations: %w", err)
		}

		a.DAO = pgDAO
		log.Println("PostgreSQL DAO initialized successfully")
		return nil

	case "mongodb", "mongo":
		// TODO: Implement MongoDB DAO
		return fmt.Errorf("MongoDB driver not yet implemented")

	default:
		return fmt.Errorf("unsupported database driver: %s", a.Config.Database.Driver)
	}
}

// initGitHubClient initializes the GitHub client
func (a *App) initGitHubClient() {
	if a.Config.GitHub.Token != "" {
		a.GitHubClient = github.NewClient(nil).WithAuthToken(a.Config.GitHub.Token)
		log.Println("GitHub client initialized with authentication token")
	} else {
		a.GitHubClient = github.NewClient(nil)
		log.Println("GitHub client initialized without authentication (rate limit: 60 req/hour)")
	}

	// Set custom base URL if provided (for GitHub Enterprise)
	if a.Config.GitHub.BaseURL != "" {
		var err error
		a.GitHubClient, err = a.GitHubClient.WithEnterpriseURLs(a.Config.GitHub.BaseURL, a.Config.GitHub.BaseURL)
		if err != nil {
			log.Printf("Warning: failed to set custom GitHub base URL: %v", err)
		} else {
			log.Printf("GitHub client configured with custom base URL: %s", a.Config.GitHub.BaseURL)
		}
	}
}

// Close closes all application resources
func (a *App) Close() error {
	log.Println("Closing application resources...")
	
	if a.DAO != nil {
		if err := a.DAO.Close(); err != nil {
			return fmt.Errorf("failed to close DAO: %w", err)
		}
		log.Println("DAO closed successfully")
	}
	
	return nil
}

// NewAppWithDefaultConfig creates a new application with default configuration
func NewAppWithDefaultConfig(ctx context.Context) (*App, error) {
	cfg := conf.NewDefaultConfig()
	return NewApp(ctx, cfg)
}

// NewAppFromEnv creates a new application by loading configuration from environment variables
func NewAppFromEnv(ctx context.Context) (*App, error) {
	cfg, err := conf.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return NewApp(ctx, cfg)
}
