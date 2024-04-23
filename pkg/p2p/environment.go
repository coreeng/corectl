package p2p

import (
	"context"
	"errors"
	"slices"

	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/google/go-github/v59/github"
)

func CreateUpdateEnvironmentForRepository(
	githubClient *github.Client,
	repoId *git.GithubRepoFullId,
	env *environment.Environment,
) error {
	if env.Platform.Vendor == "gcp" {
		repoEnv, _, err := githubClient.Repositories.CreateUpdateEnvironment(
			context.Background(),
			repoId.Fullname.Organization,
			repoId.Fullname.Name,
			string(env.Environment),
			&github.CreateUpdateEnvironment{},
		)
		if err != nil {
			return err
		}
		defaultDomain := env.GetDefaultIngressDomain()
		if defaultDomain == nil {
			return errors.New("default ingress domain is not found")
		}
		varsToCreate := []github.ActionsVariable{
			{
				Name:  "BASE_DOMAIN",
				Value: defaultDomain.Domain,
			},
			{
				Name:  "INTERNAL_SERVICES_DOMAIN",
				Value: env.InternalServices.Domain,
			},
			{
				Name:  "DPLATFORM",
				Value: string(env.Environment),
			},
			{
				Name:  "PROJECT_ID",
				Value: env.Platform.ProjectId,
			},
			{
				Name:  "PROJECT_NUMBER",
				Value: env.Platform.ProjectNumber,
			},
		}
		for i := range varsToCreate {
			response, err := githubClient.Actions.CreateEnvVariable(
				context.Background(),
				repoId.Id,
				*repoEnv.Name,
				&varsToCreate[i],
			)
			if err != nil {
				if response.StatusCode == 409 {
					_, err := githubClient.Actions.UpdateEnvVariable(
						context.Background(),
						int(repoId.Id),
						*repoEnv.Name,
						&varsToCreate[i],
					)
					if err != nil {
						return err
					}
				} else if response.StatusCode == 200 {
					return nil
				} else {
					return err
				}
			}
		}
	}
	return nil
}

func DeleteEnvironment (
	op           *InitializeOp,
	githubClient *github.Client,
) error {
	//Remove any existing environments as per #295
	var err error
	var response *github.Response
	for _, env := range slices.Concat(op.FastFeedbackEnvs, op.ExtendedTestEnvs, op.ProdEnvs) {
		if env.Platform.Vendor == "gcp" {

			response, err = githubClient.Repositories.DeleteEnvironment(
				context.Background(),
				op.RepositoryId.Fullname.Organization,
				op.RepositoryId.Fullname.Name,
				string(env.Environment),
			)
			if err != nil {
				if response.StatusCode == 404 {
					continue
				} else {
					return err
				}
				
			}
		}
	}
	return err
}
