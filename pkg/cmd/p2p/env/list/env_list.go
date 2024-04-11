package list

import (
	"context"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvListOpts struct {
	Repo string

	Streams userio.IOStreams
}

func NewP2PListCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvListOpts{}
	var listEnvironmentsCmd = &cobra.Command{
		Use:   "list",
		Short: "List Environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}

	listEnvironmentsCmd.Flags().StringVarP(
		&opts.Repo,
		"apprepo",
		"a",
		"",
		"Application Repository")

	err := listEnvironmentsCmd.MarkFlagRequired("repo")
	if err != nil {
		return nil, err
	}

	return listEnvironmentsCmd, nil
}

func run(opts *EnvListOpts, cfg *config.Config) error {
	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	opts.Streams.Info("Retrieving Environments")
	environments, _, err := githubClient.Repositories.ListEnvironments(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.Repo,
		&github.EnvironmentListOptions{},
	)
	if err != nil {
		return err
	}
	if *environments.TotalCount > 0 {
		for i := 0; i < *environments.TotalCount; i++ {
			opts.Streams.Info("Environment: ", string(*environments.Environments[i].Name))
		}
	} else {
		opts.Streams.Info("No Existing Environments")
	}
	return nil
}
