package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
)

var gitRepoRegexp = regexp.MustCompile(`^.*[:/]([\w-.]+)/([\w-.]+)(\.git)?$`)

type RepositoryFullname struct {
	organization string
	name         string
}

type GithubRepoFullId struct {
	id int
	RepositoryFullname
}

func DeriveRepositoryFullname(localRepo *LocalRepository) (RepositoryFullname, error) {
	config, err := localRepo.repo.Config()
	if err != nil {
		return RepositoryFullname{}, err
	}
	remoteConfig, ok := config.Remotes["origin"]
	if !ok {
		return RepositoryFullname{}, fmt.Errorf("origin remote is missing, repo %q", localRepo.Path())
	}

	repoUrl := remoteConfig.URLs[0]
	return DeriveRepositoryFullnameFromUrl(repoUrl)
}

func DeriveRepositoryFullnameFromUrl(githubRepoUrl string) (RepositoryFullname, error) {
	matches := gitRepoRegexp.FindStringSubmatch(githubRepoUrl)
	if len(matches) != 4 {
		return RepositoryFullname{}, fmt.Errorf("unexpected url %q", githubRepoUrl)
	}
	orgName := matches[1]
	repoName := strings.TrimSuffix(matches[2], ".git")
	return RepositoryFullname{
		organization: orgName,
		name:         repoName,
	}, nil
}

func CreateGitHubPR(client *github.Client, title string, body string, branchName string, repoName string, organization string, dryRun bool) (*github.PullRequest, error) {
	pr_title := github.String(title)
	pr_body := github.String(body)
	branch := github.String(MainBranch)
	head := github.String(branchName)
	log.Info().
		Str("name", repoName).
		Str("branch_name", *branch).
		Str("org", organization).
		Str("repo", fmt.Sprintf("https://github.com/%s/%s", organization, repoName)).
		Str("title", *pr_title).
		Str("body", *pr_body).
		Msg("github: creating PR")
	if !dryRun {
		pullRequest, _, err := client.PullRequests.Create(
			context.Background(),
			organization,
			repoName,
			&github.NewPullRequest{
				Base:  branch,
				Head:  head,
				Title: pr_title,
				Body:  pr_body,
			})
		return pullRequest, err
	} else {
		return &github.PullRequest{
			ID:    github.Int64(1234),
			Title: pr_title,
			Base:  &github.PullRequestBranch{Label: branch},
			Head:  &github.PullRequestBranch{Label: head},
			Body:  pr_body,
		}, nil
	}
}

func (n RepositoryFullname) String() string {
	return n.organization + "/" + n.name
}

func (n RepositoryFullname) HttpUrl() string {
	return "https://github.com/" + n.organization + "/" + n.name
}

func (n RepositoryFullname) ActionsHttpUrl() string {
	return "https://github.com/" + n.organization + "/" + n.name + "/actions"
}

func (n RepositoryFullname) Organization() string {
	return n.organization
}

func (n RepositoryFullname) Name() string {
	return n.name
}

func NewGithubRepoFullId(repository *github.Repository) GithubRepoFullId {
	return GithubRepoFullId{
		id: int(*repository.ID),
		RepositoryFullname: RepositoryFullname{
			organization: *repository.Owner.Login,
			name:         *repository.Name,
		},
	}
}

func (i GithubRepoFullId) Id() int {
	return i.id
}
