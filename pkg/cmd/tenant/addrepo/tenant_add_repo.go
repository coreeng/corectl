package addrepo

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/google/go-github/v60/github"
	"github.com/spf13/cobra"
)

type TenantAddRepoOpts struct {
	TenantName    string
	RepositoryUrl string
	DryRun        bool

	Streams userio.IOStreams
}

func NewTenantAddRepoCmd(cfg *config.Config) *cobra.Command {
	opts := TenantAddRepoOpts{}
	tenantAddRepoCmd := &cobra.Command{
		Use:   "add-repo <tenant-name> <repository-url>",
		Short: "Add a repository to the tenant",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			// TODO: Change this to named params, and use same GetInput prompts as `apps create`
			opts.TenantName = args[0]
			opts.RepositoryUrl = args[1]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(&opts, cfg)
		},
	}

	config.RegisterStringParameterAsFlag(&cfg.GitHub.Token, tenantAddRepoCmd.Flags())
	config.RegisterBoolParameterAsFlag(&cfg.Repositories.AllowDirty, tenantAddRepoCmd.Flags())

	return tenantAddRepoCmd
}

func run(opts *TenantAddRepoOpts, cfg *config.Config) error {
	repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
	err := config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
	if err != nil {
		return fmt.Errorf("failed to update config repos: %w", err)
	}

	opts.Streams.Wizard(
		fmt.Sprintf("Adding repository %s to tenant %s in platform repo %s", opts.RepositoryUrl, opts.TenantName, cfg.Repositories.CPlatform.Value),
		fmt.Sprintf("Added repository %s to tenant %s in platform repo %s", opts.RepositoryUrl, opts.TenantName, cfg.Repositories.CPlatform.Value),
	)
	defer opts.Streams.CurrentHandler.Done()

	tenantsDir := configpath.GetCorectlCPlatformDir("tenants")
	t, err := tenant.FindByName(tenantsDir, opts.TenantName)
	if err != nil {
		return fmt.Errorf("failed to find the tenant: %w", err)
	}
	if t == nil {
		return fmt.Errorf("tenant is not found: %s", opts.TenantName)
	}

	opts.Streams.CurrentHandler.Info(fmt.Sprintf("adding repository to list: %s", opts.RepositoryUrl))
	if !opts.DryRun {
		if err := t.AddRepository(opts.RepositoryUrl); err != nil {
			return fmt.Errorf("failed to add repository: %w", err)
		}
	}

	repoName, err := git.DeriveRepositoryFullnameFromUrl(opts.RepositoryUrl)
	if err != nil {
		return fmt.Errorf("couldn't derive repository name from url: %w", err)
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)

	opts.Streams.CurrentHandler.Info("creating GitHub PR")
	result, err := corectltnt.CreateOrUpdate(&corectltnt.CreateOrUpdateOp{
		Tenant:            t,
		CplatformRepoPath: configpath.GetCorectlCPlatformDir(),
		BranchName:        fmt.Sprintf("%s-add-repo-%s", t.Name, repoName.Name()),
		CommitMessage:     fmt.Sprintf("Add repository %s for tenant %s", repoName.Name(), t.Name),
		PRName:            fmt.Sprintf("Add repository %s for tenant %s", repoName.Name(), t.Name),
		PRBody:            fmt.Sprintf("Add repository %s for tenant %s", repoName.Name(), t.Name),
		GitAuth:           gitAuth,
		DryRun:            opts.DryRun,
	}, githubClient)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	opts.Streams.CurrentHandler.Info(fmt.Sprintf("created PR link: %s", result.PRUrl))
	opts.Streams.CurrentHandler.Info(fmt.Sprintf("added repository for tenant %s successfully", t.Name))
	return nil
}
