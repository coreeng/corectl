package init

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"

	"os"
	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ConfigInitOpt struct {
	EnvironmentsRepo   string
	ConfigDir          string
	ConfigFile 	       string
	InitFile           string
	// RepositoriesDir    string // this shouldn't be configurable any more, we should use `config-dir/repositories` always
	GitHubToken        string
	GitHubOrganisation string
	NonInteractive     bool

	Streams userio.IOStreams
}

type ConfigErr struct {
	path, key string
	err       error
}

type p2pStageConfig struct {
	DefaultEnvs []string `yaml:"default-envs"`
}

type initConfig struct {
	Github struct {
		Organization string `yaml:"organization"`
	} `yaml:"github"`
	Repositories struct {
		Cplatform string `yaml:"cplatform"` // URL, not path
		Templates string `yaml:"templates"` // URL, not path
	} `yaml:"repositories"`
	ConfigDirectory struct {
		Directory string `yaml:"directory"` // the directory used for the config file
		File 	  string `yaml:"file"` // the filename of the config file
	} `yaml:"config-directory"`
	P2P struct {
		FastFeedback p2pStageConfig `yaml:"fast-feedback"`
		ExtendedTest p2pStageConfig `yaml:"extended-test"`
		Prod         p2pStageConfig `yaml:"prod"`
	} `yaml:"p2p"`
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
			cmd.SilenceUsage = true
			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				logger.Panic().With(zap.Error(err)).Msg("could not get non-interactive flag")
			}
			opt.NonInteractive = nonInteractive

			opt.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
				!opt.NonInteractive,
			)
			return run(cmd, &opt, cfg)
		},
	}

	newInitCmd.Flags().StringVarP(
		&opt.EnvironmentsRepo,
		"environments-repo",
		"e",
		"",
		"The GitHub repository to fetch the config file from, in the form `https://github.com/ORG/REPO`. This is mutually exclusive with the '--file' option.",
	)
	defaultConfigDir, err := config.BaseDir(cfg)
	if err != nil {
		fmt.Println("MARK: Error getting RepositoriesPath, using no default value")
		defaultConfigDir = ""
	}
	newInitCmd.Flags().StringVarP(
		&opt.ConfigDir,
		"config-dir",
		"d",
		defaultConfigDir,
		"Directory to store config yaml file, repositories, and more",
	)
	newInitCmd.Flags().StringVarP(
		&opt.ConfigFile,
		"config-file",
		"",
		"",
		"Filename to store the config yaml file, which will be stored in the config-dir",
	)
	newInitCmd.Flags().StringVarP(
		&opt.ConfigFile,
		"init-file",
		"f",
		"",
		"Config Initialization file. This is mutually exclusive with the '--environments-repo' option.", // not related to where the new corectl.yaml file should be stored
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
		"Default GitHub organisation to create apps.")

	return newInitCmd
}

func getConfigDirectory(cmd *cobra.Command, cfg *config.Config) string {
	var directory string
	initConfigDir, err := cmd.Flags().GetString("config-dir")
	if err != nil {
		logger.Panic().With(zap.Error(err)).Msg("could not get `--config-dir` flag")
	}
	// get environment variable for config dir
	envConfigDir := os.Getenv("CORECTL_CONFIG_DIR")
	if envConfigDir != "" && initConfigDir != "" {
		logger.Panic().Msg("both --config-dir option and CORECTL_CONFIG_DIR environment variable are set, please use only one");
	}
	if initConfigDir != "" {
		cfg.ConfigPaths.Directory.Value = initConfigDir
		opt.ConfigDir = initConfigDir
		fmt.Println("MARK: ConfigDir set to: ", initConfigDir)
	}
	return directory
}

func run(cmd *cobra.Command, opt *ConfigInitOpt, cfg *config.Config) error {
	

	// We don't allow the user to pass both the `--environments-repo` and the `--file` arguments
	environmentsRepoFlagValue, err := cmd.Flags().GetString("environments-repo")
	if err != nil {
		logger.Panic().With(zap.Error(err)).Msg("could not get `--environments-repo` flag")
	}
	initFileFlagValue, err := cmd.Flags().GetString("init-file")
	if err != nil {
		logger.Panic().With(zap.Error(err)).Msg("could not get `--environments-repo` flag")
	}
	if environmentsRepoFlagValue != "" && initFileFlagValue != "" {
		return fmt.Errorf("`--environments-repo` and `--file` are mutually exclusive")
	}

	// Parse the github token
	githubTokenInput := opt.createGitHubTokenInputSwitch()
	githubToken, err := githubTokenInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	githubClient := github.NewClient(nil).WithAuthToken(githubToken)

	// If the user passed the `--file` argument, read the init config from this file.
	// Otherwise, fetch the init config from the `corectl.yaml` file in the environments repo.
	var configBytes []byte
	var initFile string // This variable is just used for error messages
	if initFileFlagValue != "" {
		initFile = initFileFlagValue
		configBytes, err = os.ReadFile(initFileFlagValue)
		if err != nil {
			return err
		}
	} else {
		repoFile := "corectl.yaml"

		// Prompt user if `--environments-repo` wasn't set on the command line
		environmentsRepoInput := opt.createEnvironmentsRepoInputSwitch()
		environmentsRepoFlagValue, err := environmentsRepoInput.GetValue(opt.Streams)
		if err != nil {
			return err
		}
		initFile = fmt.Sprintf("%s/%s", environmentsRepoFlagValue, repoFile)

		configBytes, err = fetchInitConfigFromGitHub(githubClient, environmentsRepoFlagValue, repoFile)
		if err != nil {
			return err
		}
	}

	var initC initConfig
	err = yaml.Unmarshal(configBytes, &initC)
	if err != nil {
		return err
	}
	githubOrgInInitFile := initC.Github.Organization

	fmt.Println("opt.ConfigDir: ", opt.ConfigDir)
	repositoriesDir, err := cfg.RepositoriesDir()
	if err != nil {
		fmt.Println("MARK: Error getting RepositoriesPath, using no default value")
		return err
	}
	fmt.Println("repositoriesDir: ", repositoriesDir)

	if err = os.MkdirAll(repositoriesDir, 0o755); err != nil {
		return err
	}

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

	gitAuth := git.UrlTokenAuthMethod(githubToken)
	clonedRepositories, err := cloneRepositories(opt.Streams, gitAuth, githubClient, repositoriesDir, cplatformRepoFullname, templateRepoFullname)

	opt.Streams.Info("MARK: Cloned repositories into " + repositoriesDir)
	fmt.Println("MARK: Cloned repositories into " + repositoriesDir)

	if err != nil {
		return tryAppendHint(err, configBaseDir)
	}

	opt.Streams.Info("MARK: Cloned repositories, YAY!")

	cplatformRepoFullName, err := git.DeriveRepositoryFullname(clonedRepositories.cplatform)
	if err != nil {
		return err
	}
	if githubOrgInInitFile != "" && opt.GitHubOrganisation == "" {
		opt.GitHubOrganisation = githubOrgInInitFile
	}
	//TODO: can we fail quick if noninteractive mode is turned on and the flag is not set?
	githubOrgInput := opt.createGitHubOrganisationInputSwitch(cplatformRepoFullName.Organization())
	githubOrg, err := githubOrgInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}

	opt.Streams.Info("MARK: GITHUB ORG Switched")

	cfg.Repositories.CPlatform.Value = cplatformRepoFullName.HttpUrl()
	cfg.Repositories.Templates.Value = templateRepoFullname.HttpUrl()
	cfg.GitHub.Token.Value = githubToken
	cfg.GitHub.Organization.Value = githubOrg
	cfg.P2P.FastFeedback.DefaultEnvs.Value = initC.P2P.FastFeedback.DefaultEnvs
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = initC.P2P.ExtendedTest.DefaultEnvs
	cfg.P2P.Prod.DefaultEnvs.Value = initC.P2P.Prod.DefaultEnvs

	opt.Streams.Info("MARK: Ready to save")

	path, err := cfg.Save()
	if err != nil {
		fmt.Println("MARK: Error saving config. Returning error")
		return err
	}

	opt.Streams.Info("MARK: Saved")
	
	opt.Streams.Info("Configuration is saved to: " + path)
	return nil
}

func fetchInitConfigFromGitHub(githubClient *github.Client, repoUrl string, repoFile string) ([]byte, error) {
	// Parse the given repository URL
	re := regexp.MustCompile(`^http[s]?://[^/]+/([^/]+)/([^/]+)$`)
	matches := re.FindStringSubmatch(repoUrl)
	if matches == nil || len(matches) != 3 {
		return nil, fmt.Errorf("invalid repository URL '%s'", repoUrl)
	}
	org := matches[1]
	repo := matches[2]

	// Check what is the default branch for this repo
	ctx := context.Background()
	repoDetails, _, err := githubClient.Repositories.Get(ctx, org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository details for '%s': %w", repoUrl, err)
	}
	defaultBranch := repoDetails.GetDefaultBranch()
	if defaultBranch == "" {
		return nil, fmt.Errorf("default branch not found for '%s'", repoUrl)
	}

	// Fetch the file content for the default branch
	opt := github.RepositoryContentGetOptions{Ref: defaultBranch}
	data, _, _, err := githubClient.Repositories.GetContents(ctx, org, repo, repoFile, &opt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file '%s' from https://github.com/%s/%s: %w", repoFile, org, repo, err)
	}
	if data == nil || data.Content == nil {
		return nil, fmt.Errorf("file '%s' is empty or not found in https://github.com/%s/%s", repoFile, org, repo)
	}

	// Base64-decode the file content
	content, err := base64.StdEncoding.DecodeString(*data.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode content of file '%s' in https://github.com/%s/%s: %w", repoFile, org, repo, err)
	}
	return content, nil
}

func RepositoriesPath(c *config.Config) (string, error) {
	configPath, err := config.BaseDir(c)
	if err != nil {
		return "", err
	}
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
	streams.Wizard("Cloning repositories", "Cloned repositories")
	defer streams.CurrentHandler.Done()
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
	streams.CurrentHandler.Info(fmt.Sprintf("cloning platform repo: %s", cloneOpt.URL))
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
	streams.CurrentHandler.Info(fmt.Sprintf("cloning templates: %s", cloneOpt.URL))
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

func (opt *ConfigInitOpt) createEnvironmentsRepoInputSwitch() *userio.InputSourceSwitch[string, string] {
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.EnvironmentsRepo),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Environments repository URL",
				Placeholder:    "https://github.com/ORG/REPO",
				ValidateAndMap: userio.Required,
			}, nil
		},
		ValidateAndMap: userio.Required,
		ErrMessage:     "Environment repository URL is invalid",
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
