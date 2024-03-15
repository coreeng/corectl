package template

import (
	"github.com/coreeng/corectl/pkg/cmd/template/describe"
	"github.com/coreeng/corectl/pkg/cmd/template/list"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewTemplateCmd(cfg *config.Config) *cobra.Command {
	templateCmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"templates"},
		Short:   "Operations with tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	templateCmd.AddCommand(describe.NewTemplateDescribeCmd(cfg))
	templateCmd.AddCommand(list.NewTemplateListCmd(cfg))
	templateCmd.AddCommand(render.NewTemplateRenderCmd(cfg))

	return templateCmd
}
