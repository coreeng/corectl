package git

import (
	"fmt"
	"github.com/google/go-github/v59/github"
	"regexp"
	"strings"
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
