package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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
	AppName       string
	Description   string
	Config        string
}

func NewTemplateRenderCmd(cfg *config.Config) *cobra.Command {
	var opts = TemplateRenderOpts{}
	templateRenderCmd := &cobra.Command{
		Use:   "render <template-name> <target-path>",
		Short: "Render template locally",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			opts.Streams = userio.NewIOStreams(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.OutOrStderr())
			opts.TemplateName = args[0]
			opts.TargetPath = args[1]
			opts.TemplatesPath = cfg.Repositories.Templates.Value
			return run(opts, cfg)
		},
	}

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
	templateRenderCmd.Flags().StringVar(
		&opts.AppName,
		"name",
		"",
		"Application name for app.yaml (can also be provided via --arg or --args-file)",
	)
	templateRenderCmd.Flags().StringVar(
		&opts.Description,
		"description",
		"",
		"Application description for app.yaml",
	)
	templateRenderCmd.Flags().StringVar(
		&opts.Config,
		"config",
		"",
		"JSON config to merge with template config for app.yaml",
	)

	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		templateRenderCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		templateRenderCmd.Flags(),
	)

	return templateRenderCmd
}

func run(opts TemplateRenderOpts, cfg *config.Config) error {
	// Skip repository update if a custom templates path was provided via --templates flag
	// We check if the templates path differs from the default GetCorectlTemplatesDir()
	defaultTemplatesPath := configpath.GetCorectlTemplatesDir()
	if opts.TemplatesPath != "" && opts.TemplatesPath != defaultTemplatesPath {
		// Using custom templates directory, skip repository update
	} else {
		// Using default templates directory, update repository
		repoParams := []config.Parameter[string]{cfg.Repositories.Templates}
		err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
		if err != nil {
			return fmt.Errorf("failed to update config repos: %w", err)
		}
	}

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

	// Build additional args for app.yaml creation
	var additionalArgs []template.Argument
	if opts.AppName != "" {
		additionalArgs = append(additionalArgs, template.Argument{Name: "name", Value: opts.AppName})
	}
	if opts.Description != "" {
		additionalArgs = append(additionalArgs, template.Argument{Name: "description", Value: opts.Description})
	}

	if err := templateRenderer.Render(templ, opts.TargetPath, opts.DryRun, opts.Config, additionalArgs...); err != nil {
		return err
	}

	return nil
}

type TemplateRenderer interface {
	Render(spec *template.Spec, targetDirectory string, dryRun bool, configJSON string, additionalArgs ...template.Argument) error
}

type FlagsAwareTemplateRenderer struct {
	ArgsFile string
	Args     []string
	Streams  userio.IOStreams
}

func (r *FlagsAwareTemplateRenderer) Render(spec *template.Spec, targetDirectory string, dryRun bool, configJSON string, additionalArgs ...template.Argument) error {
	if spec == nil {
		return nil
	}

	// Merge template config with configJSON overrides
	mergedConfig := make(map[string]any)
	if spec.Config != nil {
		mergedConfig = DeepMerge(mergedConfig, spec.Config)
	}
	if configJSON != "" {
		var configOverrides map[string]any
		if err := json.Unmarshal([]byte(configJSON), &configOverrides); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}
		mergedConfig = DeepMerge(mergedConfig, configOverrides)
	}

	// Add merged config to additional args
	additionalArgs = append(additionalArgs, template.Argument{Name: "config", Value: mergedConfig})

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

	logger.Debug().With(
		zap.String("spec.Name", spec.Name),
		zap.String("spec.Description", spec.Description),
		zap.String("target_dir", targetDirectory),
		zap.Bool("dry_run", dryRun)).
		Msg("rendering template")
	if !dryRun {
		if err = template.Render(fulfilledTemplate, targetDirectory); err != nil {
			return err
		}
	}

	// Write app.yaml - extract name and description from arguments
	appName, description := extractAppYAMLArgs(args)
	if appName != "" {
		if err := WriteAppYAML(targetDirectory, appName, description, mergedConfig, dryRun); err != nil {
			return err
		}
	}

	return nil
}

// extractAppYAMLArgs extracts name and description from template arguments
func extractAppYAMLArgs(args []template.Argument) (name string, description string) {
	for _, arg := range args {
		switch arg.Name {
		case "name":
			if s, ok := arg.Value.(string); ok {
				name = s
			}
		case "description":
			if s, ok := arg.Value.(string); ok {
				description = s
			}
		}
	}
	return
}

type StubTemplateRenderer struct {
	Renderer             TemplateRenderer
	PassedAdditionalArgs [][]template.Argument
	PassedConfigJSON     []string
}

func (r *StubTemplateRenderer) Render(spec *template.Spec, targetDirectory string, dryRun bool, configJSON string, additionalArgs ...template.Argument) error {
	r.PassedAdditionalArgs = append(r.PassedAdditionalArgs, additionalArgs)
	r.PassedConfigJSON = append(r.PassedConfigJSON, configJSON)
	return r.Renderer.Render(spec, targetDirectory, dryRun, configJSON, additionalArgs...)
}

// DeepMerge merges override into base, recursively merging nested maps
func DeepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any)

	for k, v := range base {
		result[k] = v
	}

	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := baseVal.(map[string]any)
			overrideMap, overrideIsMap := v.(map[string]any)
			if baseIsMap && overrideIsMap {
				result[k] = DeepMerge(baseMap, overrideMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}

// appYAML represents the structure of app.yaml with fields in desired order
type appYAML struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Config      map[string]any `yaml:"config"`
}

// WriteAppYAML writes the app configuration to app.yaml at the target directory
func WriteAppYAML(targetDir string, appName string, description string, config map[string]any, dryRun bool) error {
	appConfigPath := filepath.Join(targetDir, "app.yaml")
	logger.Debug().With(
		zap.String("path", appConfigPath),
		zap.Bool("dry_run", dryRun)).
		Msg("writing app.yaml config file")

	if dryRun {
		return nil
	}

	// Build app config with ordered fields: name, description, config
	appConfig := appYAML{
		Name:        appName,
		Description: description,
		Config:      config,
	}

	// Use custom encoder for 2-space indentation
	var buf bytes.Buffer
	buf.WriteString("---\n")
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(appConfig); err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close YAML encoder: %w", err)
	}

	if err := os.WriteFile(appConfigPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write app.yaml: %w", err)
	}

	return nil
}
