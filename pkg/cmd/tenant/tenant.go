package tenant

import (
	"github.com/coreeng/corectl/pkg/cmd/tenant/create"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewTenantCmd(cfg *config.Config) *cobra.Command {
	tenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Operations with tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	tenantCmd.AddCommand(create.NewTenantCreateCmd(cfg))

	return tenantCmd
}
