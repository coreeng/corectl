package git

import (
	"errors"
	"github.com/google/go-github/v59/github"
	"regexp"
	"strings"
)

var gitRepoRegexp = regexp.MustCompile(`^.*[:/]([\w-.]+)/([\w-.]+)(\.git)?$`)

type RepositoryFullname struct {
	Organization string
	Name         string
}

type GithubRepoFullId struct {
	Id       int
	Fullname RepositoryFullname
}

func DeriveRepositoryFullname(localRepo *LocalRepository) (RepositoryFullname, error) {
	config, err := localRepo.repo.Config()
	if err != nil {
		return RepositoryFullname{}, err
	}
	remoteConfig, ok := config.Remotes["origin"]
	if !ok {
		return RepositoryFullname{}, errors.New("origin remote is missing")
	}

	repoUrl := remoteConfig.URLs[0]
	return DeriveRepositoryFullnameFromUrl(repoUrl)
}

func DeriveRepositoryFullnameFromUrl(githubRepoUrl string) (RepositoryFullname, error) {
	matches := gitRepoRegexp.FindStringSubmatch(githubRepoUrl)
	if len(matches) != 4 {
		return RepositoryFullname{}, errors.New("unexpected url")
	}
	orgName := matches[1]
	repoName := strings.TrimSuffix(matches[2], ".git")
	return RepositoryFullname{
		Organization: orgName,
		Name:         repoName,
	}, nil
}

func (n RepositoryFullname) AsString() string {
	return n.Organization + "/" + n.Name
}

func (n RepositoryFullname) HttpUrl() string {
	return "https://github.com/" + n.Organization + "/" + n.Name
}

func (n RepositoryFullname) ActionsHttpUrl() string {
	return "https://github.com/" + n.Organization + "/" + n.Name + "/actions"
}

func NewGithubRepoFullId(repository *github.Repository) GithubRepoFullId {
	return GithubRepoFullId{
		Id: int(*repository.ID),
		Fullname: RepositoryFullname{
			Organization: *repository.Owner.Login,
			Name:         *repository.Name,
		},
	}
}
