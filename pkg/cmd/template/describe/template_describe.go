package describe

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
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
			cmd.SilenceUsage = true
			templateName := args[0]
			if !opts.IgnoreChecks {
				streams := userio.NewIOStreams(os.Stdin, os.Stdout, os.Stderr)
				repoParams := []config.Parameter[string]{cfg.Repositories.Templates}
				err := config.Update(cfg.IsPersisted(), cfg.GitHub.Token.Value, streams, cfg.Repositories.AllowDirty.Value, repoParams)
				if err != nil {
					return fmt.Errorf("failed to update config repos: %w", err)
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
