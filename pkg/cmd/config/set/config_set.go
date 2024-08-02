package set

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
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
			opts.Path = args[0]
			opts.Value = args[1]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
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

	updatedCfg, err := updateConfig(opts, cfg)
	if err != nil {
		return err
	}
	if err := updatedCfg.Save(); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}
	return nil
}

func updateConfig(opts *ConfigSetOpts, cfg *config.Config) (*config.Config, error) {
	var cfgYaml yaml.Node
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	if err := yaml.Unmarshal(cfgBytes, &cfgYaml); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	path, err := yamlpath.NewPath(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find property to modify: %w", err)
	}
	nodeToModify, err := path.Find(&cfgYaml)
	if len(nodeToModify) != 1 {
		return nil, fmt.Errorf("path represents multiple nodes: %w", err)
	}
	if nodeToModify[0].Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("path does not represent a scalar node: %w", err)
	}
	nodeToModify[0].Value = opts.Value
	cfgBytes, err = yaml.Marshal(&cfgYaml)
	if err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	cfgNew := config.NewConfig()
	if err := yaml.Unmarshal(cfgBytes, &cfgNew); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}
	return cfgNew, nil
}
