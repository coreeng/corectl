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
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type TenantCreateOpt struct {
	Name           string
	Kind           string
	Owner          string
	Type           string
	Description    string
	ContactEmail   string
	Environments   []string
	Repo           string
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
				logger.Panic().With(zap.Error(err)).Msg("could not get non-interactive flag")
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
		&opt.Kind,
		"kind",
		"",
		"Tenant kind: 'OrgUnit' or 'DeliveryUnit'",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.Owner,
		"owner",
		"",
		"Owner OrgUnit name (required for DeliveryUnit)",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.Type,
		"type",
		"",
		"Delivery unit type: 'application' or 'infrastructure' (required for DeliveryUnit)",
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
	tenantCreateCmd.Flags().StringSliceVar(
		&opt.Environments,
		"environments",
		nil,
		"Environments, available to tenant",
	)
	tenantCreateCmd.Flags().StringVar(
		&opt.Repo,
		"repo",
		"",
		"Repository URL for the delivery unit (optional, DeliveryUnit only)",
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
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opt.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	tenantsPath := configpath.GetCorectlCPlatformDir("tenants", "tenants")
	existingTenants, err := coretnt.List(tenantsPath)
	if err != nil {
		return err
	}
	rootTenant := coretnt.RootTenant(tenantsPath)

	envsDir := configpath.GetCorectlCPlatformDir("environments")
	envFilePath := filepath.Join(envsDir, "environments.yaml")
	envs, err := listEnabledEnvironments(envFilePath)
	if err != nil {
		logger.Info().Msgf("Failed to read environments file '%s': %s. Falling back to listing directories in '%s'.", envFilePath, err, envsDir)
		envs, err = environment.List(envsDir)
		if err != nil {
			return err
		}
	}
	if len(envs) == 0 {
		return fmt.Errorf("no enabled environment found in '%s' and no environment directory found in '%s'", envFilePath, envsDir)
	}

	nameInput := opt.createNameInputSwitch(existingTenants)
	kindInput := opt.createKindInputSwitch()
	descriptionInput := opt.createDescriptionInputSwitch()
	contactEmailInput := opt.createContactEmailInputSwitch()
	envsInput := opt.createEnvironmentsInputSwitch(envs)
	adminGroupInput := opt.createAdminGroupInputSwitch()
	readOnlyGroupInput := opt.createReadOnlyGroupInputSwitch()

	name, err := nameInput.GetValue(opt.Streams)
	if err != nil {
		return err
	}
	kind, err := kindInput.GetValue(opt.Streams)
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
	tenantEnvironments, err := envsInput.GetValue(opt.Streams)
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
		Kind:          kind,
		Description:   description,
		ContactEmail:  contactEmail,
		Environments:  tenantEnvironments,
		AdminGroup:    adminGroup,
		ReadOnlyGroup: readOnlyGroup,
		CloudAccess:   make([]coretnt.CloudAccess, 0),
	}

	var ownerTenant *coretnt.Tenant

	switch kind {
	case "OrgUnit":
		if opt.Repo != "" {
			return fmt.Errorf("cannot specify --repo for OrgUnit: only DeliveryUnits can have a repository")
		}
		ownerTenant = rootTenant

	case "DeliveryUnit":
		ownerInput := opt.createOwnerInputSwitch(rootTenant, existingTenants)
		owner, err := ownerInput.GetValue(opt.Streams)
		if err != nil {
			return err
		}
		ownerTenant = &owner

		typeInput := opt.createTypeInputSwitch()
		duType, err := typeInput.GetValue(opt.Streams)
		if err != nil {
			return err
		}
		t.Type = duType
		t.Owner = owner.Name

		repoInput := opt.createRepoInputSwitch()
		repo, err := repoInput.GetValue(opt.Streams)
		if err != nil {
			return err
		}
		t.Repo = repo
	}

	_, err = createTenant(opt.DryRun, cfg, &t, ownerTenant, existingTenants)
	if err != nil {
		return err
	}
	return nil
}

func createTenant(
	dryRun bool,
	cfg *config.Config,
	t *coretnt.Tenant,
	ownerTenant *coretnt.Tenant,
	allTenants []coretnt.Tenant,
) (tenant.CreateOrUpdateResult, error) {
	logger.Warn().Msgf("Creating tenant %s in platform repository: %s", t.Name, cfg.Repositories.CPlatform.Value)

	tenantMap := map[string]*coretnt.Tenant{
		t.Name: t,
	}
	for _, tnt := range allTenants {
		tenantMap[tnt.Name] = &tnt
	}

	if err := validateTenant(tenantMap, t); err != nil {
		logger.Warn().Msgf("Unable to create such a tenant: %s", err)
		return tenant.CreateOrUpdateResult{}, err
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	result, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            t,
			OwnerTenant:       ownerTenant,
			CplatformRepoPath: configpath.GetCorectlCPlatformDir(),
			BranchName:        fmt.Sprintf("new-%s-tenant-%s", t.Kind, t.Name),
			CommitMessage:     fmt.Sprintf("Add new %s tenant: %s", t.Kind, t.Name),
			PRName:            fmt.Sprintf("New %s tenant: %s", t.Kind, t.Name),
			PRBody:            fmt.Sprintf("Adds new %s tenant '%s'", t.Kind, t.Name),
			GitAuth:           gitAuth,
			DryRun:            dryRun,
		}, githubClient,
	)
	if err != nil {
		logger.Warn().Msgf("Failed to create a PR for new %s tenant: %s", t.Kind, err)
	} else {
		logger.Warn().Msgf("Created PR for new %s tenant %s: %s", t.Kind, t.Name, result.PRUrl)
	}
	return result, err
}

func validateTenant(tenantMap map[string]*coretnt.Tenant, t *coretnt.Tenant) error {
	validationResult := coretnt.ValidateTenants(tenantMap)
	for _, warn := range validationResult.Warnings {
		var tenantRelatedWarn coretnt.TenantRelatedError
		if errors.As(warn, &tenantRelatedWarn) && tenantRelatedWarn.IsRelatedToTenant(t) {
			logger.Error().Msg(warn.Error())
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

func (opt *TenantCreateOpt) createOwnerInputSwitch(rootTenant *coretnt.Tenant, existingTenants []coretnt.Tenant) userio.InputSourceSwitch[string, coretnt.Tenant] {
	orgUnits := make([]coretnt.Tenant, 0)
	for _, t := range existingTenants {
		if t.Kind == "OrgUnit" {
			orgUnits = append(orgUnits, t)
		}
	}

	var items []string
	var lines []string
	for _, ou := range orgUnits {
		items = append(items, ou.Name)
		lines = append(lines, ou.Name)
	}

	return userio.InputSourceSwitch[string, coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(opt.Owner),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt:         "Owner OrgUnit:",
				Items:          items,
				DisplayedItems: lines,
			}, nil
		},
		ValidateAndMap: func(inp string) (coretnt.Tenant, error) {
			inp = strings.TrimSpace(inp)
			if inp == rootTenant.Name {
				return *rootTenant, nil
			}
			idx := slices.IndexFunc(orgUnits, func(t coretnt.Tenant) bool {
				return t.Name == inp
			})
			if idx >= 0 {
				return orgUnits[idx], nil
			}
			return coretnt.Tenant{}, errors.New("unknown OrgUnit")
		},
		ErrMessage: "invalid owner OrgUnit",
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

func (opt *TenantCreateOpt) createRepoInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		if inp == "" {
			return "", nil
		}
		t := &coretnt.Tenant{
			Kind: "DeliveryUnit",
			Repo: inp,
		}
		err := t.ValidateField("Repo")
		if err != nil {
			return "", err
		}
		return inp, nil
	}
	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opt.Repo),
		Optional:     true,
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:         "Repository URL (optional GitHub link):",
				ValidateAndMap: validateFn,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid repository URL",
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

func (opt *TenantCreateOpt) createKindInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		if inp != "OrgUnit" && inp != "DeliveryUnit" {
			return "", errors.New("kind must be either 'OrgUnit' or 'DeliveryUnit'")
		}
		t := &coretnt.Tenant{
			Kind: inp,
		}
		err := t.ValidateField("Kind")
		if err != nil {
			return "", err
		}
		return inp, nil
	}

	defaultKind := opt.Kind
	if defaultKind == "" {
		defaultKind = "OrgUnit"
	}

	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(defaultKind),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Tenant kind:",
				Items:  []string{"OrgUnit", "DeliveryUnit"},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid tenant kind",
	}
}

func (opt *TenantCreateOpt) createTypeInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		if inp != "application" && inp != "infrastructure" {
			return "", errors.New("type must be either 'application' or 'infrastructure'")
		}
		t := &coretnt.Tenant{
			Kind: "DeliveryUnit",
			Type: inp,
		}
		err := t.ValidateField("Type")
		if err != nil {
			return "", err
		}
		return inp, nil
	}

	defaultType := opt.Type
	if defaultType == "" {
		defaultType = "application"
	}

	return userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(defaultType),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Delivery unit type:",
				Items:  []string{"application", "infrastructure"},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid delivery unit type",
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
