package env

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
	"strings"
)

type EnvOpenResourceOpt struct {
	Environment string
	Resource    string

	Streams userio.IOStreams
}

func openResource(cfg *config.Config) *cobra.Command {
	var opts EnvOpenResourceOpt
	cmd := cobra.Command{
		Use:   "open <environment> <resource>",
		Short: "Open a resource of environment. Available resources are: " + strings.Join(corectlenv.ResourceStringList(), ", "),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Environment = args[0]
			opts.Resource = args[1]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(cfg, &opts)
		},
	}
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		cmd.Flags(),
	)
	return &cmd
}

func run(cfg *config.Config, opts *EnvOpenResourceOpt) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}
	env, err := environment.FindByName(
		environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value),
		opts.Environment,
	)
	if err != nil {
		return err
	}
	if env == nil {
		return fmt.Errorf("environment %s not found", opts.Environment)
	}
	if err := corectlenv.OpenResource(corectlenv.ResourceType(opts.Resource), env); err != nil {
		return fmt.Errorf("couldn't open %s: %w", opts.Resource, err)
	}
	return nil
}
