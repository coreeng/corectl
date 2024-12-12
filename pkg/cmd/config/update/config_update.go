package update

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
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
			cmd.SilenceUsage = true
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(&opts, cfg)
		},
	}

	return configUpdateCmd
}

func run(opts *ConfigUpdateOpts, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{
		cfg.Repositories.CPlatform,
		cfg.Repositories.Templates,
	}

	return config.Update(
		cfg.IsPersisted(),
		cfg.GitHub.Token.Value,
		opts.Streams,
		cfg.Repositories.AllowDirty.Value,
		repoParams,
	)
}
