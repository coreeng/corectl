package testconfig

import "github.com/google/go-github/v90/github"

func NewGitHubClient() *github.Client {
	client, err := github.NewClient(github.WithAuthToken(Cfg.GitHubToken))
	if err != nil {
		panic(err)
	}
	return client
}
