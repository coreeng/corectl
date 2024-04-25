package p2p

import (
	"context"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/coreeng/developer-platform/pkg/p2p"
	"github.com/google/go-github/v59/github"
	"slices"
)

func CleanUpRepoEnvs(
	repoId p2p.GitHubRepoFullId,
	fastFeedbackEnvs []environment.Environment,
	extendedTestEnvs []environment.Environment,
	prodEnvs []environment.Environment,
	githubClient *github.Client,
) error {
	//Remove any existing environments as per #295
	var err error
	var response *github.Response
	for _, env := range slices.Concat(fastFeedbackEnvs, extendedTestEnvs, prodEnvs) {
		if env.Platform.Type() != environment.GCPVendorType {
			continue
		}
		response, err = githubClient.Repositories.DeleteEnvironment(
			context.Background(),
			repoId.Organization(),
			repoId.Organization(),
			env.Environment,
		)
		if err != nil {
			if response.StatusCode == 404 {
				continue
			} else {
				return err
			}
		}
	}
	return err
}
