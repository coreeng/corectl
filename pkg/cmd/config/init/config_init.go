package init

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ConfigInitOpt struct {
	EnvironmentsRepo   string
	File               string
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
	newInitCmd.Flags().StringVarP(
		&opt.File,
		"file",
		"f",
		"",
		"Initialization file. This is mutually exclusive with the '--environments-repo' option.",
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

type p2pStageConfig struct {
	DefaultEnvs []string `yaml:"default-envs"`
}
type initConfig struct {
	Github struct {
		Organization string `yaml:"organization"`
	} `yaml:"github"`
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

func run(cmd *cobra.Command, opt *ConfigInitOpt, cfg *config.Config) error {
	// We don't allow the user to pass both the `--environments-repo` and the `--file` arguments
	environmentsRepoFlagValue, err := cmd.Flags().GetString("environments-repo")
	if err != nil {
		logger.Panic().With(zap.Error(err)).Msg("could not get `--environments-repo` flag")
	}
	fileFlagValue, err := cmd.Flags().GetString("file")
	if err != nil {
		logger.Panic().With(zap.Error(err)).Msg("could not get `--environments-repo` flag")
	}
	if environmentsRepoFlagValue != "" && fileFlagValue != "" {
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
	if fileFlagValue != "" {
		initFile = fileFlagValue
		configBytes, err = os.ReadFile(fileFlagValue)
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

	if err = os.MkdirAll(configpath.GetCorectlCacheDir(), 0o755); err != nil {
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
	clonedRepositories, err := cloneRepositories(opt.Streams, gitAuth, githubClient, cplatformRepoFullname, templateRepoFullname)
	if err != nil {
		return tryAppendHint(err, configBaseDir)
	}

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

	cfg.Repositories.CPlatform.Value = initC.Repositories.Cplatform
	cfg.Repositories.Templates.Value = initC.Repositories.Templates
	cfg.GitHub.Token.Value = githubToken
	cfg.GitHub.Organization.Value = githubOrg
	cfg.P2P.FastFeedback.DefaultEnvs.Value = initC.P2P.FastFeedback.DefaultEnvs
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = initC.P2P.ExtendedTest.DefaultEnvs
	cfg.P2P.Prod.DefaultEnvs.Value = initC.P2P.Prod.DefaultEnvs

	if err = cfg.Save(); err != nil {
		return err
	}

	opt.Streams.Info("Configuration is saved to: " + cfg.Path())
	opt.Streams.Info(`
To keep configuration up to date, periodically run:
corectl config update`,
	)
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

type cloneRepositoriesResult struct {
	cplatform *git.LocalRepository
	templates *git.LocalRepository
}

func cloneRepositories(
	streams userio.IOStreams,
	gitAuth git.AuthMethod,
	githubClient *github.Client,
	cplatformRepoFullname git.RepositoryFullname,
	templatesRepoFullname git.RepositoryFullname,
) (cloneRepositoriesResult, error) {
	streams.Wizard("Cloning repositories", "Cloned repositories")
	defer streams.CurrentHandler.Done()
	// Use retry logic to handle potential propagation delays
	cplatformGitHubRepo, _, err := git.RetryGitHubAPI(
		func() (*github.Repository, *github.Response, error) {
			return githubClient.Repositories.Get(
				context.Background(),
				cplatformRepoFullname.Organization(),
				cplatformRepoFullname.Name(),
			)
		},
		git.DefaultMaxRetries,
		git.DefaultBaseDelay,
	)
	if err != nil {
		return cloneRepositoriesResult{}, fmt.Errorf("failed to get cplatform repository after retries: %w", err)
	}
	cloneOpt := git.CloneOp{
		URL:        cplatformGitHubRepo.GetCloneURL(),
		TargetPath: configpath.GetCorectlCPlatformDir(),
		Auth:       gitAuth,
	}
	streams.CurrentHandler.Info(fmt.Sprintf("cloning platform repo: %s", cloneOpt.URL))
	cplatformRepository, err := git.CloneToLocalRepository(cloneOpt)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}

	// Use retry logic to handle potential propagation delays
	templatesGitHubRepo, _, err := git.RetryGitHubAPI(
		func() (*github.Repository, *github.Response, error) {
			return githubClient.Repositories.Get(
				context.Background(),
				templatesRepoFullname.Organization(),
				templatesRepoFullname.Name(),
			)
		},
		git.DefaultMaxRetries,
		git.DefaultBaseDelay,
	)
	if err != nil {
		return cloneRepositoriesResult{}, fmt.Errorf("failed to get templates repository after retries: %w", err)
	}
	cloneOpt = git.CloneOp{
		URL:        templatesGitHubRepo.GetCloneURL(),
		TargetPath: configpath.GetCorectlTemplatesDir(),
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
