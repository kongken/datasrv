package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kongken/datasrv/service/datasrv/internal/dao"
	"github.com/kongken/datasrv/service/datasrv/internal/service"
)

func main() {
	ctx := context.Background()

	// Get configuration from environment variables
	dbDSN := os.Getenv("DATABASE_DSN")
	if dbDSN == "" {
		dbDSN = "host=localhost port=5432 user=postgres password=postgres dbname=github_issues sslmode=disable"
		log.Printf("DATABASE_DSN not set, using default: %s", dbDSN)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("GITHUB_TOKEN not set, using unauthenticated requests (rate limit: 60 req/hour)")
	}

	// Create the service with configuration
	cfg := &service.Config{
		DatabaseDSN: dbDSN,
		GitHubToken: githubToken,
	}

	svc, err := service.NewGitHubServiceWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create GitHub service: %v", err)
	}

	// Example 1: Fetch and store all open issues from a repository
	owner := "golang"
	repo := "go"

	log.Printf("Fetching all open issues from %s/%s...", owner, repo)
	if err := svc.FetchAndStoreAllIssues(ctx, owner, repo, "open"); err != nil {
		log.Fatalf("Failed to fetch and store issues: %v", err)
	}

	log.Println("Successfully fetched and stored all open issues")

	// Example 2: List issues from the database
	log.Println("\nListing first 10 issues from database:")
	issues, err := svc.ListIssues(ctx, &dao.ListOptions{
		Limit:  10,
		Offset: 0,
		State:  "open",
	})
	if err != nil {
		log.Fatalf("Failed to list issues: %v", err)
	}

	for _, issue := range issues {
		fmt.Printf("Issue #%d: %s (State: %s)\n", issue.Number, issue.Title, issue.State)
	}

	// Example 3: Get a specific issue by number
	issueNumber := int32(1)
	issue, err := svc.GetIssueByNumber(ctx, issueNumber)
	if err != nil {
		log.Printf("Failed to get issue #%d: %v", issueNumber, err)
	} else {
		fmt.Printf("\nIssue #%d details:\n", issueNumber)
		fmt.Printf("  Title: %s\n", issue.Title)
		fmt.Printf("  State: %s\n", issue.State)
		fmt.Printf("  Comments: %d\n", issue.Comments)
		fmt.Printf("  Created: %s\n", issue.CreatedAt)
	}

	log.Println("\nExample completed successfully")
}
