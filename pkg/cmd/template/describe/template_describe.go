package describe

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

type TemplateDescribeOpts struct {
	IgnoreChecks bool
}

func NewTemplateDescribeCmd(cfg *config.Config) *cobra.Command {
	opts := TemplateDescribeOpts{}
	templateDescribeCmd := &cobra.Command{
		Use:   "describe <template-name>",
		Short: "Show detailed information about passed template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			if !opts.IgnoreChecks {
				if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
					return err
				}
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

	templateDescribeCmd.Flags().BoolVar(
		&opts.IgnoreChecks,
		"ignore-checks",
		false,
		"Ignore checks for uncommitted changes and branch status",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateDescribeCmd.Flags(),
	)

	return templateDescribeCmd
}
