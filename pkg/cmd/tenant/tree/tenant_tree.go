package tree

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/spf13/cobra"
)

type TenantTreeOpts struct {
	Streams userio.IOStreams
}

func NewTenantTreeCmd(cfg *config.Config) *cobra.Command {
	opts := TenantTreeOpts{}
	tenantListCmd := &cobra.Command{
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

	config.RegisterStringParameterAsFlag(&cfg.Repositories.CPlatform, tenantListCmd.Flags())
	config.RegisterBoolParameterAsFlag(&cfg.Repositories.AllowDirty, tenantListCmd.Flags())
	// XXX FABRICE: add a flag to start the tree somewhere else than root

	return tenantListCmd
}

func run(opts *TenantTreeOpts, cfg *config.Config) error {
	if !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, false); err != nil {
			return err
		}
	}

	node, err := corectltnt.GetTenantTree(cfg, "")
	if err != nil {
		return fmt.Errorf("failed to build tenants tree: %w", err)
	}

	output := corectltnt.RenderTenantTree(node)
	for _, line := range output {
		fmt.Println(line)
	}

	return nil
}
