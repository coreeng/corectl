package set

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
)

type ConfigSetOpts struct {
	Path  string
	Value string

	Streams userio.IOStreams
}

func NewConfigSetCmd(cfg *config.Config) *cobra.Command {
	opts := ConfigSetOpts{}
	configSetCmd := &cobra.Command{
		Use:   "set <parameter-name> <parameter-value>",
		Short: "Set configuration parameters",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			opts.Path = args[0]
			opts.Value = args[1]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(&opts, cfg)
		},
	}

	return configSetCmd
}

func run(opts *ConfigSetOpts, cfg *config.Config) error {
	if !cfg.IsPersisted() {
		opts.Streams.Info(
			"No config found\n" +
				"Consider running initializing corectl first:\n" +
				"  corectl config init",
		)
		return nil
	}

	updatedCfg, err := cfg.SetValue(opts.Path, opts.Value)
	if err != nil {
		return err
	}
	if err := updatedCfg.Save(); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}
	return nil
}
