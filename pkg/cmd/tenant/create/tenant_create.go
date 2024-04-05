package create

import (
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/environment"
	"github.com/coreeng/corectl/pkg/git"
	"github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"net/mail"
	"slices"
	"strings"
	"unicode/utf8"
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

	Streams userio.IOStreams
}

func NewTenantCreateCmd(cfg *config.Config) *cobra.Command {
	opt := TenantCreateOpt{}
	tenantCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
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
		[]string{},
		"Environments, available to tenant",
	)
	tenantCreateCmd.Flags().StringSliceVar(
		&opt.Repositories,
		"repositories",
		[]string{},
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
	tenantCreateCmd.Flags().BoolVar(
		&opt.NonInteractive,
		"nonint",
		false,
		"Disable interactive inputs",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		tenantCreateCmd.Flags(),
	)
	config.RegisterStringParameterAsFlag(
		&cfg.GitHub.Token,
		tenantCreateCmd.Flags(),
	)

	return tenantCreateCmd
}

func run(opt *TenantCreateOpt, cfg *config.Config) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}

	existingTenants, err := tenant.List(cfg.Repositories.CPlatform.Value)
	if err != nil {
		return err
	}
	envs, err := environment.List(cfg.Repositories.CPlatform.Value)
	if err != nil {
		return err
	}

	nameInput := opt.createNameInputSwitch()
	parentInput := opt.createParentInputSwitch(existingTenants)
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

	t := tenant.Tenant{
		Name:          name,
		Parent:        parent,
		Description:   description,
		ContactEmail:  contactEmail,
		CostCentre:    costCentre,
		Environments:  tenantEnvironments,
		Repositories:  repositories,
		AdminGroup:    adminGroup,
		ReadonlyGroup: readOnlyGroup,
		CloudAccess:   make([]interface{}, 0),
	}

	result, err := createTenant(opt.Streams, cfg, &t)
	if err != nil {
		return err
	}
	opt.Streams.Info("Created PR link: ", result.PRUrl)
	opt.Streams.Info("Tenant created successfully: ", string(t.Name))
	return nil
}

func createTenant(
	streams userio.IOStreams,
	cfg *config.Config,
	t *tenant.Tenant,
) (tenant.CreateOrUpdateResult, error) {
	spinnerHandler := streams.Spinner("Creating tenant...")
	defer spinnerHandler.Done()
	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	result, err := tenant.CreateOrUpdate(
		&tenant.CreateOrUpdateOp{
			Tenant:            t,
			CplatformRepoPath: cfg.Repositories.CPlatform.Value,
			BranchName:        fmt.Sprintf("new-tenant-%s", t.Name),
			CommitMessage:     fmt.Sprintf("Add new tenant: %s", t.Name),
			PRName:            fmt.Sprintf("New tenant: %s", t.Name),
			PRBody:            fmt.Sprintf("Adds new tenant '%s'", t.Name),
			GitAuth:           gitAuth,
		}, githubClient,
	)
	return result, err
}

func (opt *TenantCreateOpt) createNameInputSwitch() userio.InputSourceSwitch[string, tenant.Name] {
	validateFn := func(inp string) (tenant.Name, error) {
		inp = strings.TrimSpace(inp)
		name := tenant.Name(inp)
		if err := tenant.ValidateName(name); err != nil {
			return "", err
		}
		return name, nil
	}
	return userio.InputSourceSwitch[string, tenant.Name]{
		DefaultValue: userio.AsZeroable(opt.Name),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.TextInput[string]{
				Prompt:      "Tenant name (valid K8S namespace name):",
				Placeholder: "tenant-name",
				ValidateAndMap: func(inp string) (string, error) {
					name, err := validateFn(inp)
					return string(name), err
				},
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     "invalid tenant name",
	}
}

func (opt *TenantCreateOpt) createParentInputSwitch(existingTenants []tenant.Tenant) userio.InputSourceSwitch[string, tenant.Name] {
	availableTenantNames := make([]string, len(existingTenants)+1)
	availableTenantNames[0] = string(tenant.RootName)
	for i, t := range existingTenants {
		availableTenantNames[i+1] = string(t.Name)
	}
	return userio.InputSourceSwitch[string, tenant.Name]{
		DefaultValue: userio.AsZeroable(opt.Parent),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Parent tenant:",
				Items:  availableTenantNames,
			}, nil
		},
		ValidateAndMap: func(inp string) (tenant.Name, error) {
			inp = strings.TrimSpace(inp)
			if !slices.Contains(availableTenantNames, inp) {
				return "", errors.New("unknown tenant")
			}
			return tenant.Name(inp), nil
		},
		ErrMessage: "invalid parent tenant",
	}
}

func (opt *TenantCreateOpt) createDescriptionInputSwitch() userio.InputSourceSwitch[string, string] {
	validateFn := func(inp string) (string, error) {
		inp = strings.TrimSpace(inp)
		if err := tenant.ValidateDescription(inp); err != nil {
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
		_, err := mail.ParseAddress(inp)
		if err != nil {
			return "", err
		}
		return inp, err
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
		if err := tenant.ValidateCostCentre(inp); err != nil {
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

func (opt *TenantCreateOpt) createEnvironmentsInputSwitch(envs []environment.Environment) userio.InputSourceSwitch[[]string, []environment.Name] {
	var envNames []string
	for _, env := range envs {
		envNames = append(envNames, string(env.Environment))
	}
	return userio.InputSourceSwitch[[]string, []environment.Name]{
		DefaultValue: userio.AsZeroableSlice(opt.Environments),
		InteractivePromptFn: func() (userio.InputPrompt[[]string], error) {
			return &userio.MultiSelect{
				Prompt: "Environments:",
				Items:  envNames,
			}, nil
		},
		ValidateAndMap: func(inp []string) ([]environment.Name, error) {
			envs := make([]environment.Name, len(inp))
			for i, env := range inp {
				env = strings.TrimSpace(env)
				if env == "" {
					continue
				}
				if !slices.Contains(envNames, env) {
					return nil, errors.New("unknown environment")
				}
				envs[i] = environment.Name(env)
			}
			return envs, nil
		},
		ErrMessage: "invalid environment list",
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
			if err := tenant.ValidateRepositoryLink(repo); err != nil {
				return nil, err
			}
			filteredRepos = append(filteredRepos, repo)
		}
		return filteredRepos, nil

	}
	return userio.InputSourceSwitch[[]string, []string]{
		DefaultValue: userio.AsZeroableSlice(opt.Repositories),
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
		chCount := utf8.RuneCountInString(inp)
		if chCount < 1 {
			return "", errors.New("required")
		}
		if chCount > 253 {
			return "", errors.New("max length is 253")
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
		chCount := utf8.RuneCountInString(inp)
		if chCount < 1 {
			return "", errors.New("required")
		}
		if chCount > 253 {
			return "", errors.New("max length is 253")
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
