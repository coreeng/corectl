package init

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type ConfigInitOpt struct {
	File               string
	RepositoriesDir    string
	GitHubToken        string
	GitHubOrganisation string
	NonInteractive     bool

	Streams userio.IOStreams
}

type ConfigErr struct {
	path, key string
	err       error
}

func (c ConfigErr) Error() string {
	return fmt.Sprintf("init config key %q invalid, path %q: %s", c.key, c.path, c.err)
}

func NewConfigInitCmd(cfg *config.Config) *cobra.Command {
	opt := ConfigInitOpt{}
	newInitCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize corectl before work",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				!opt.NonInteractive,
			)
			return run(&opt, cfg)
		},
	}

	newInitCmd.Flags().StringVarP(
		&opt.File,
		"file",
		"f",
		"",
		"Initialization file. Please, ask platform engineer to provide it.",
	)
	defaultRepositoriesPath, err := repositoriesPath()
	if err != nil {
		// We couldn't calculate the default value. That's fine, because the user could override it, it will be checked later.
		defaultRepositoriesPath = ""
	}
	newInitCmd.Flags().StringVarP(
		&opt.RepositoriesDir,
		"repositories",
		"r",
		defaultRepositoriesPath,
		"Directory to store platform local repositories. Default is near config file.",
	)
	newInitCmd.Flags().StringVarP(
		&opt.GitHubToken,
		"github-token",
		"t",
		"",
		"Personal GitHub access token.",
	)
	newInitCmd.Flags().StringVarP(
		&opt.GitHubOrganisation,
		"github-organization",
		"o",
		"",
		"GitHub organisation of your company.")
	newInitCmd.Flags().BoolVar(
		&opt.NonInteractive,
		"nonint",
		false,
		"Do not try to prompt user for missing input.",
	)

	return newInitCmd
}

type p2pStageConfig struct {
	DefaultEnvs []string `yaml:"default-envs"`
}
type initConfig struct {
	Repositories struct {
		Cplatform string `yaml:"cplatform"`
		Templates string `yaml:"templates"`
	} `yaml:"repositories"`
	P2P struct {
		FastFeedback p2pStageConfig `yaml:"fast-feedback"`
		ExtendedTest p2pStageConfig `yaml:"extended-test"`
		Prod         p2pStageConfig `yaml:"prod"`
	} `yaml:"p2p"`
}

func run(opt *ConfigInitOpt, cfg *config.Config) error {
	initFileInput := opt.createInitFileInputSwitch()
	githubTokenInput := opt.createGitHubTokenInputSwitch()

	initFile, err := initFileInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}

	fileBytes, err := os.ReadFile(initFile)
	if err != nil {
		return err
	}

	var initC initConfig
	err = yaml.Unmarshal(fileBytes, &initC)
	if err != nil {
		return err
	}

	repositoriesDir := opt.RepositoriesDir
	if repositoriesDir == "" {
		repositoriesDir, err = repositoriesPath()
		if err != nil {
			return err
		}
	}
	if err = os.MkdirAll(repositoriesDir, 0o755); err != nil {
		return err
	}

	githubToken, err := githubTokenInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	githubClient := github.NewClient(nil).
		WithAuthToken(githubToken)
	gitAuth := git.UrlTokenAuthMethod(githubToken)

	cplatformRepoFullname, err := git.DeriveRepositoryFullnameFromUrl(initC.Repositories.Cplatform)
	if err != nil {
		return ConfigErr{initFile, "cplatform", err}
	}
	templateRepoFullname, err := git.DeriveRepositoryFullnameFromUrl(initC.Repositories.Templates)
	if err != nil {
		return ConfigErr{initFile, "templates", err}
	}
	configBaseDir, err := cfg.BaseDir()
	if err != nil {
		return fmt.Errorf("failed to construct corectl config directory path: %w", err)
	}
	clonedRepositories, err := cloneRepositories(opt.Streams, gitAuth, githubClient, repositoriesDir, cplatformRepoFullname, templateRepoFullname)
	if err != nil {
		return tryAppendHint(err, configBaseDir)
	}

	cplatformRepoFullName, err := git.DeriveRepositoryFullname(clonedRepositories.cplatform)
	if err != nil {
		return err
	}
	//TODO: can we fail quick if noninteractive mode is turned on and the flag is not set?
	githubOrgInput := opt.createGitHubOrganisationInputSwitch(cplatformRepoFullName.Organization())
	githubOrg, err := githubOrgInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}

	cfg.Repositories.CPlatform.Value = clonedRepositories.cplatform.Path()
	cfg.Repositories.Templates.Value = clonedRepositories.templates.Path()
	cfg.GitHub.Token.Value = githubToken
	cfg.GitHub.Organization.Value = githubOrg
	cfg.P2P.FastFeedback.DefaultEnvs.Value = initC.P2P.FastFeedback.DefaultEnvs
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = initC.P2P.ExtendedTest.DefaultEnvs
	cfg.P2P.Prod.DefaultEnvs.Value = initC.P2P.Prod.DefaultEnvs

	if err = cfg.Save(); err != nil {
		return err
	}

	opt.Streams.Info("Configuration is saved to: ", cfg.Path())
	opt.Streams.Info(`
To keep configuration up to date, periodically run:
corectl config update`,
	)
	return nil
}

func repositoriesPath() (string, error) {
	configPath, err := config.Path()
	if err != nil {
		return "", err
	}
	configPath = filepath.Dir(configPath)
	return filepath.Join(configPath, "repositories"), nil
}

type cloneRepositoriesResult struct {
	cplatform *git.LocalRepository
	templates *git.LocalRepository
}

func cloneRepositories(
	streams userio.IOStreams,
	gitAuth git.AuthMethod,
	githubClient *github.Client,
	repositoriesDir string,
	cplatformRepoFullname git.RepositoryFullname,
	templatesRepoFullname git.RepositoryFullname,
) (cloneRepositoriesResult, error) {
	cloneReposSpinner := streams.Spinner("Cloning repositories...")
	defer cloneReposSpinner.Done()
	cplatformGitHubRepo, _, err := githubClient.Repositories.Get(
		context.Background(),
		cplatformRepoFullname.Organization(),
		cplatformRepoFullname.Name(),
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	cloneOpt := git.CloneOp{
		URL:        cplatformGitHubRepo.GetCloneURL(),
		TargetPath: filepath.Join(repositoriesDir, cplatformRepoFullname.Name()),
		Auth:       gitAuth,
	}
	cplatformRepository, err := git.CloneToLocalRepository(cloneOpt)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}

	templatesGitHubRepo, _, err := githubClient.Repositories.Get(
		context.Background(),
		templatesRepoFullname.Organization(),
		templatesRepoFullname.Name(),
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	cloneOpt = git.CloneOp{
		URL:        templatesGitHubRepo.GetCloneURL(),
		TargetPath: filepath.Join(repositoriesDir, templatesRepoFullname.Name()),
		Auth:       gitAuth,
	}
	templatesRepository, err := git.CloneToLocalRepository(cloneOpt)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	return cloneRepositoriesResult{
		cplatform: cplatformRepository,
		templates: templatesRepository,
	}, nil
}

func tryAppendHint(err error, configBaseDir string) error {
	switch {
	case errors.As(err, &git.RepositoryCloneErr{}):
		return fmt.Errorf("%w: initialised already? run `corectl config update` to update repositories, alternatively to initialise again delete corectl config dir at %q and run `corectl config init`", err, configBaseDir)
	default:
		return err
	}
}

func (opt *ConfigInitOpt) createInitFileInputSwitch() *userio.InputSourceSwitch[string, string] {
	fileValidator := userio.NewFileValidator(userio.FileValidatorOptions{
		ExistingOnly: true,
		FilesOnly:    true,
	})
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.File),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			dir, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			return &userio.FilePicker{
				Prompt:         "Select file with configuration for initialization:",
				WorkingDir:     dir,
				ValidateAndMap: fileValidator,
			}, nil
		},
		ValidateAndMap: fileValidator,
		ErrMessage:     "init file is invalid",
	}
}

func (opt *ConfigInitOpt) createGitHubTokenInputSwitch() *userio.InputSourceSwitch[string, string] {
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.GitHubToken),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Personal GitHub access token",
				Placeholder:    "ghc_qwerty",
				ValidateAndMap: userio.Required,
			}, nil
		},
		ValidateAndMap: userio.Required,
		ErrMessage:     "GitHub token is invalid",
	}
}

func (opt *ConfigInitOpt) createGitHubOrganisationInputSwitch(suggestedOrg string) *userio.InputSourceSwitch[string, string] {
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.GitHubOrganisation),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "GitHub organisation",
				InitialValue:   suggestedOrg,
				Placeholder:    suggestedOrg,
				ValidateAndMap: userio.Required,
			}, nil
		},
		ValidateAndMap: userio.Required,
		ErrMessage:     "GitHub organisation is invalid",
	}
}
