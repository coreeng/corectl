package testconfig

import "github.com/google/go-github/v59/github"

func NewGitHubClient() *github.Client {
	return github.NewClient(nil).
		WithAuthToken(Cfg.GitHubToken)
}
