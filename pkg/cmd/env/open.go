package env

import (
	"fmt"
	"strings"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
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
				cmd.OutOrStderr(),
			)
			return run(cfg, &opts)
		},
	}
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		cmd.Flags(),
	)
	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		cmd.Flags(),
	)
	return &cmd
}

func run(cfg *config.Config, opts *EnvOpenResourceOpt) error {
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.IsPersisted(), cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
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

	resourceType := corectlenv.ResourceType(opts.Resource)
	if url, err := corectlenv.OpenResource(resourceType, env); err != nil {
		return fmt.Errorf("couldn't open %s: %w", opts.Resource, err)
	} else {
		wizard := opts.Streams.Wizard(
			fmt.Sprintf("Opening %s for env %s: %s", resourceType, env.Environment, url),
			fmt.Sprintf("Opened %s for env %s: %s", resourceType, env.Environment, url),
		)
		defer wizard.Done()
		return browser.OpenURL(url)
	}
}
