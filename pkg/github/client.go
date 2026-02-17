package github

import "github.com/google/go-github/v82/github"

func NewClient() *github.Client {
	client := github.NewClient(nil)
	return client
}
