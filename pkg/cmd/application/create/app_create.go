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
	Prefix         string
	ArgsFile       string
	Args           []string
	Config         string
	DryRun         bool
	Public         bool

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
		"Org unit that will own the new delivery unit",
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
		&opts.Prefix,
		"prefix",
		"",
		"Optional dashboard-only hierarchy prefix for the delivery unit created by this command",
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
	appCreateCmd.Flags().StringVarP(
		&opts.Config,
		"config",
		"c",
		"",
		"JSON configuration object to pass to the template",
	)

	appCreateCmd.Flags().BoolVarP(
		&opts.DryRun,
		"dry-run",
		"n",
		false,
		"Dry run",
	)

	appCreateCmd.Flags().BoolVar(
		&opts.Public,
		"public",
		false,
		"Create a public GitHub repository (default is private)",
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
	// Default description to app name if not set
	if opts.Description == "" {
		opts.Description = opts.Name
	}

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

	duType, err := deliveryUnitTypeFromTemplate(fromTemplate)
	if err != nil {
		return err
	}

	ownerOrgUnit, err := selector.OrgUnit(configpath.GetCorectlCPlatformDir("tenants"), opts.Tenant, opts.Streams)
	if err != nil {
		return err
	}
	logger.Info().Msgf("org unit selected: %s", ownerOrgUnit.Name)

	appTenant, err := createDeliveryUnitForOrgUnit(opts, ownerOrgUnit, duType)
	if err != nil {
		return fmt.Errorf("failed to create delivery unit: %w", err)
	}
	logger.Info().Msgf("Will create delivery unit '%s' owned by org unit '%s'", appTenant.Name, ownerOrgUnit.Name)

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
		logger.Warn().Msgf("Creating PR with new delivery unit '%s' for org unit %s in platform repo (monorepo mode)",
			appTenant.Name, ownerOrgUnit.Name)
		tenantUpdateResult, err := createPRWithNewTenantAndRepo(opts, cfg, githubClient, appTenant, ownerOrgUnit, createdAppResult)
		if err != nil {
			return err
		}
		logger.Warn().Msgf("Created PR with new delivery unit: %s", tenantUpdateResult.PRUrl)

		nextStepsMessage = fmt.Sprintf(
			nextStepsMessageTemplateMonoRepo,
			createdAppResult.PRUrl,
			createdAppResult.RepositoryFullname.ActionsHttpUrl(),
			createdAppResult.RepositoryFullname.String(),
			createdAppResult.RepositoryFullname.String(),
		)
	} else {
		logger.Warn().Msgf("Creating PR with new delivery unit '%s' and application %s for org unit %s in platform repo",
			appTenant.Name, opts.Name, ownerOrgUnit.Name)
		tenantUpdateResult, err := createPRWithNewTenantAndRepo(opts, cfg, githubClient, appTenant, ownerOrgUnit, createdAppResult)
		if err != nil {
			return err
		}
		logger.Warn().Msgf("Created PR with new delivery unit and application: %s", tenantUpdateResult.PRUrl)

		nextStepsMessage = fmt.Sprintf(
			nextStepsMessageTemplateSingleRepo,
			tenantUpdateResult.PRUrl,
			createdAppResult.RepositoryFullname.ActionsHttpUrl(),
			createdAppResult.RepositoryFullname.String(),
			createdAppResult.RepositoryFullname.String(),
		)
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
		Config:           opts.Config,
		Public:           opts.Public,
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

func deliveryUnitTypeFromTemplate(t *template.Spec) (string, error) {
	// Map software template kind to core-platform delivery unit type.
	if t == nil {
		return "application", nil
	}
	k := strings.ToLower(strings.TrimSpace(t.Kind))
	if k == "" {
		k = "app"
	}
	switch k {
	case "app":
		return "application", nil
	case "infra":
		return "infrastructure", nil
	default:
		return "", fmt.Errorf("unknown template kind: %s (expected 'app' or 'infra')", t.Kind)
	}
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

func createDeliveryUnitForOrgUnit(
	opts *AppCreateOpt,
	orgUnit *coretnt.Tenant,
	duType string,
) (*coretnt.Tenant, error) {
	duName := opts.Name
	if orgUnit == nil {
		return nil, fmt.Errorf("org unit is required")
	}
	if orgUnit.Kind != "OrgUnit" {
		return nil, fmt.Errorf("selected tenant '%s' is not an org unit", orgUnit.Name)
	}

	tenantsPath := configpath.GetCorectlCPlatformDir("tenants")
	existingTenants, err := coretnt.List(tenantsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing tenants: %w", err)
	}
	for _, t := range existingTenants {
		if t.Name == duName {
			return nil, fmt.Errorf("tenant with name '%s' already exists", duName)
		}
	}

	du := &coretnt.Tenant{
		Name:          duName,
		Kind:          "DeliveryUnit",
		Type:          duType,
		Owner:         orgUnit.Name,
		Prefix:        strings.TrimSpace(opts.Prefix),
		Description:   opts.Description,
		ContactEmail:  orgUnit.ContactEmail,
		Environments:  orgUnit.Environments,
		Repo:          "",
		AdminGroup:    orgUnit.AdminGroup,
		ReadOnlyGroup: orgUnit.ReadOnlyGroup,
		CloudAccess:   make([]coretnt.CloudAccess, 0),
	}

	// Validate the tenant
	tenantMap := map[string]*coretnt.Tenant{
		du.Name: du,
	}
	addExistingTenants(tenantMap, existingTenants)

	validationResult := coretnt.ValidateTenants(tenantMap)
	for _, warn := range validationResult.Warnings {
		var tenantRelatedWarn coretnt.TenantRelatedError
		if errors.As(warn, &tenantRelatedWarn) && tenantRelatedWarn.IsRelatedToTenant(du) {
			logger.Error().Msg(warn.Error())
		}
	}
	var tenantRelatedErr coretnt.TenantRelatedError
	if len(validationResult.Errors) > 0 &&
		errors.As(validationResult.Errors[0], &tenantRelatedErr) &&
		tenantRelatedErr.IsRelatedToTenant(du) {
		return nil, tenantRelatedErr
	}

	return du, nil
}

func createPRWithNewTenantAndRepo(
	opts *AppCreateOpt,
	cfg *config.Config,
	githubClient *github.Client,
	appTenant *coretnt.Tenant,
	ownerOrgUnit *coretnt.Tenant,
	createdAppResult application.CreateResult,
) (*tenant.CreateOrUpdateResult, error) {
	appTenant.Repo = createdAppResult.RepositoryFullname.HttpUrl()

	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)

	result, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			OwnerTenant:       ownerOrgUnit,
			CplatformRepoPath: configpath.GetCorectlCPlatformDir(),
			BranchName:        fmt.Sprintf("new-du-tenant-%s", appTenant.Name),
			CommitMessage:     fmt.Sprintf("Add new delivery unit: %s", appTenant.Name),
			PRName:            fmt.Sprintf("New delivery unit: %s", appTenant.Name),
			PRBody:            fmt.Sprintf("Adds new delivery unit '%s'", appTenant.Name),
			GitAuth:           gitAuth,
			DryRun:            opts.DryRun,
		},
		githubClient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR for new delivery unit and repo: %w", err)
	}

	return &result, nil
}

func addExistingTenants(tenantMap map[string]*coretnt.Tenant, tenants []coretnt.Tenant) {
	for i := range tenants {
		tenantMap[tenants[i].Name] = &tenants[i]
	}
}
