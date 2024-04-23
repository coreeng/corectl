package p2p

import (
	"context"
	"encoding/json"

	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/google/go-github/v59/github"
)

type StageVarName string

const (
	FastFeedbackVar StageVarName = "FAST_FEEDBACK"
	ExtendedTestVar StageVarName = "EXTENDED_TEST"
	ProdVar         StageVarName = "PROD"
)

type StageRepositoryConfig struct {
	Include []StageTarget `json:"include,omitempty"`
}

type StageTarget struct {
	DeployEnv string `json:"deploy_env"`
}

func NewStageRepositoryConfig(targetEnvs []environment.Environment) StageRepositoryConfig {
	var targets []StageTarget
	for _, env := range targetEnvs {
		targets = append(targets, StageTarget{DeployEnv: string(env.Environment)})
	}
	return StageRepositoryConfig{Include: targets}
}

func CreateStageRepositoryConfig(
	githubClient *github.Client,
	repoFullname *git.RepositoryFullname,
	varName StageVarName,
	config StageRepositoryConfig,
) error {	
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	response, err := githubClient.Actions.CreateRepoVariable(
		context.Background(),
		repoFullname.Organization,
		repoFullname.Name,
		&github.ActionsVariable{
			Name:  string(varName),
			Value: string(configBytes),
		},
	)
	if err !=nil {
		if response.StatusCode == 409 {
			_, err = githubClient.Actions.UpdateRepoVariable(
				context.Background(),
				repoFullname.Organization,
				repoFullname.Name,
				&github.ActionsVariable{
					Name: string(varName),
					Value: string(configBytes),
				},
			)
			if err != nil {
				return err
			}
		}
	}
	return err
}
