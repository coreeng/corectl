package export

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/selector"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/p2p"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type exportOpts struct {
	tenant          string
	environmentName string
	repoPath        string
	streams         userio.IOStreams
}

func (eo *exportOpts) processFlags(cplatRepoPath string) (*p2p.EnvVarContext, error) {
	argTenant, err := selector.Tenant(cplatRepoPath, eo.tenant, eo.streams)
	if err != nil {
		return nil, err
	}
	argEnv, err := selector.Environment(cplatRepoPath, eo.environmentName, eo.streams)
	if err != nil {
		return nil, err
	}
	argRepo, err := eo.appRepoPathSelector()
	if err != nil {
		return nil, err
	}

	return &p2p.EnvVarContext{Tenant: argTenant, Environment: argEnv, AppRepo: argRepo}, nil
}

func (eo *exportOpts) appRepoPathSelector() (*git.LocalRepository, error) {
	inputRepoPath := eo.createRepoPathInputSwitch(eo.repoPath)
	repoPathOutput, err := inputRepoPath.GetValue(eo.streams)
	if err != nil {
		return nil, err
	}
	repo, err := git.OpenLocalRepository(repoPathOutput)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (eo *exportOpts) createRepoPathInputSwitch(defaultName string) userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		if _, err := os.Stat(inp); err != nil {
			return "", fmt.Errorf("cannot load repo at path %s: %w", inp, err)
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(defaultName),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:      "App repository path:",
				Placeholder: "repoPath",
				ValidateAndMap: func(inp string) (string, error) {
					name, err := validateFn(inp)
					return name, err
				},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     fmt.Sprintf("invalid repository path %s", defaultName),
	}
}

func NewP2PExportCmd(cfg *config.Config) (*cobra.Command, error) {
	opts := &exportOpts{}
	var exportCommand = &cobra.Command{
		Use:   "export",
		Short: "Produce export statements for environment variables required to execute p2p targets, to automatically export in current shell run 'eval $(corectl p2p export [flags])'",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(opts, &cfg.Repositories.CPlatform)
		},
	}

	currDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	exportCommand.Flags().StringVarP(
		&opts.environmentName,
		"environment",
		"e",
		"",
		"Environment to export variables for P2P",
	)
	exportCommand.Flags().StringVarP(
		&opts.tenant,
		"tenant",
		"t",
		"",
		"Tenant to export variables for P2P",
	)
	exportCommand.Flags().StringVarP(
		&opts.repoPath,
		"repoPath",
		"r",
		currDir,
		"Local repository path to export variables for P2P, defaults to current exec directory",
	)
	return exportCommand, nil
}

func run(opts *exportOpts, cplatRepoPath *config.Parameter[string]) error {
	if _, err := config.ResetConfigRepositoryState(cplatRepoPath); err != nil {
		return err
	}
	context, err := opts.processFlags(cplatRepoPath.Value)
	if err != nil {
		return err
	}
	p2pVars, err := p2p.NewP2pEnvVariables(context)
	if err != nil {
		return err
	}
	exportCmd, err := p2pVars.AsExportCmd()
	if err != nil {
		return err
	}
	opts.streams.Info(exportCmd)
	return nil
}