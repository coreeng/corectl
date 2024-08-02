package add_repo

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/git"
	corectltnt "github.com/coreeng/corectl/pkg/tenant"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
	"net/url"
	"path"
)

type TenantAddRepoOpts struct {
	TenantName    string
	RepositoryUrl string

	Streams userio.IOStreams
}

func NewTenantAddRepoCmd(cfg *config.Config) *cobra.Command {
	opts := TenantAddRepoOpts{}
	tenantAddRepoCmd := &cobra.Command{
		Use:   "add-repo <tenant-name> <repository-url>",
		Short: "Add a repository to the tenant",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TenantName = args[0]
			opts.RepositoryUrl = args[1]
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}

	config.RegisterStringParameterAsFlag(&cfg.Repositories.CPlatform, tenantAddRepoCmd.Flags())
	config.RegisterStringParameterAsFlag(&cfg.GitHub.Token, tenantAddRepoCmd.Flags())

	return tenantAddRepoCmd
}

func run(opts *TenantAddRepoOpts, cfg *config.Config) error {
	if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform); err != nil {
		return err
	}

	tenantsDir := tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value)
	t, err := tenant.FindByName(tenantsDir, opts.TenantName)
	if err != nil {
		return fmt.Errorf("failed to find the tenant: %w", err)
	}
	if t == nil {
		return fmt.Errorf("tenant is not found: %s", opts.TenantName)
	}

	if err := t.AddRepository(opts.RepositoryUrl); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	repoUrl, err := url.Parse(opts.RepositoryUrl)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	pathLastPart := path.Base(repoUrl.Path)
	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	gitAuth := git.UrlTokenAuthMethod(cfg.GitHub.Token.Value)
	result, err := corectltnt.CreateOrUpdate(&corectltnt.CreateOrUpdateOp{
		Tenant:            t,
		CplatformRepoPath: cfg.Repositories.CPlatform.Value,
		BranchName:        fmt.Sprintf("%s-add-repo-%s", t.Name, pathLastPart),
		CommitMessage:     fmt.Sprintf("Add repository %s for tenant %s", pathLastPart, t.Name),
		PRName:            fmt.Sprintf("Add repository %s for tenant %s", pathLastPart, t.Name),
		PRBody:            fmt.Sprintf("Add repository %s for tenant %s", pathLastPart, t.Name),
		GitAuth:           gitAuth,
	}, githubClient)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	opts.Streams.Info("Created PR link: ", result.PRUrl)
	opts.Streams.Info("Added repository for tenant ", t.Name, " successfully")

	return nil
}
