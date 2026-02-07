package github

import "github.com/google/go-github/v82/github"

func NewClient() {
	client := github.NewClient(nil)
}
