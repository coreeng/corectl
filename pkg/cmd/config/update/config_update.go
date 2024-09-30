package update

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/spf13/cobra"
)

type ConfigUpdateOpts struct {
	Streams userio.IOStreams
}

func NewConfigUpdateCmd(cfg *config.Config) *cobra.Command {
	opts := ConfigUpdateOpts{}
	configUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Pull updates from remote repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}

	return configUpdateCmd
}

func run(opts *ConfigUpdateOpts, cfg *config.Config) error {
	if !cfg.IsPersisted() {
		opts.Streams.Info(
			"No config found\n" +
				"Consider running initializing corectl first:\n" +
				"  corectl config init",
		)
		return nil
	}

	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)

	err := updateRepository(&cfg.Repositories.CPlatform, gitAuth, opts.Streams)
	if err != nil {
		return err
	}

	err = updateRepository(&cfg.Repositories.Templates, gitAuth, opts.Streams)
	if err != nil {
		return err
	}
	return nil
}

func updateRepository(repoParam *config.Parameter[string], gitAuth git.AuthMethod, streams userio.IOStreams) error {
	isUpdated, err := func() (bool, error) {
		streams.Wizard(
			fmt.Sprintf("Updating %s", repoParam.Name()),
			fmt.Sprintf("Updated %s", repoParam.Name()),
		)
		defer streams.CurrentHandler.Done()
		repo, err := config.ResetConfigRepositoryState(repoParam, false)
		if err != nil {
			return false, err
		}
		pullResult, err := repo.Pull(gitAuth)
		if err != nil {
			return false, fmt.Errorf("couldn't pull changes for %s: %v", repoParam.Name(), err)
		}
		return pullResult.IsUpdated, nil
	}()
	if err != nil {
		return err
	}

	var msg string
	if isUpdated {
		msg = fmt.Sprintf("%s is updated succesfully!", repoParam.Name())
	} else {
		msg = fmt.Sprintf("%s is up to date!", repoParam.Name())
	}
	streams.Info(msg)
	return nil
}
