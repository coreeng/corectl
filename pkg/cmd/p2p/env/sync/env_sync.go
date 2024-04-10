package sync

import (
	"context"
	"os"

	"path/filepath"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type EnvCreateOpts struct {
	NonInteractive  bool
	RepositoriesDir string
	AppRepo         string
	Name            string
	DPlatformRepo   string
	Streams         userio.IOStreams
}

func NewP2PSyncCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvCreateOpts{}
	var syncEnvironmentsCmd = &cobra.Command{
		Use:   "sync <environment> ",
		Short: "Synchronise Environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			opts.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				!opts.NonInteractive,
			)
			return run(&opts, cfg)
		},
	}

	syncEnvironmentsCmd.Flags().StringVarP(
		&opts.RepositoriesDir,
		"repositories",
		"r",
		"",
		"Directory to store platform local repositories. Default is near config file.")
	err := syncEnvironmentsCmd.MarkFlagRequired("repositories")
	if err != nil {
		return nil, err
	}
	syncEnvironmentsCmd.Flags().StringVarP(
		&opts.DPlatformRepo,
		"dplatformrepo",
		"d",
		"",
		"DPlatform Repository URL")
	err = syncEnvironmentsCmd.MarkFlagRequired("dplatformrepo")
	if err != nil {
		return nil, err
	}

	syncEnvironmentsCmd.Flags().StringVarP(
		&opts.AppRepo,
		"apprepo",
		"a",
		"",
		"Application Repo")
	err = syncEnvironmentsCmd.MarkFlagRequired("apprepo")
	if err != nil {
		return nil, err
	}

	syncEnvironmentsCmd.Flags().BoolVar(
		&opts.NonInteractive,
		"nonint",
		false,
		"Disable interactive inputs")

	return syncEnvironmentsCmd, nil
}

func run(opts *EnvCreateOpts, cfg *config.Config) error {

	type domainItem struct {
		Name   string `yaml:"name"`
		Domain string `yaml:"domain"`
	}
	type ingressDomains struct {
		DomainItem []domainItem `yaml:"ingressDomains,flow"`
	}

	type envConfig struct {
		Platform struct {
			ProjectId     string `yaml:"projectId"`
			ProjectNumber string `yaml:"projectNumber"`
		} `yaml:"platform"`
		InternalServices struct {
			Domain string `yaml:"domain"`
		} `yaml:"internalServices"`
		IngressDomains []domainItem `yaml:"ingressDomains"`
	}

	repositoriesDir := opts.RepositoriesDir
	if repositoriesDir == "" {
		configPath, err := config.Path()
		if err != nil {
			return err
		}
		configPath = filepath.Dir(configPath)
		repositoriesDir = filepath.Join(configPath, "repositories")
	}
	if err := os.MkdirAll(repositoriesDir, 0o755); err != nil {
		return err
	}
	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)

	dplatformRepoFullname, err := git.DeriveRepositoryFullnameFromUrl(opts.DPlatformRepo)

	// Clone or Reset DPlatform Repo
	clonedRepository, err := cloneRepository(opts.Streams, gitAuth, githubClient, repositoriesDir, dplatformRepoFullname)
	if err != nil {
		return err
	}

	// Read Existing Config
	configFile := filepath.Join(repositoriesDir, dplatformRepoFullname.Name, "environments", opts.Name, "config.yaml")
	fileBytes, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var envC envConfig
	err = yaml.Unmarshal(fileBytes, &envC)
	if err != nil {
		return err
	}
	var domain domainItem
	for _, domain = range envC.IngressDomains {
		if domain.Name == "default" {
			break
		}
	}
	varsToCreate := []github.ActionsVariable{
		{
			Name:  "BASE_DOMAIN",
			Value: domain.Domain,
		},
		{
			Name:  "INTERNAL_SERVICES_DOMAIN",
			Value: envC.InternalServices.Domain,
		},
		{
			Name:  "DPLATFORM",
			Value: opts.Name,
		},
		{
			Name:  "PROJECT_ID",
			Value: envC.Platform.ProjectId,
		},
		{
			Name:  "PROJECT_NUMBER",
			Value: envC.Platform.ProjectNumber,
		},
	}
	_ = varsToCreate

	opts.Streams.Info("Repository local path " + clonedRepository.dplatform.Path())
	opts.Streams.Info("Creating Environment...")
	environments, _, err := githubClient.Repositories.CreateUpdateEnvironment(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.AppRepo,
		opts.Name,
		&github.CreateUpdateEnvironment{},
	)
	if err != nil {
		return err
	}

	repository, _, err := githubClient.Repositories.Get(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.AppRepo)
	opts.Streams.Info("Repo: " + *repository.Name)

	for i := range varsToCreate {
		opts.Streams.Info("Removing var: " + varsToCreate[i].Name)
		response, err := githubClient.Actions.DeleteEnvVariable(
			context.Background(),
			int(repository.GetID()),
			*environments.Name,
			varsToCreate[i].Name)
		if err != nil {
			if response.StatusCode == 404 {
				opts.Streams.Info("Skipping " + varsToCreate[i].Name + ", not present")
			} else {
				return err
			}
		}
		opts.Streams.Info("Creating Var: " + varsToCreate[i].Name)

		_, err = githubClient.Actions.CreateEnvVariable(
			context.Background(),
			int(repository.GetID()),
			*environments.Name,
			&varsToCreate[i],
		)
		if err != nil {
			return err
		}
	}

	return nil
}

type cloneRepositoryResult struct {
	dplatform *git.LocalRepository
}

func cloneRepository(
	streams userio.IOStreams,
	gitAuth git.AuthMethod,
	githubClient *github.Client,
	repositoryDir string,
	dplatformRepoFullname git.RepositoryFullname,
) (cloneRepositoryResult, error) {
	var dplatformRepository *git.LocalRepository
	cloneReposSpinner := streams.Spinner("Cloning Repository...")
	defer cloneReposSpinner.Done()
	if _, err := os.Stat(filepath.Join(repositoryDir, dplatformRepoFullname.Name)); err == nil {
		streams.Info("Local Repository exists, resetting...")
		dplatformRepository, err = git.OpenAndResetRepositoryState(filepath.Join(repositoryDir, dplatformRepoFullname.Name))
	} else {
		dplatformGitHubRepo, _, err := githubClient.Repositories.Get(
			context.Background(),
			dplatformRepoFullname.Organization,
			dplatformRepoFullname.Name)

		if err != nil {
			return cloneRepositoryResult{}, err
		}
		dplatformRepository, err = git.CloneToLocalRepository(git.CloneOp{
			URL:        dplatformGitHubRepo.GetCloneURL(),
			TargetPath: filepath.Join(repositoryDir, dplatformRepoFullname.Name),
			Auth:       gitAuth,
		})
		if err != nil {
			return cloneRepositoryResult{}, err
		}
	}
	return cloneRepositoryResult{
		dplatform: dplatformRepository,
	}, nil
}
