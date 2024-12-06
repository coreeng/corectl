package tree

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/spf13/cobra"
)

type TenantTreeOpts struct {
	From string // Name of the tenant to start the tenant tree from; use "" to start from root

	Streams userio.IOStreams
}

func NewTenantTreeCmd(cfg *config.Config) *cobra.Command {
	opts := TenantTreeOpts{}
	tenantTreeCmd := &cobra.Command{
		Use:   "tree",
		Short: "List tenants as a tree",
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

	config.RegisterStringParameterAsFlag(&cfg.Repositories.CPlatform, tenantTreeCmd.Flags())
	config.RegisterBoolParameterAsFlag(&cfg.Repositories.AllowDirty, tenantTreeCmd.Flags())

	tenantTreeCmd.Flags().StringVarP(
		&opts.From,
		"from",
		"f",
		"",
		"The tenant to start the tree view from.",
	)

	return tenantTreeCmd
}

func run(opts *TenantTreeOpts, cfg *config.Config) error {
	if !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, false); err != nil {
			return err
		}
	}

	rootNodes, err := corectltnt.GetTenantTrees(cfg, opts.From)
	if err != nil {
		return fmt.Errorf("failed to build tenant trees: %w", err)
	}

	for _, rootNode := range rootNodes {
		output := corectltnt.RenderTenantTree(rootNode)
		for _, line := range output {
			// XXX FABRICE use opts.Streams instead of fmt.println()
			fmt.Println(line)
		}
	}

	return nil
}
