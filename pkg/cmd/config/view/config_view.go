package view

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ConfigViewOpts struct {
	Streams userio.IOStreams
	Raw     bool
}

func NewConfigViewCmd(cfg *config.Config) *cobra.Command {
	var opts = ConfigViewOpts{}
	var configViewCmd = &cobra.Command{
		Use:   "view",
		Short: "Displays local corectl configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(&opts, cfg)
		},
	}

	configViewCmd.Flags().BoolVar(&opts.Raw, "raw", false, "Print raw configuration (with sensitive data)")

	return configViewCmd
}

func run(opts *ConfigViewOpts, cfg *config.Config) error {
	if !cfg.IsPersisted() {
		opts.Streams.Info(
			"No config found\n" +
				"Consider running initializing corectl first:\n" +
				"  corectl config init",
		)
		return nil
	}

	cfgCopy := *cfg

	if !opts.Raw {
		cfgCopy.GitHub.Token.Value = "REDACTED"
	}

	encoder := yaml.NewEncoder(opts.Streams.GetOutput())
	encoder.SetIndent(2)
	if err := encoder.Encode(cfgCopy); err != nil {
		return fmt.Errorf("failed to print corectl configuration: %w", err)
	}
	return nil
}
