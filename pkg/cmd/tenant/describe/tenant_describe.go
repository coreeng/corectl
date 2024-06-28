package list

import (
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/spf13/cobra"
)

type TenantDescribeOpts struct {
	TenantName string
	Streams    userio.IOStreams
}

func NewTenantDescribeCmd(cfg *config.Config) *cobra.Command {
	var opts = TenantDescribeOpts{}
	var tenantListCmd = &cobra.Command{
		Use:   "describe <tenant-name>",
		Short: "Describe tenant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TenantName = args[0]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		tenantListCmd.Flags(),
	)
	return tenantListCmd
}

func run(opts *TenantDescribeOpts, cfg *config.Config) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}
	tenants, err := tenant.List(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}
	table := corectltnt.NewTable(opts.Streams)
	for _, t := range tenants {
		table.Append(t)
	}
	table.Render()
	return nil
}
