package sync

import (
	"context"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvCreateOpts struct {
	AppRepo string
	Name    string
	Streams userio.IOStreams
}

func NewP2PSyncCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvCreateOpts{}
	var syncEnvironmentsCmd = &cobra.Command{
		Use:   "sync <environment> ",
		Short: "Synchronise Environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}

	syncEnvironmentsCmd.Flags().StringVarP(
		&opts.AppRepo,
		"apprepo",
		"a",
		"",
		"Application Repository")
	err := syncEnvironmentsCmd.MarkFlagRequired("apprepo")
	if err != nil {
		return nil, err
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		syncEnvironmentsCmd.Flags())

	return syncEnvironmentsCmd, nil
}

func run(opts *EnvCreateOpts, cfg *config.Config) error {

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)

	repository, _, err := githubClient.Repositories.Get(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.AppRepo)
	if err != nil {
		return err
	}
	repoId := git.NewGithubRepoFullId(repository)
	env, err := environment.GetEnvironmentByName(cfg.Repositories.CPlatform.Value, opts.Name)
	if err != nil {
		opts.Streams.Info(err.Error())
		return err
	}
	opts.Streams.Info("Updating " + opts.Name + " environment for " + repoId.Fullname.Name)
	err = p2p.CreateUpdateEnvironmentForRepository(
		githubClient,
		&repoId,
		&env,
	)
	if err != nil {
		return err
	}

	return nil
}
