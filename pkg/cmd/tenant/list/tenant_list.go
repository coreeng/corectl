package list

import (
	"fmt"

	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/spf13/cobra"
)

type TenantListOpts struct {
	Streams userio.IOStreams
}

func NewTenantListCmd(cfg *config.Config) *cobra.Command {
	opts := TenantListOpts{}
	tenantListCmd := &cobra.Command{
		Use:   "list",
		Short: "List tenants",
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

	return tenantListCmd
}

func run(opts *TenantListOpts, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	tenants, err := tenant.List(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}

	table := corectltnt.NewTable(opts.Streams)
	for _, t := range tenants {
		table.AppendRow(t)
	}
	table.Render()
	return nil
}
