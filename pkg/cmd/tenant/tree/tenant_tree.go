package tree

import (
	"fmt"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
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

	tenants, err := coretnt.List(coretnt.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}

	from := coretnt.RootName
	if opts.From != "" {
		from = opts.From
	}
	tenants = append(tenants, coretnt.Tenant{Name: coretnt.RootName})
	rootNode, err := corectltnt.GetTenantTree(tenants, from)
	if err != nil {
		return fmt.Errorf("failed to build tenant tree: %w", err)
	}

	_, lines := corectltnt.RenderTenantTree(rootNode)
	for _, line := range lines {
		fmt.Fprintln(opts.Streams.GetOutput(), line)
	}
	return nil
}
