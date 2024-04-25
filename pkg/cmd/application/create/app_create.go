package create

import (
	"cmp"
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/application"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"slices"
	"strings"
)

type AppCreateOpt struct {
	Name           string
	LocalPath      string
	NonInteractive bool
	FromTemplate   string
	Tenant         string

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
	appCreateCmd.Flags().StringVarP(
		&opts.Tenant,
		"tenant",
		"",
		"",
		"Tenant to configure for P2P",
	)
	appCreateCmd.Flags().BoolVar(
		&opts.NonInteractive,
		"nonint",
		false,
		"Disable interactive inputs",
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
		&cfg.Repositories.CPlatform,
		appCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.Templates,
		appCreateCmd.Flags(),
	)

	return appCreateCmd, nil
}

func run(opts *AppCreateOpt, cfg *config.Config) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates); err != nil {
		return err
	}

	existingTemplates, err := template.List(cfg.Repositories.Templates.Value)
	if err != nil {
		return err
	}
	templateInput := opts.createTemplateInput(existingTemplates)
	fromTemplate, err := templateInput.GetValue(opts.Streams)
	if err != nil {
		return err
	}

	existingTenants, err := coretnt.List(coretnt.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}
	defaultTenant := cfg.Tenant.Value
	tenantInput := opts.createTenantInput(existingTenants, defaultTenant)
	appTenant, err := tenantInput.GetValue(opts.Streams)
	if err != nil {
		return err
	}

	existingEnvs, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
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

	createdAppResult, err := createNewApp(opts, cfg, githubClient, appTenant, fromTemplate, existingEnvs)
	if err != nil {
		return err
	}
	opts.Streams.Info("Created repository: ", createdAppResult.RepositoryFullname.HttpUrl())

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
	opts.Streams.Info(nextStepsMessage)
	return nil
}

const nextStepsMessageTemplate = `
Note: to complete application onboarding to the Core Platform you have to first merge PR with configuration update for the tenant.
PR url: %s

After the PR is merged, you application is ready to be deployed to the Core Platform!
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
	appTenant *coretnt.Tenant,
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

	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	appCreateOp := application.CreateOp{
		Name:             opts.Name,
		OrgName:          cfg.GitHub.Organization.Value,
		LocalPath:        opts.LocalPath,
		Tenant:           appTenant,
		FastFeedbackEnvs: fastFeedbackEnvs,
		ExtendedTestEnvs: extendedTestEnvs,
		ProdEnvs:         prodEnvs,
		Template:         &fulfilledTemplate,
		GitAuth:          gitAuth,
	}
	if err := application.ValidateCreate(appCreateOp, githubClient); err != nil {
		return application.CreateResult{}, err
	}
	createResult, err := application.Create(
		appCreateOp,
		githubClient,
	)
	return createResult, err
}

func filterEnvs(nameFilter []string, envs []environment.Environment) []environment.Environment {
	var result []environment.Environment
	for _, env := range envs {
		if slices.Contains(nameFilter, env.Environment) {
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
) (tenant.CreateOrUpdateResult, error) {
	spinnerHandler := opts.Streams.Spinner("Creating PR with new application for tenant...")
	defer spinnerHandler.Done()

	if err := appTenant.AddRepository(createdAppResult.RepositoryFullname.HttpUrl()); err != nil {
		return tenant.CreateOrUpdateResult{}, err
	}
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	tenantUpdateResult, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			CplatformRepoPath: cfg.Repositories.CPlatform.Value,
			BranchName:        fmt.Sprintf("%s-add-repo-%s", appTenant.Name, createdAppResult.RepositoryFullname.Name()),
			CommitMessage:     fmt.Sprintf("Add new repository %s for tenant %s", createdAppResult.RepositoryFullname.Name(), appTenant.Name),
			PRName:            fmt.Sprintf("Add new repository %s for tenant %s", createdAppResult.RepositoryFullname.Name(), appTenant.Name),
			PRBody:            fmt.Sprintf("Adding repository for new app %s (%s) to tenant '%s'", opts.Name, createdAppResult.RepositoryFullname.HttpUrl(), appTenant.Name),
			GitAuth:           gitAuth,
		},
		githubClient,
	)
	return tenantUpdateResult, err
}

func (opts *AppCreateOpt) createTenantInput(existingTenant []coretnt.Tenant, defaultTenantName string) userio.InputSourceSwitch[string, *coretnt.Tenant] {
	availableTenantNames := make([]string, len(existingTenant)+1)
	availableTenantNames[0] = coretnt.RootName
	for i, t := range existingTenant {
		availableTenantNames[i+1] = t.Name
	}
	return userio.InputSourceSwitch[string, *coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(cmp.Or(opts.Tenant, defaultTenantName)),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt:          fmt.Sprintf("Tenant (default is '%s'):", defaultTenantName),
				Items:           availableTenantNames,
				PreselectedItem: defaultTenantName,
			}, nil
		},
		ValidateAndMap: func(inp string) (*coretnt.Tenant, error) {
			inpName := strings.TrimSpace(inp)
			tenantIndex := slices.IndexFunc(existingTenant, func(t coretnt.Tenant) bool {
				return t.Name == inpName
			})
			if tenantIndex < 0 {
				return nil, errors.New("unknown tenant")
			}
			return &existingTenant[tenantIndex], nil
		},
		ErrMessage: "invalid tenant",
	}
}

func (opts *AppCreateOpt) createTemplateInput(existingTemplates []template.Spec) userio.InputSourceSwitch[string, *template.Spec] {
	availableTemplateNames := make([]string, len(existingTemplates))
	for i, t := range existingTemplates {
		availableTemplateNames[i] = t.Name
	}
	return userio.InputSourceSwitch[string, *template.Spec]{
		DefaultValue: userio.AsZeroable(opts.FromTemplate),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Template:",
				Items:  availableTemplateNames,
			}, nil
		},
		ValidateAndMap: func(inp string) (*template.Spec, error) {
			inp = strings.TrimSpace(inp)
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
