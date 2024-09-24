package sync

import (
	"context"
	"fmt"
	"slices"

	"github.com/coreeng/developer-platform/pkg/environment"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	corep2p "github.com/coreeng/developer-platform/pkg/p2p"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvCreateOpts struct {
	AppRepo string
	Tenant  string
	Clean   bool
	Streams userio.IOStreams
}

func NewP2PSyncCmd(cfg *config.Config) (*cobra.Command, error) {
	var opts = EnvCreateOpts{}
	var syncEnvironmentsCmd = &cobra.Command{
		Use:   "sync <app repository> <tenant>",
		Short: "Synchronise Environments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.AppRepo = args[0]
			opts.Tenant = args[1]

			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}
	syncEnvironmentsCmd.Flags().BoolVar(
		&opts.Clean,
		"clean",
		false,
		"Clean existing environments",
	)
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		syncEnvironmentsCmd.Flags())

	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Organization,
		syncEnvironmentsCmd.Flags())

	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Token,
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

	spinnerHandler := opts.Streams.Wizard("Configuring environments", "Configured environments")
	defer spinnerHandler.Done()

	t, err := tenant.FindByName(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value), opts.Tenant)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tenant not found: %s", opts.Tenant)
	}
	environments, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}
	fastFeedbackEnvs := filterEnvsByNames(cfg.P2P.FastFeedback.DefaultEnvs.Value, environments)
	extendedTestEnvs := filterEnvsByNames(cfg.P2P.ExtendedTest.DefaultEnvs.Value, environments)
	prodEnvs := filterEnvsByNames(cfg.P2P.Prod.DefaultEnvs.Value, environments)
	repoId := git.NewGithubRepoFullId(repository)
	if opts.Clean {
		err = p2p.CleanUpRepoEnvs(
			repoId,
			fastFeedbackEnvs,
			extendedTestEnvs,
			prodEnvs,
			githubClient,
		)
		if err != nil {
			return err
		}
	}
	op := corep2p.SynchronizeOp{
		RepositoryId:     &repoId,
		Tenant:           t,
		FastFeedbackEnvs: fastFeedbackEnvs,
		ExtendedTestEnvs: extendedTestEnvs,
		ProdEnvs:         prodEnvs,
	}
	if err = corep2p.SynchronizeRepository(
		&op,
		githubClient,
	); err != nil {
		return err
	}
	return nil
}

func filterEnvsByNames(names []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(names, env.Environment) {
			result = append(result, env)
		}
	}
	return result
}
