package github

import "github.com/google/go-github/v82/github"

// NewClient creates a GitHub API client using the default HTTP client.
func NewClient() *github.Client {
	return github.NewClient(nil)
}
