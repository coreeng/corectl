package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/application"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/selector"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/confirmation"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"
)

type AppCreateOpt struct {
	Name           string
	LocalPath      string
	FromTemplate   string
	Tenant         string
	GitHubRepoName string
	Description    string
	ArgsFile       string
	Args           []string
	DryRun         bool

	Streams userio.IOStreams
}

func NewAppCreateCmd(cfg *config.Config) (*cobra.Command, error) {
	var opts = AppCreateOpt{}
	var appCreateCmd = &cobra.Command{
		Use:   "create <app-name> [<local-path>]",
		Short: "Create new application",
		Long: `Creates new application:

- create a git repository locally and initializes it with application skeleton generated from a template
- template is rendered with arguments: {{name}}, {{tenant}}
- create a new github repository with p2p related variables
- create a new PR to environment repository with configuration update for the new application (if necessary)

NOTE:
- If <local-path> is not set, it defaults to ./<app-name>.
- If <local-path> is an existing git repository it will add a new application to it 
  without creating a new remote repository.
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			opts.Name = args[0]
			if len(args) > 1 {
				opts.LocalPath = args[1]
			} else {
				opts.LocalPath = "./" + opts.Name
			}

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				logger.Panic().With(zap.Error(err)).Msg("could not get non-interactive flag")
			}

			opts.Streams = userio.NewIOStreamsWithInteractive(
				os.Stdin,
				os.Stdout,
				os.Stderr,
				!nonInteractive,
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
	appCreateCmd.Flags().StringVarP(
		&opts.Tenant,
		"tenant",
		"",
		"",
		"Tenant to configure for P2P",
	)
	appCreateCmd.Flags().StringVar(
		&opts.GitHubRepoName,
		"github-repo",
		"",
		"GitHub repository name (defaults to app name)",
	)
	appCreateCmd.Flags().StringVar(
		&opts.Description,
		"description",
		"",
		"Description for the GitHub repository",
	)
	appCreateCmd.Flags().StringVar(
		&opts.ArgsFile,
		"args-file",
		"",
		"Path to YAML file containing template arguments",
	)
	appCreateCmd.Flags().StringSliceVarP(
		&opts.Args,
		"arg",
		"a",
		[]string{},
		"Template argument in the format: <arg-name>=<arg-value>",
	)

	appCreateCmd.Flags().BoolVarP(
		&opts.DryRun,
		"dry-run",
		"n",
		false,
		"Dry run",
	)

	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
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

	return appCreateCmd, nil
}

func run(opts *AppCreateOpt, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{
		cfg.Repositories.CPlatform,
		cfg.Repositories.Templates,
	}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	repoOrg, repoName, err := git.GetLocalRepoOrgAndName(filepath.Dir(opts.LocalPath))
	isMonorepo := err == nil
	if isMonorepo && opts.Streams.IsInteractive() {
		msg := fmt.Sprintf("Creating application %s in existing repo https://github.com/%s/%s, are you sure?", opts.Name, repoOrg, repoName)
		confirmation, err := confirmation.GetInput(opts.Streams, msg)
		if err != nil {
			return fmt.Errorf("could not get confirmation from user: %w", err)
		}
		if !confirmation {
			return fmt.Errorf("aborted by user")
		}
	}
	var msg string
	if isMonorepo {
		msg = fmt.Sprintf("Creating new application %s in existing repo: https://github.com/%s/%s", opts.Name, repoOrg, repoName)
	} else {
		githubRepoName := opts.GitHubRepoName
		if githubRepoName == "" {
			githubRepoName = opts.Name
		}
		msg = fmt.Sprintf("Creating new application %s: https://github.com/%s/%s", opts.Name, cfg.GitHub.Organization.Value, githubRepoName)
	}

	logger.Info().Msg(msg)

	existingTemplates, err := template.List(configpath.GetCorectlTemplatesDir())
	if err != nil {
		return err
	}
	templateInput := opts.createTemplateInput(existingTemplates)
	fromTemplate, err := templateInput.GetValue(opts.Streams)
	if fromTemplate != nil {
		logger.Info().Msgf("template selected: %s", fromTemplate.Name)
	} else {
		logger.Info().Msg("no template selected")
	}

	if err != nil {
		return err
	}

	selectedTenant, err := selector.Tenant(configpath.GetCorectlCPlatformDir("tenants"), opts.Tenant, opts.Streams)
	if err != nil {
		return err
	}
	logger.Info().Msgf("tenant selected: %s", selectedTenant.Name)

	// If the selected tenant is a team, create an app tenant under it
	var appTenant *coretnt.Tenant
	var teamTenant *coretnt.Tenant
	if selectedTenant.Kind == "team" {
		logger.Info().Msgf("Selected tenant '%s' is a team. Creating app tenant for application '%s'", selectedTenant.Name, opts.Name)
		teamTenant = selectedTenant
		appTenant, err = createAppTenantForTeam(opts, selectedTenant)
		if err != nil {
			return fmt.Errorf("failed to create app tenant: %w", err)
		}
		logger.Info().Msgf("Will create app tenant '%s' under team '%s'", appTenant.Name, selectedTenant.Name)
	} else {
		appTenant = selectedTenant
	}

	existingEnvs, err := environment.List(configpath.GetCorectlCPlatformDir("environments"))
	if err != nil {
		return err
	}

	if err = userio.ValidateFilePath(opts.LocalPath, userio.FileValidatorOptions{
		DirsOnly:   true,
		DirIsEmpty: true,
	}); err != nil {
		return err
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)

	newAppOrg := ""
	if isMonorepo {
		newAppOrg = repoOrg
	}
	createdAppResult, err := createNewApp(newAppOrg, opts, cfg, githubClient, appTenant, fromTemplate, existingEnvs)
	if err != nil {
		return err
	}
	if createdAppResult.MonorepoMode {
		logger.Warn().Msgf("added %s to repository: %s", opts.Name, createdAppResult.RepositoryFullname.HttpUrl())
	} else {
		logger.Warn().Msgf("created repository: %s", createdAppResult.RepositoryFullname.HttpUrl())
	}

	var nextStepsMessage string
	if createdAppResult.MonorepoMode {
		nextStepsMessage = fmt.Sprintf(
			nextStepsMessageTemplateMonoRepo,
			createdAppResult.PRUrl,
			createdAppResult.RepositoryFullname.ActionsHttpUrl(),
			createdAppResult.RepositoryFullname.String(),
			createdAppResult.RepositoryFullname.String(),
		)
	} else {
		var tenantUpdateResult *tenant.CreateOrUpdateResult

		// If we created an app tenant from a team, create a combined PR with both the tenant and the repo
		if teamTenant != nil {
			logger.Warn().Msgf("Creating PR with new app tenant '%s' and application %s for team %s in platform repo",
				appTenant.Name, opts.Name, teamTenant.Name)
			tenantUpdateResult, err = createPRWithNewTenantAndRepo(opts, cfg, githubClient, appTenant, teamTenant, createdAppResult)
			if err != nil {
				return err
			}
			logger.Warn().Msgf("Created PR with new app tenant and application: %s", tenantUpdateResult.PRUrl)
		} else {
			tenantUpdateResult, err = createPRWithUpdatedReposListForTenant(opts, cfg, githubClient, appTenant, createdAppResult)
			if err != nil {
				return err
			}
		}

		if tenantUpdateResult != nil {
			nextStepsMessage = fmt.Sprintf(
				nextStepsMessageTemplateSingleRepo,
				tenantUpdateResult.PRUrl,
				createdAppResult.RepositoryFullname.ActionsHttpUrl(),
				createdAppResult.RepositoryFullname.String(),
				createdAppResult.RepositoryFullname.String(),
			)
		} else {
			nextStepsMessage = fmt.Sprintf(
				nextStepsMessageTemplateSingleRepoSkippedTenant,
				createdAppResult.RepositoryFullname.ActionsHttpUrl(),
				createdAppResult.RepositoryFullname.String(),
				createdAppResult.RepositoryFullname.String(),
			)
		}
	}
	logger.Warn().Msg(strings.TrimSpace(nextStepsMessage))

	return nil
}

const (
	nextStepsMessageTemplateSingleRepo = `
To complete application onboarding to the Core Platform you have to first merge PR with configuration update for the tenant.
PR url: %s

After the PR is merged, your application is ready to be deployed to the Core Platform!
It will either happen with next commit or you can do it manually by triggering P2P workflow.
To do it, use GitHub web-interface or GitHub CLI.
Workflows link: %s
GitHub CLI commands:
  gh workflow list -R %s
  gh workflow run <workflow-id> -R %s
`
	nextStepsMessageTemplateSingleRepoSkippedTenant = `
Your application is ready to be deployed to the Core Platform!
It will either happen with next commit or you can do it manually by triggering P2P workflow.
To do it, use GitHub web-interface or GitHub CLI.
Workflows link: %s
GitHub CLI commands:
  gh workflow list -R %s
  gh workflow run <workflow-id> -R %s
`
	nextStepsMessageTemplateMonoRepo = `
To complete application onboarding to the Core Platform you have to first merge PR that adds this application to your repository.
PR url: %s

After the PR is merged, your application is ready to be deployed to the Core Platform!
It will either happen with next commit or you can do it manually by triggering P2P workflow.
To do it, use GitHub web-interface or GitHub CLI.
Workflows link: %s
GitHub CLI commands:
  gh workflow list -R %s
  gh workflow run <workflow-id> -R %s
`
)

func createNewApp(
	newAppOrg string, // if new app is in a monorepo, `newAppOrg` is the github org of the monorepo; otherwise ""
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *coretnt.Tenant,
	fromTemplate *template.Spec,
	existingEnvs []environment.Environment,
) (application.CreateResult, error) {
	fastFeedbackEnvs := filterEnvsByNames(cfg.P2P.FastFeedback.DefaultEnvs.Value, existingEnvs)
	extendedTestEnvs := filterEnvsByNames(cfg.P2P.ExtendedTest.DefaultEnvs.Value, existingEnvs)
	prodEnvs := filterEnvsByNames(cfg.P2P.Prod.DefaultEnvs.Value, existingEnvs)

	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	templateRenderer := &render.FlagsAwareTemplateRenderer{
		ArgsFile: opts.ArgsFile,
		Args:     opts.Args,
		Streams:  opts.Streams,
	}
	service := application.NewService(templateRenderer, githubClient, opts.DryRun)
	if newAppOrg == "" {
		newAppOrg = cfg.GitHub.Organization.Value
	}
	createOp := application.CreateOp{
		Name:             opts.Name,
		GitHubRepoName:   opts.GitHubRepoName,
		Description:      opts.Description,
		OrgName:          newAppOrg,
		LocalPath:        opts.LocalPath,
		Tenant:           appTenant,
		FastFeedbackEnvs: fastFeedbackEnvs,
		ExtendedTestEnvs: extendedTestEnvs,
		ProdEnvs:         prodEnvs,
		Template:         fromTemplate,
		GitAuth:          gitAuth,
	}
	if err := service.ValidateCreate(createOp); err != nil {
		return application.CreateResult{}, err
	}
	createResult, err := service.Create(createOp)
	return createResult, err
}

func filterEnvsByNames(names []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(names, env.Environment) {
			result = append(result, env)
		}
	}
	return result
}

func createPRWithUpdatedReposListForTenant(
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *coretnt.Tenant,
	createdAppResult application.CreateResult,
) (*tenant.CreateOrUpdateResult, error) {
	logger.Warn().Msgf("Creating PR with new application %s for tenant %s in platform repo",
		opts.Name, appTenant.Name)

	if err := appTenant.AddRepository(createdAppResult.RepositoryFullname.HttpUrl()); err != nil && errors.Is(err, coretnt.ErrRepositoryAlreadyPresent) {
		logger.Warn().Msgf("Application is already registered for tenant. Skipping.")
		return nil, nil
	} else if err != nil {
		logger.Error().Msgf("Failed to add application to tenant: %s", err)
		return nil, err
	}
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	logger.Warn().Msgf("ensuring tenant repository exists: %s", createdAppResult.RepositoryFullname.Name())

	tenantUpdateResult, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			CplatformRepoPath: configpath.GetCorectlCPlatformDir(),
			BranchName:        fmt.Sprintf("%s-add-repo-%s", appTenant.Name, createdAppResult.RepositoryFullname.Name()),
			CommitMessage:     fmt.Sprintf("Add new repository %s for tenant %s", createdAppResult.RepositoryFullname.Name(), appTenant.Name),
			PRName:            fmt.Sprintf("Add new repository %s for tenant %s", createdAppResult.RepositoryFullname.Name(), appTenant.Name),
			PRBody:            fmt.Sprintf("Adding repository for new app %s (%s) to tenant '%s'", opts.Name, createdAppResult.RepositoryFullname.HttpUrl(), appTenant.Name),
			GitAuth:           gitAuth,
			DryRun:            opts.DryRun,
		},
		githubClient,
	)
	if err != nil {
		logger.Error().Msgf("Failed to create PR for tenant to add a new application repository: %s", err)

		return nil, err
	}
	logger.Warn().Msgf("Created PR with new application %s for tenant %s: %s",
		opts.Name, appTenant.Name, tenantUpdateResult.PRUrl)

	return &tenantUpdateResult, nil
}

func (opts *AppCreateOpt) createTemplateInput(existingTemplates []template.Spec) userio.InputSourceSwitch[string, *template.Spec] {
	availableTemplateNames := make([]string, len(existingTemplates)+1)
	availableTemplateNames[0] = "<empty>"
	for i, t := range existingTemplates {
		availableTemplateNames[i+1] = t.Name
	}
	return userio.InputSourceSwitch[string, *template.Spec]{
		DefaultValue: userio.AsZeroable(opts.FromTemplate),
		Optional:     true,
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Template:",
				Items:  availableTemplateNames,
			}, nil
		},
		ValidateAndMap: func(inp string) (*template.Spec, error) {
			inp = strings.TrimSpace(inp)
			if inp == "<empty>" || inp == "" {
				return nil, nil
			}
			templateIndex := slices.IndexFunc(existingTemplates, func(spec template.Spec) bool {
				return spec.Name == inp
			})
			if templateIndex < 0 {
				return nil, errors.New("unknown template")
			}
			return &existingTemplates[templateIndex], nil
		},
		ErrMessage: "invalid template",
	}
}

func createAppTenantForTeam(
	opts *AppCreateOpt,
	teamTenant *coretnt.Tenant,
) (*coretnt.Tenant, error) {
	// Generate a unique name for the app tenant
	appTenantName := opts.Name

	// Check if the app tenant already exists
	tenantsPath := configpath.GetCorectlCPlatformDir("tenants")
	existingTenants, err := coretnt.List(tenantsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing tenants: %w", err)
	}

	// Check for name collision
	for _, t := range existingTenants {
		if t.Name == appTenantName {
			return nil, fmt.Errorf("tenant with name '%s' already exists", appTenantName)
		}
	}

	// Create the app tenant inheriting properties from the team
	appTenant := &coretnt.Tenant{
		Name:          appTenantName,
		Kind:          "app",
		Parent:        teamTenant.Name,
		Description:   opts.Name,
		ContactEmail:  teamTenant.ContactEmail,
		Environments:  teamTenant.Environments,
		Repos:         []string{}, // Will be populated after the repo is created
		AdminGroup:    teamTenant.AdminGroup,
		ReadOnlyGroup: teamTenant.ReadOnlyGroup,
		CloudAccess:   make([]coretnt.CloudAccess, 0),
	}

	// Validate the tenant
	tenantMap := map[string]*coretnt.Tenant{
		appTenant.Name: appTenant,
	}
	for _, t := range existingTenants {
		tenantMap[t.Name] = &t
	}

	validationResult := coretnt.ValidateTenants(tenantMap)
	for _, warn := range validationResult.Warnings {
		var tenantRelatedWarn coretnt.TenantRelatedError
		if errors.As(warn, &tenantRelatedWarn) && tenantRelatedWarn.IsRelatedToTenant(appTenant) {
			logger.Error().Msg(warn.Error())
		}
	}
	var tenantRelatedErr coretnt.TenantRelatedError
	if len(validationResult.Errors) > 0 &&
		errors.As(validationResult.Errors[0], &tenantRelatedErr) &&
		tenantRelatedErr.IsRelatedToTenant(appTenant) {
		return nil, tenantRelatedErr
	}

	return appTenant, nil
}

func createPRWithNewTenantAndRepo(
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *coretnt.Tenant,
	teamTenant *coretnt.Tenant,
	createdAppResult application.CreateResult,
) (*tenant.CreateOrUpdateResult, error) {
	// Add the repository to the tenant before creating the PR
	if err := appTenant.AddRepository(createdAppResult.RepositoryFullname.HttpUrl()); err != nil {
		return nil, fmt.Errorf("failed to add repository to tenant: %w", err)
	}

	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)

	result, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			ParentTenant:      teamTenant,
			CplatformRepoPath: configpath.GetCorectlCPlatformDir(),
			BranchName:        fmt.Sprintf("new-app-tenant-%s", appTenant.Name),
			CommitMessage:     fmt.Sprintf("Add new app tenant: %s", appTenant.Name),
			PRName:            fmt.Sprintf("New app tenant: %s", appTenant.Name),
			PRBody:            fmt.Sprintf("Adds new app tenant '%s'", appTenant.Name),
			GitAuth:           gitAuth,
			DryRun:            opts.DryRun,
		},
		githubClient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR for new app tenant and repo: %w", err)
	}

	return &result, nil
}
