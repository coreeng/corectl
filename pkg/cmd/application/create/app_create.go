package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"

	"github.com/coreeng/corectl/pkg/application"
	"github.com/coreeng/corectl/pkg/cmd/template/render"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/selector"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/confirmation"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/template"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
)

type AppCreateOpt struct {
	Name         string
	LocalPath    string
	FromTemplate string
	Tenant       string
	ArgsFile     string
	Args         []string
	DryRun       bool

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
				log.Panic().Err(err).Msg("could not get non-interactive flag")
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
	_, repoName, err := git.GetLocalRepoOrgAndName(filepath.Dir(opts.LocalPath))
	if err == nil && opts.Streams.IsInteractive() {
		confirmation, err := confirmation.GetInput(
			opts.Streams,
			fmt.Sprintf("Creating application %s in existing repo https://github.com/%s/%s, are you sure?",
				opts.Name,
				cfg.GitHub.Organization.Value,
				repoName,
			),
		)
		if err != nil {
			return fmt.Errorf("could not get confirmation from user: %w", err)
		}
		if !confirmation {
			return fmt.Errorf("aborted by user")
		}
	}
	var msg string
	if err != nil {
		// We are creating the new app in its own repo
		msg = fmt.Sprintf("Creating new application %s: https://github.com/%s/%s", opts.Name, cfg.GitHub.Organization.Value, opts.Name)
	} else {
		// We are creating the new app in a monorepo
		msg = fmt.Sprintf("Creating new application %s in existing repo: https://github.com/%s/%s", opts.Name, cfg.GitHub.Organization.Value, repoName)
	}
	wizard := opts.Streams.Wizard(msg, "")
	defer wizard.Done()

	opts.Streams.CurrentHandler.Info(fmt.Sprintf("resetting local repository [repo=%s]", cfg.Repositories.CPlatform.Value))
	if !opts.DryRun && !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, opts.DryRun); err != nil {
			return fmt.Errorf("failed to reset config repository state for CPlatform: %w", err)
		}
	}
	opts.Streams.CurrentHandler.Info(fmt.Sprintf("resetting local repository [repo=%s]", cfg.Repositories.Templates.Value))
	if !opts.DryRun && !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.Templates, opts.DryRun); err != nil {
			return fmt.Errorf("failed to reset config repository state for Templates: %w", err)
		}
	}

	existingTemplates, err := template.List(cfg.Repositories.Templates.Value)
	if err != nil {
		return err
	}
	templateInput := opts.createTemplateInput(existingTemplates)
	fromTemplate, err := templateInput.GetValue(opts.Streams)
	if fromTemplate != nil {
		opts.Streams.CurrentHandler.Info(fmt.Sprintf("template selected: %s", fromTemplate.Name))
	} else {
		opts.Streams.CurrentHandler.Info("no template selected")
	}

	if err != nil {
		return err
	}

	appTenant, err := selector.Tenant(cfg.Repositories.CPlatform.Value, opts.Tenant, opts.Streams)
	if err != nil {
		return err
	}
	opts.Streams.CurrentHandler.Info(fmt.Sprintf("tenant selected: %s", appTenant.Name))

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
	if createdAppResult.MonorepoMode {
		opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitle(fmt.Sprintf("added %s to repository: %s", opts.Name, createdAppResult.RepositoryFullname.HttpUrl()))
	} else {
		opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitle(fmt.Sprintf("created repository: %s", createdAppResult.RepositoryFullname.HttpUrl()))
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
		tenantUpdateResult, err := createPRWithUpdatedReposListForTenant(opts, cfg, githubClient, appTenant, createdAppResult)
		if err != nil {
			return err
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
	opts.Streams.CurrentHandler.Warn(strings.TrimSpace(nextStepsMessage))

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
	createOp := application.CreateOp{
		Name:             opts.Name,
		OrgName:          cfg.GitHub.Organization.Value,
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
	opts.Streams.CurrentHandler.SetTask(fmt.Sprintf(
		"Creating PR with new application %s for tenant %s in platform repo %s",
		opts.Name, opts.Tenant, cfg.Repositories.CPlatform.Value,
	), "")

	if err := appTenant.AddRepository(createdAppResult.RepositoryFullname.HttpUrl()); err != nil && errors.Is(err, coretnt.ErrRepositoryAlreadyPresent) {
		opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitleWithStatus(
			"Application is already registered for tenant. Skipping.",
			wizard.TaskStatusSkipped,
		)
		return nil, nil
	} else if err != nil {
		opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitleWithStatus(
			fmt.Sprintf("Failed to add application to tenant: %s", err),
			wizard.TaskStatusError,
		)
		return nil, err
	}
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	opts.Streams.CurrentHandler.Info(fmt.Sprintf("ensuring tenant repository exists: %s", createdAppResult.RepositoryFullname.Name()))

	tenantUpdateResult, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            appTenant,
			CplatformRepoPath: cfg.Repositories.CPlatform.Value,
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
		opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitleWithStatus(
			fmt.Sprintf("Failed to create PR for tenant to add a new application repository: %s", err),
			wizard.TaskStatusError,
		)
		return nil, err
	}
	opts.Streams.CurrentHandler.SetCurrentTaskCompletedTitle(fmt.Sprintf(
		"Created PR with new application %s for tenant %s: %s",
		opts.Name, appTenant.Name, tenantUpdateResult.PRUrl,
	))
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
