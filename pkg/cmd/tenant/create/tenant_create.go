package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/phuslu/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type TenantCreateOpt struct {
	Name           string
	Parent         string
	Description    string
	ContactEmail   string
	CostCentre     string
	Environments   []string
	Repositories   []string
	AdminGroup     string
	ReadOnlyGroup  string
	NonInteractive bool
	DryRun         bool

	Streams userio.IOStreams
}

func NewTenantCreateCmd(cfg *config.Config) *cobra.Command {
	opt := TenantCreateOpt{}
	tenantCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				log.Panic().Err(err).Msg("could not get non-interactive flag")
			}
			opt.NonInteractive = nonInteractive

			opt.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
				!opt.NonInteractive,
			)
			return run(&opt, cfg)
		},
	}

	tenantCreateCmd.Flags().StringVar(
		&opt.Name,
		"name",
		"",
		"Tenant name. Should be valid K8S label.",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.Parent,
		"parent",
		"",
		"Parent tenant name",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.Description,
		"description",
		"",
		"Description for tenant",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.ContactEmail,
		"contact-email",
		"",
		"Contact email for tenant",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.CostCentre,
		"cost-centre",
		"",
		"Cost centre of tenant. Should be valid K8S label.",
	)
	tenantCreateCmd.Flags().StringSliceVar(
		&opt.Environments,
		"environments",
		nil,
		"Environments, available to tenant",
	)
	tenantCreateCmd.Flags().StringSliceVar(
		&opt.Repositories,
		"repositories",
		nil,
		"Repositories, tenant is responsible for.",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.AdminGroup,
		"admin-group",
		"",
		"Admin group for tenant",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.ReadOnlyGroup,
		"readonly-group",
		"",
		"Readonly group for tenant",
	)

	tenantCreateCmd.Flags().BoolVarP(
		&opt.DryRun,
		"dry-run",
		"n",
		false,
		"Dry run",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		tenantCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Token,
		tenantCreateCmd.Flags(),
	)
	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		tenantCreateCmd.Flags(),
	)

	return tenantCreateCmd
}

func run(opt *TenantCreateOpt, cfg *config.Config) error {
	if !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, false); err != nil {
			return err
		}
	}

	tenantsPath := coretnt.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value)
	existingTenants, err := coretnt.List(tenantsPath)
	if err != nil {
		return err
	}
	rootTenant := coretnt.RootTenant(tenantsPath)

	envsDir := environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value)
	envFilePath := filepath.Join(envsDir, "environments.yaml")
	envs, err := listEnabledEnvironments(envFilePath)
	if err != nil {
		log.Warn().Msgf("Failed to read environments file '%s': %s. Falling back to listing directories in '%s'.", envFilePath, err, envsDir)
		envs, err = environment.List(envsDir)
		if err != nil {
			return err
		}
	}
	if len(envs) == 0 {
		return fmt.Errorf("no enabled environment found in '%s' and no environment directory found in '%s'", envFilePath, envsDir)
	}

	nameInput := opt.createNameInputSwitch(existingTenants)
	parentInput := opt.createParentInputSwitch(rootTenant, existingTenants)
	descriptionInput := opt.createDescriptionInputSwitch()
	contactEmailInput := opt.createContactEmailInputSwitch()
	costCentreInput := opt.createCostCentreInputSwitch()
	envsInput := opt.createEnvironmentsInputSwitch(envs)
	repositoriesInput := opt.createRepositoriesInputSwitch()
	adminGroupInput := opt.createAdminGroupInputSwitch()
	readOnlyGroupInput := opt.createReadOnlyGroupInputSwitch()

	name, err := nameInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	parent, err := parentInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	description, err := descriptionInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	contactEmail, err := contactEmailInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	costCentre, err := costCentreInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	tenantEnvironments, err := envsInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	repositories, err := repositoriesInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	adminGroup, err := adminGroupInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	readOnlyGroup, err := readOnlyGroupInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}

	t := coretnt.Tenant{
		Name:          name,
		Parent:        parent.Name,
		Description:   description,
		ContactEmail:  contactEmail,
		CostCentre:    costCentre,
		Environments:  tenantEnvironments,
		Repos:         repositories,
		AdminGroup:    adminGroup,
		ReadOnlyGroup: readOnlyGroup,
		CloudAccess:   make([]coretnt.CloudAccess, 0),
	}

	_, err = createTenant(opt.Streams, opt.DryRun, cfg, &t, &parent, existingTenants)
	if err != nil {
		return err
	}
	return nil
}

func createTenant(
	streams userio.IOStreams,
	dryRun bool,
	cfg *config.Config,
	t *coretnt.Tenant,
	parentTenant *coretnt.Tenant,
	allTenants []coretnt.Tenant,
) (tenant.CreateOrUpdateResult, error) {
	wizardHandler := streams.Wizard(
		fmt.Sprintf("Creating tenant %s in platform repository: %s", t.Name, cfg.Repositories.CPlatform.Value),
		"", // We don't know the PR URL yet, using SetCurrentTaskCompletedTitle
	)
	defer wizardHandler.Done()

	tenantMap := map[string]*coretnt.Tenant{
		t.Name: t,
	}
	for _, tenant := range allTenants {
		tenantMap[tenant.Name] = &tenant
	}

	if err := validateTenant(tenantMap, t, wizardHandler); err != nil {
		wizardHandler.SetCurrentTaskCompletedTitleWithStatus(
			fmt.Sprintf("Unable to create such a tenant: %s", err),
			wizard.TaskStatusError,
		)
		return tenant.CreateOrUpdateResult{}, err
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	result, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            t,
			ParentTenant:      parentTenant,
			CplatformRepoPath: cfg.Repositories.CPlatform.Value,
			BranchName:        fmt.Sprintf("new-tenant-%s", t.Name),
			CommitMessage:     fmt.Sprintf("Add new tenant: %s", t.Name),
			PRName:            fmt.Sprintf("New tenant: %s", t.Name),
			PRBody:            fmt.Sprintf("Adds new tenant '%s'", t.Name),
			GitAuth:           gitAuth,
			DryRun:            dryRun,
		}, githubClient,
	)
	if err != nil {
		wizardHandler.SetCurrentTaskCompletedTitleWithStatus(fmt.Sprintf("Failed to create a PR for new tenant: %s", err), wizard.TaskStatusError)
	} else {
		wizardHandler.SetCurrentTaskCompletedTitle(fmt.Sprintf("Created PR for new tenant %s: %s", t.Name, result.PRUrl))
	}
	return result, err
}

func validateTenant(tenantMap map[string]*coretnt.Tenant, t *coretnt.Tenant, wizardHandler wizard.Handler) error {
	validationResult := coretnt.ValidateTenants(tenantMap)
	for _, warn := range validationResult.Warnings {
		var tenantRelatedWarn coretnt.TenantRelatedError
		if errors.As(warn, &tenantRelatedWarn) && tenantRelatedWarn.IsRelatedToTenant(t) {
			wizardHandler.Warn(warn.Error())
		}
	}
	var tenantRelatedErr coretnt.TenantRelatedError
	if len(validationResult.Errors) > 0 &&
		errors.As(validationResult.Errors[0], &tenantRelatedErr) &&
		tenantRelatedErr.IsRelatedToTenant(t) {
		return tenantRelatedErr
	}
	return nil
}

func (opt *TenantCreateOpt) createNameInputSwitch(existingTenants []coretnt.Tenant) userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			Name: inp,
		}
		if err := t.ValidateField("Name"); err != nil {
			return "", err
		}

		existingTenantI := slices.IndexFunc(existingTenants, func(tnt coretnt.Tenant) bool {
			return tnt.Name == inp
		})
		if existingTenantI >= 0 || inp == coretnt.RootName {
			return "", errors.New("tenant already exists")
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.Name),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:      "Tenant name (valid K8S namespace name):",
				Placeholder: "tenant-name",
				ValidateAndMap: func(inp string) (string, error) {
					name, err := validateFn(inp)
					return name, err
				},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid tenant name",
	}
}

func (opt *TenantCreateOpt) createParentInputSwitch(rootTenant *coretnt.Tenant, existingTenants []coretnt.Tenant) userio.InputSourceSwitch[string, coretnt.Tenant] {
	existingTenants = append(existingTenants, coretnt.Tenant{Name: coretnt.RootName})
	node, err := tenant.GetTenantTree(existingTenants, coretnt.RootName)
	if err != nil {
		panic(fmt.Sprintf("Failed to build tree of tenants: %s", err))
	}
	items, lines := tenant.RenderTenantTree(node)

	return userio.InputSourceSwitch[string, coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(opt.Parent),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt:         "Parent tenant:",
				Items:          items,
				DisplayedItems: lines,
			}, nil
		},
		ValidateAndMap: func(inp string) (coretnt.Tenant, error) {
			inp = strings.TrimSpace(inp)
			if inp == rootTenant.Name {
				return *rootTenant, nil
			}

			tenantIndx := slices.IndexFunc(existingTenants, func(t coretnt.Tenant) bool {
				return t.Name == inp
			})
			if tenantIndx >= 0 {
				return existingTenants[tenantIndx], nil
			}

			return coretnt.Tenant{}, errors.New("unknown tenant")
		},
		ErrMessage: "invalid parent tenant",
	}
}

func (opt *TenantCreateOpt) createDescriptionInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			Description: inp,
		}
		err := t.ValidateField("Description")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.Description),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Description:",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid tenant description",
	}
}

func (opt *TenantCreateOpt) createContactEmailInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			ContactEmail: inp,
		}
		err := t.ValidateField("ContactEmail")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.ContactEmail),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Contact email:",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid contact email",
	}
}

func (opt *TenantCreateOpt) createCostCentreInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			CostCentre: inp,
		}
		err := t.ValidateField("CostCentre")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.CostCentre),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Cost centre (valid K8S label value):",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid cost centre",
	}
}

func (opt *TenantCreateOpt) createEnvironmentsInputSwitch(envs []environment.Environment) userio.InputSourceSwitch[[]string, []string] {
	var envNames []string
	for _, env := range envs {
		envNames = append(envNames, env.Environment)
	}

	validateFn := func(inp []string) ([]string, error) {
		envs := []string{}
		for _, env := range inp {
			env = strings.TrimSpace(env)
			if env == "" {
				continue
			}
			if !slices.Contains(envNames, env) {
				return nil, fmt.Errorf("unknown environment: %s", env)
			}
			envs = append(envs, env)
		}
		if len(envs) == 0 {
			return nil, fmt.Errorf("at least one environment must be selected")
		}
		return envs, nil
	}

	return userio.InputSourceSwitch[[]string, []string]{
		DefaultValue: userio.AsZeroableSlice(opt.Environments),
		Optional:     true,
		InteractivePromptFn: func() (userio.InputPrompt[[]string], error) {
			return &userio.MultiSelect{
				Prompt: "Environments ('space' to select, 'enter' to validate):",
				Items:  envNames,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid environment list",
	}
}

func (opt *TenantCreateOpt) createRepositoriesInputSwitch() userio.InputSourceSwitch[[]string, []string] {
	validateFn := func(repos []string) ([]string, error) {
		var filteredRepos []string
		for _, repo := range repos {
			repo = strings.TrimSpace(repo)
			if repo == "" {
				continue
			}
			filteredRepos = append(filteredRepos, repo)
		}
		t := &coretnt.Tenant{
			Repos: filteredRepos,
		}
		err := t.ValidateField("Repos")
		if err != nil {
			return nil, err
		}
		return filteredRepos, nil

	}
	return userio.InputSourceSwitch[[]string, []string]{
		DefaultValue: userio.AsZeroableSlice(opt.Repositories),
		Optional:     true,
		InteractivePromptFn: func() (userio.InputPrompt[[]string], error) {
			return &userio.TextInput[[]string]{
				Prompt: "Repositories (comma separated GitHub links):",
				ValidateAndMap: func(inp string) ([]string, error) {
					return validateFn(strings.Split(inp, ","))
				},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid repositories list",
	}
}

func (opt *TenantCreateOpt) createAdminGroupInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			AdminGroup: inp,
		}
		err := t.ValidateField("AdminGroup")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.AdminGroup),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Admin group:",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid admin group",
	}
}

func (opt *TenantCreateOpt) createReadOnlyGroupInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		t := &coretnt.Tenant{
			ReadOnlyGroup: inp,
		}
		err := t.ValidateField("ReadOnlyGroup")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.ReadOnlyGroup),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Read only group:",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid read only group",
	}
}

type envFile struct {
	Enabled []string `yaml:"enabled"`
}

func listEnabledEnvironments(envFilePath string) ([]environment.Environment, error) {
	file, err := os.Open(envFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't open file '%s': %w", envFilePath, err)
	}

	var data envFile
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file '%s': %w", envFilePath, err)
	}

	var envs []environment.Environment
	for _, env := range data.Enabled {
		envs = append(envs, environment.Environment{Environment: env})
	}
	return envs, nil
}
