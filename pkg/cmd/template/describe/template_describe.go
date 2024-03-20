package describe

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

func NewTemplateDescribeCmd(cfg *config.Config) *cobra.Command {
	templateDescribeCmd := &cobra.Command{
		Use:   "describe <template-name>",
		Short: "Show detailed information about passed template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
				return err
			}
			t, err := template.FindByName(cfg.Repositories.Templates.Value, templateName)
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("%s: unknown template", templateName)
			}

			tBytes, err := yaml.Marshal(&t)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(tBytes)); err != nil {
				return err
			}

			return nil
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateDescribeCmd.Flags(),
	)

	return templateDescribeCmd
}
