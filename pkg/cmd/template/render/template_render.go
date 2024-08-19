package render

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
	"os"
)

type TemplateRenderOpts struct {
	IgnoreChecks bool
}

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
	var opts = TemplateRenderOpts{}
	templateRenderCmd := &cobra.Command{
		Use:   "render <template-name> <target-path>",
		Short: "Render template locally",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			targetPath := args[1]

			stat, err := os.Stat(targetPath)
			if err != nil {
				pathError := err.(*os.PathError)
				return fmt.Errorf("%s: %v", pathError.Path, pathError.Err)
			}
			if !stat.IsDir() {
				return fmt.Errorf("%s: not a directory", targetPath)
			}

			if !opts.IgnoreChecks {
				if _, err = config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
					return err
				}
			}

			templatesPath := cfg.Repositories.Templates.Value
			t, err := template.FindByName(templatesPath, templateName)
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("%s: unknown template", templateName)
			}

			fulfilledT := template.FulfilledTemplate{
				Spec:      t,
				Arguments: []template.Argument{},
			}
			if err := template.Render(&fulfilledT, targetPath); err != nil {
				return err
			}

			return nil
		},
	}

	templateRenderCmd.Flags().BoolVar(
		&opts.IgnoreChecks,
		"ignore-checks",
		false,
		"Ignore checks for uncommitted changes and branch status",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)

	return templateRenderCmd
}
