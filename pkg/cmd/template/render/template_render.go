package render

import (
	"fmt"
	"os"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
)

type TemplateRenderOpts struct {
	Streams userio.IOStreams

	IgnoreChecks  bool
	ArgsFile      string
	Args          []string
	TemplateName  string
	TargetPath    string
	TemplatesPath string
	DryRun        bool
}

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
	var opts = TemplateRenderOpts{}
	templateRenderCmd := &cobra.Command{
		Use:   "render <template-name> <target-path>",
		Short: "Render template locally",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(cmd.InOrStdin(), cmd.OutOrStdout())
			opts.TemplateName = args[0]
			opts.TargetPath = args[1]
			opts.TemplatesPath = cfg.Repositories.Templates.Value

			if !opts.IgnoreChecks {
				if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates, false); err != nil {
					return err
				}
			}

			return run(opts)
		},
	}

	templateRenderCmd.Flags().BoolVar(
		&opts.IgnoreChecks,
		"ignore-checks",
		false,
		"Ignore checks for uncommitted changes and branch status",
	)

	templateRenderCmd.Flags().StringVar(
		&opts.ArgsFile,
		"args-file",
		"",
		"Path to YAML file containing template arguments",
	)
	templateRenderCmd.Flags().StringSliceVarP(
		&opts.Args,
		"arg",
		"a",
		[]string{},
		"Template argument in the format: <arg-name>=<arg-value>",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)

	return templateRenderCmd
}

func run(opts TemplateRenderOpts) error {
	stat, err := os.Stat(opts.TargetPath)
	if err != nil {
		pathError := err.(*os.PathError)
		return fmt.Errorf("%s: %v", pathError.Path, pathError.Err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s: not a directory", opts.TargetPath)
	}

	templ, err := template.FindByName(opts.TemplatesPath, opts.TemplateName)
	if err != nil {
		return err
	}
	if templ == nil {
		return fmt.Errorf("%s: unknown template", opts.TemplateName)
	}

	templateRenderer := &FlagsAwareTemplateRenderer{
		ArgsFile: opts.ArgsFile,
		Args:     opts.Args,
		Streams:  opts.Streams,
	}

	if err := templateRenderer.Render(templ, opts.TargetPath, opts.DryRun); err != nil {
		return err
	}

	return nil
}

type TemplateRenderer interface {
	Render(spec *template.Spec, targetDirectory string, dryRun bool, additionalArgs ...template.Argument) error
}

type FlagsAwareTemplateRenderer struct {
	ArgsFile string
	Args     []string
	Streams  userio.IOStreams
}

func (r *FlagsAwareTemplateRenderer) Render(spec *template.Spec, targetDirectory string, dryRun bool, additionalArgs ...template.Argument) error {
	if spec == nil {
		return nil
	}

	args, err := CollectArgsFromAllSources(
		spec,
		r.ArgsFile,
		r.Args,
		r.Streams,
		additionalArgs,
	)
	if err != nil {
		return err
	}

	fulfilledTemplate := &template.FulfilledTemplate{
		Spec:      spec,
		Arguments: args,
	}

	if !dryRun {
		err = template.Render(fulfilledTemplate, targetDirectory)
	}
	return err
}

type StubTemplateRenderer struct {
	Renderer             TemplateRenderer
	PassedAdditionalArgs [][]template.Argument
}

func (r *StubTemplateRenderer) Render(spec *template.Spec, targetDirectory string, dryRun bool, additionalArgs ...template.Argument) error {
	r.PassedAdditionalArgs = append(r.PassedAdditionalArgs, additionalArgs)
	return r.Renderer.Render(spec, targetDirectory, dryRun, additionalArgs...)
}
