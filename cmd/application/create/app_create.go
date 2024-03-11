package create

import (
	"fmt"
	"github.com/coreeng/developer-platform/dpctl/application"
	"github.com/coreeng/developer-platform/dpctl/cmd/config"
	"github.com/coreeng/developer-platform/dpctl/cmd/userio"
	"github.com/coreeng/developer-platform/dpctl/environment"
	"github.com/coreeng/developer-platform/dpctl/template"
	"github.com/coreeng/developer-platform/dpctl/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"slices"
)

type AppCreateOpt struct {
	Name           string
	LocalPath      string
	NonInteractive bool
	FromTemplate   string

	Streams userio.IOStreams
}

func NewAppCreateCmd(cfg *config.Config) (*cobra.Command, error) {
	var opts = AppCreateOpt{}
	var appCreateCmd = &cobra.Command{
		Use:   "create <app-name> [<local-path>]",
		Short: "Create new application",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			if len(args) > 1 {
				opts.LocalPath = args[1]
			} else {
				opts.LocalPath = "./" + opts.Name
			}
			opts.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				!opts.NonInteractive,
			)
			return run(&opts, cfg)
		},
	}

	appCreateCmd.Flags().StringVarP(
		&opts.FromTemplate,
		"from-template",
		"t",
		"",
		"Template to use to create an application",
	)
	if err := appCreateCmd.MarkFlagRequired("from-template"); err != nil {
		return nil, err
	}
	appCreateCmd.Flags().BoolVar(
		&opts.NonInteractive,
		"nonint",
		false,
		"Disable interactive inputs",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Tenant,
		appCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Token,
		appCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Organization,
		appCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.DPlatform,
		appCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		appCreateCmd.Flags(),
	)

	return appCreateCmd, nil
}

func run(opts *AppCreateOpt, cfg *config.Config) error {
	appTenantName := tenant.Name(cfg.Tenant.Value)
	appTenant, err := tenant.FindByName(cfg.Repositories.DPlatform.Value, appTenantName)
	if err != nil {
		return err
	}
	if appTenant == nil {
		return fmt.Errorf("%s: unknown tenant", appTenantName)
	}

	fromTemplate, err := template.FindByName(cfg.Repositories.Templates.Value, opts.FromTemplate)
	if err != nil {
		return err
	}
	if fromTemplate == nil {
		return fmt.Errorf("%s: unknown template", opts.FromTemplate)
	}

	existingEnvs, err := environment.List(cfg.Repositories.DPlatform.Value)
	if err != nil {
		return err
	}

	if err = userio.ValidateFilePath(opts.LocalPath, userio.FileValidatorOptions{
		DirsOnly:   true,
		DirIsEmpty: true,
	}); err != nil {
		return err
	}

	var tenantEnvs []environment.Environment
	for _, env := range existingEnvs {
		if slices.Contains(appTenant.Environments, env.Environment) {
			tenantEnvs = append(tenantEnvs, env)
		}
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)

	createdAppResult, err := createNewApp(opts, cfg, githubClient, appTenant, fromTemplate, existingEnvs)
	if err != nil {
		return err
	}
	if err = opts.Streams.Info("Created repository: ", createdAppResult.RepositoryFullname.HttpUrl()); err != nil {
		return err
	}

	tenantUpdateResult, err := createPRWithUpdatedReposListForTenant(opts, cfg, githubClient, appTenant, createdAppResult)
	if err != nil {
		return err
	}

	nextStepsMessage := fmt.Sprintf(
		nextStepsMessageTemplate,
		tenantUpdateResult.PRUrl,
		createdAppResult.RepositoryFullname.ActionsHttpUrl(),
		createdAppResult.RepositoryFullname.AsString(),
		createdAppResult.RepositoryFullname.AsString(),
	)
	if err := opts.Streams.Info(nextStepsMessage); err != nil {
		return err
	}
	return nil
}

const nextStepsMessageTemplate = `
Note: to complete application onboarding to the developer platform you have to first merge PR with configuration update for the tenant.
PR url: %s

After the PR is merged, you application is ready to be deployed to the IDP!
It will either happen with next commit or you can do it manually by triggering P2P workflow.
To do it, use GitHub web-interface or GitHub CLI.
Workflows link: %s
GitHub CLI commands:
  gh workflow list -R %s
  gh workflow run <workflow-id> -R %s
`

func createNewApp(
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *tenant.Tenant,
	fromTemplate *template.Spec,
	existingEnvs []environment.Environment,
) (application.CreateResult, error) {
	spinnerHandler := opts.Streams.Spinner("Creating new application...")
	defer spinnerHandler.Done()

	fastFeedbackEnvs := filterEnvs(cfg.P2P.FastFeedback.DefaultEnvs.Value, existingEnvs)
	extendedTestEnvs := filterEnvs(cfg.P2P.ExtendedTest.DefaultEnvs.Value, existingEnvs)
	prodEnvs := filterEnvs(cfg.P2P.Prod.DefaultEnvs.Value, existingEnvs)

	fulfilledTemplate := template.FulfilledTemplate{
		Spec:      fromTemplate,
		Arguments: []template.Argument{},
	}

	createResult, err := application.Create(
		application.CreateOp{
			Name:             opts.Name,
			OrgName:          cfg.GitHub.Organization.Value,
			LocalPath:        opts.LocalPath,
			Tenant:           appTenant,
			FastFeedbackEnvs: fastFeedbackEnvs,
			ExtendedTestEnvs: extendedTestEnvs,
			ProdEnvs:         prodEnvs,
			TemplatesPath:    cfg.Repositories.Templates.Value,
			Template:         &fulfilledTemplate,
		},
		githubClient,
	)
	return createResult, err
}

func createPRWithUpdatedReposListForTenant(
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *tenant.Tenant,
	createdAppResult application.CreateResult,
) (tenant.CreateOrUpdateResult, error) {
	spinnerHandler := opts.Streams.Spinner("Creating PR with new application for tenant...")
	defer spinnerHandler.Done()

	if err := appTenant.AddRepository(createdAppResult.RepositoryFullname.HttpUrl()); err != nil {
		return tenant.CreateOrUpdateResult{}, err
	}
	tenantUpdateResult, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			DPlatformRepoPath: cfg.Repositories.DPlatform.Value,
			BranchName:        string(appTenant.Name) + "-add-repo-" + createdAppResult.RepositoryFullname.Name,
			CommitMessage:     "Add new repository " + createdAppResult.RepositoryFullname.Name + " for tenant " + string(appTenant.Name),
			PRName:            "Add new repository " + createdAppResult.RepositoryFullname.Name + " for tenant " + string(appTenant.Name),
		},
		githubClient,
	)
	return tenantUpdateResult, err
}

func filterEnvs(nameFilter []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(nameFilter, string(env.Environment)) {
			result = append(result, env)
		}
	}
	return result
}
