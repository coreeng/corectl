package tree

import (
	"fmt"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
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
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	tenants, err := coretnt.List(configpath.GetCorectlCPlatformDir("tenants"))
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
		if _, err := fmt.Fprintln(opts.Streams.GetOutput(), line); err != nil {
			return fmt.Errorf("failed to write tenant tree line: %w", err)
		}
	}
	return nil
}
