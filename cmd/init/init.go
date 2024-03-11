package init

import (
	"context"
	"errors"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
	"github.com/coreeng/developer-platform/dpctl/cmd/userio"
	"github.com/coreeng/developer-platform/dpctl/git"
	"github.com/coreeng/developer-platform/dpctl/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type InitOpt struct {
	File               string
	RepositoriesDir    string
	Tenant             string
	GitHubToken        string
	GitHubOrganisation string
	NonInteractive     bool

	Streams userio.IOStreams
}

func NewInitCmd(cfg *config.Config) *cobra.Command {
	opt := InitOpt{}
	newInitCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize dpctl before work",
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
	newInitCmd.Flags().StringVarP(
		&opt.RepositoriesDir,
		"repositories",
		"r",
		"",
		"Directory to store platform local repositories.",
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
		DPlatform string `yaml:"dplatform"`
		Templates string `yaml:"templates"`
	} `yaml:"repositories"`
	P2P struct {
		FastFeedback p2pStageConfig `yaml:"fast-feedback"`
		ExtendedTest p2pStageConfig `yaml:"extended-test"`
		Prod         p2pStageConfig `yaml:"prod"`
	} `yaml:"p2p"`
}

func run(opt *InitOpt, cfg *config.Config) error {
	initFileInput := opt.createInitFileInputSwitch()
	repositoriesDirInput := opt.createRepositoriesDirInputSwitch()
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

	repositoriesDir, err := repositoriesDirInput.GetValue(opt.Streams)
	if err != nil {
		return err
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

	dplatformRepoFullname, err := git.DeriveRepositoryFullnameFromUrl(initC.Repositories.DPlatform)
	if err != nil {
		return err
	}
	templateRepoFullname, err := git.DeriveRepositoryFullnameFromUrl(initC.Repositories.Templates)

	clonedRepositories, err := cloneRepositories(opt.Streams, githubClient, repositoriesDir, dplatformRepoFullname, templateRepoFullname)
	if err != nil {
		return err
	}

	dplatformRepoFullName, err := git.DeriveRepositoryFullname(clonedRepositories.dplatform)
	if err != nil {
		return err
	}
	//TODO: can we fail quick if noninteractive mode is turned on and the flag is not set?
	githubOrgInput := opt.createGitHubOrganisationInputSwitch(dplatformRepoFullName.Organization)
	githubOrg, err := githubOrgInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}

	tenants, err := tenant.List(clonedRepositories.dplatform.Path())
	if err != nil {
		return err
	}
	tenantInput := opt.createTenantInputSwitch(tenants)
	tenantName, err := tenantInput.GetValue(opt.Streams)

	cfg.Tenant.Value = string(tenantName)
	cfg.Repositories.DPlatform.Value = clonedRepositories.dplatform.Path()
	cfg.Repositories.Templates.Value = clonedRepositories.templates.Path()
	cfg.GitHub.Token.Value = githubToken
	cfg.GitHub.Organization.Value = githubOrg
	cfg.P2P.FastFeedback.DefaultEnvs.Value = initC.P2P.FastFeedback.DefaultEnvs
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = initC.P2P.ExtendedTest.DefaultEnvs
	cfg.P2P.Prod.DefaultEnvs.Value = initC.P2P.Prod.DefaultEnvs

	if err = cfg.Save(); err != nil {
		return err
	}

	err = opt.Streams.Info("Configuration is saved to: ", cfg.Path())
	if err != nil {
		return err
	}

	return nil
}

type cloneRepositoriesResult struct {
	dplatform *git.LocalRepository
	templates *git.LocalRepository
}

func cloneRepositories(
	streams userio.IOStreams,
	githubClient *github.Client,
	repositoriesDir string,
	dplatformRepoFullname git.RepositoryFullname,
	templatesRepoFullname git.RepositoryFullname,
) (cloneRepositoriesResult, error) {
	cloneReposSpinner := streams.Spinner("Cloning repositories...")
	defer cloneReposSpinner.Done()
	dplatformGitHubRepo, _, err := githubClient.Repositories.Get(
		context.Background(),
		dplatformRepoFullname.Organization,
		dplatformRepoFullname.Name,
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	dplatformRepository, err := git.CloneToLocalRepository(
		dplatformGitHubRepo.GetSSHURL(),
		filepath.Join(repositoriesDir, dplatformRepoFullname.Name),
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}

	templatesGitHubRepo, _, err := githubClient.Repositories.Get(
		context.Background(),
		templatesRepoFullname.Organization,
		templatesRepoFullname.Name,
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	templatesRepository, err := git.CloneToLocalRepository(
		templatesGitHubRepo.GetSSHURL(),
		filepath.Join(repositoriesDir, templatesRepoFullname.Name),
	)
	if err != nil {
		return cloneRepositoriesResult{}, err
	}
	return cloneRepositoriesResult{
		dplatform: dplatformRepository,
		templates: templatesRepository,
	}, nil
}

func (opt *InitOpt) createInitFileInputSwitch() *userio.InputSourceSwitch[string, string] {
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

func (opt *InitOpt) createRepositoriesDirInputSwitch() *userio.InputSourceSwitch[string, string] {
	fileValidator := userio.NewFileValidator(userio.FileValidatorOptions{DirsOnly: true})
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.RepositoriesDir),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			workingDir, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			return &userio.FilePicker{
				Prompt:         "Directory to store platform repositories:",
				WorkingDir:     workingDir,
				ValidateAndMap: fileValidator,
			}, nil
		},
		ValidateAndMap: fileValidator,
		ErrMessage:     "repositories dir is invalid",
	}
}

func (opt *InitOpt) createTenantInputSwitch(availableTenants []tenant.Tenant) *userio.InputSourceSwitch[string, tenant.Name] {
	validateFn := func(s string) (string, error) {
		s = strings.TrimSpace(s)
		for _, t := range availableTenants {
			if string(t.Name) == s {
				return s, nil
			}
		}
		return s, errors.New("unknown tenant")
	}
	return &userio.InputSourceSwitch[string, tenant.Name]{
		DefaultValue: userio.AsZeroable(opt.Tenant),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			items := make([]string, len(availableTenants))
			for i, t := range availableTenants {
				items[i] = string(t.Name)
			}
			return &userio.SingleSelect{
				Prompt: "Pick your tenancy",
				Items:  items,
			}, nil
		},
		ValidateAndMap: func(s string) (tenant.Name, error) {
			s, err := validateFn(s)
			if err != nil {
				return "", err
			}
			return tenant.Name(s), nil
		},
		ErrMessage: "tenant is invalid",
	}
}

func (opt *InitOpt) createGitHubTokenInputSwitch() *userio.InputSourceSwitch[string, string] {
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

func (opt *InitOpt) createGitHubOrganisationInputSwitch(suggestedOrg string) *userio.InputSourceSwitch[string, string] {
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
