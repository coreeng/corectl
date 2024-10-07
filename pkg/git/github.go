package git

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

type CoreCtlAsset struct {
	Version   string
	Url       string
	Changelog string
}

func GetLatestCorectlAsset(release *github.RepositoryRelease) (*CoreCtlAsset, error) {
	dummyAsset := CoreCtlAsset{}
	if release.Assets == nil {
		return &dummyAsset, errors.New("no assets found for the latest release")
	}

	architecture := runtime.GOARCH

	// Required due to the goreleaser config
	if architecture == "amd64" {
		architecture = "x86_64"
	}
	targetAssetName := fmt.Sprintf("corectl_%s_%s.tar.gz", runtime.GOOS, architecture)
	for _, asset := range release.Assets {
		assetName := strings.ToLower(asset.GetName())
		if assetName == targetAssetName {
			log.Debug().Str("asset", assetName).Msg("github: found release asset with matching architecture & os")
			return &CoreCtlAsset{
				Url:       *asset.BrowserDownloadURL,
				Version:   *release.TagName,
				Changelog: *release.Body,
			}, nil
		}
	}

	return &dummyAsset, errors.New("no asset found for the current architecture and OS")

}

func GetLatestCorectlRelease(client *github.Client) (*github.RepositoryRelease, error) {
	dummyRelease := github.RepositoryRelease{}
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "coreeng", "corectl")
	if err != nil {
		return &dummyRelease, err
	}
	return release, nil
}

func DownloadCorectlAsset(asset *CoreCtlAsset) (io.ReadCloser, error) {
	resp, err := http.Get(asset.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to download corectl release: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download corectl release: status code %v", resp.StatusCode)
	}

	return resp.Body, err
}

func DecompressCorectlAssetInMemory(tarData io.ReadCloser) (*tar.Reader, error) {
	gzr, err := gzip.NewReader(tarData)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()
	tarReader := tar.NewReader(gzr)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar archive: %v", err)
		}

		if filepath.Base(header.Name) == "corectl" && header.Typeflag == tar.TypeReg {
			return tarReader, nil
		}
	}
	return nil, fmt.Errorf("corectl binary not found in the release")
}

func WriteCorectlAssetToPath(tarReader *tar.Reader, targetPath string) (bool, error) {
	binaryName := "corectl"
	outFile, err := os.Create(targetPath)
	if err != nil {
		return false, fmt.Errorf("failed to create %s binary in %s: %v", binaryName, targetPath, err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, fmt.Errorf("failed to copy %s binary: %v", binaryName, err)
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return false, fmt.Errorf("failed to set executable permissions on %s binary: %v", binaryName, err)
	}

	log.Debug().Msgf("%s has been installed to %s", binaryName, targetPath)
	return true, nil
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
		Bool("dry_run", dryRun).
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
		id := github.Int64(1234)
		return &github.PullRequest{
			ID:      id,
			Title:   pr_title,
			Base:    &github.PullRequestBranch{Label: branch},
			Head:    &github.PullRequestBranch{Label: head},
			Body:    pr_body,
			HTMLURL: github.String(fmt.Sprintf("https://github.com/%s/%s/pull/%d", organization, repoName, *id)),
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
