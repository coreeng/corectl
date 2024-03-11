package render

import (
	"fmt"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
	"github.com/coreeng/developer-platform/dpctl/template"
	"github.com/spf13/cobra"
	"os"
)

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
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
			if err := template.Render(&fulfilledT, templatesPath, targetPath); err != nil {
				return err
			}

			return nil
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)

	return templateRenderCmd
}
