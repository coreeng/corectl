package describe

import (
	"fmt"

	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type TenantDescribeOpts struct {
	TenantName string
	Streams    userio.IOStreams
}

func NewTenantDescribeCmd(cfg *config.Config) *cobra.Command {
	var opts = TenantDescribeOpts{}
	var tenantDescribeCmd = &cobra.Command{
		Use:   "describe <tenant-name>",
		Short: "Describe tenant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			opts.TenantName = args[0]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(&opts, cfg)
		},
	}
	config.RegisterStringParameterAsFlag(&cfg.Repositories.CPlatform, tenantDescribeCmd.Flags())
	config.RegisterBoolParameterAsFlag(&cfg.Repositories.AllowDirty, tenantDescribeCmd.Flags())
	return tenantDescribeCmd
}

func run(opts *TenantDescribeOpts, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	t, err := tenant.FindByName(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value), opts.TenantName)
	if err != nil {
		return fmt.Errorf("failed to find the tenant: %w", err)
	}
	if t == nil {
		return fmt.Errorf("tenant is not found: %s", opts.TenantName)
	}

	encoder := yaml.NewEncoder(opts.Streams.GetOutput())
	encoder.SetIndent(2)
	if err := encoder.Encode(t); err != nil {
		return fmt.Errorf("failed to print tenant: %w", err)
	}
	return nil
}
