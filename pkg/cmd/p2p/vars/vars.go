package vars

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/spf13/cobra"
	"strings"
)

type VarsOpts struct {
	Tenant  string
	Env     string
	Streams userio.IOStreams
}

func NewP2PVarsCmd(cfg *config.Config) (*cobra.Command, error) {
	var opts = VarsOpts{}
	var varsCommand = &cobra.Command{
		Use:   "vars <env> <tenant>", // TODO rather than env should this take fast feedback / extended test / prod?
		Short: "Echo vars for tenant",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Env = args[0]
			opts.Tenant = args[1]

			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return run(&opts, cfg)
		},
	}
	return varsCommand, nil
}

func run(opts *VarsOpts, cfg *config.Config) error {
	t, err := tenant.FindByName(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value), opts.Tenant)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("tenant not found: %s", opts.Tenant)
	}
	env, err := environment.FindByName(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value), opts.Env)
	if err != nil {
		return err
	}
	if env == nil {
		envs, _ := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
		var allEnvs []string
		for _, value := range envs{
			allEnvs = append(allEnvs, value.Environment)
		}
		return fmt.Errorf("env not found: %s. Valid values are: [%s]", opts.Env, strings.Join(allEnvs, ","))
	}
	e := env.Platform.(*environment.GCPVendor)
	// TODO source registry
	opts.Streams.Info("export ", "REGISTRY=", fmt.Sprintf("%s-docker.pkg.dev/%s/tenant/%s", e.Region, e.ProjectId, opts.Tenant))
	opts.Streams.Info("export ", "VERSION=local")
	opts.Streams.Info("# To point your shell to minikube's docker-daemon, run:")
	opts.Streams.Info(fmt.Sprintf("# eval $(corectl p2p vars %s %s)", opts.Env, opts.Tenant))
	return nil
}


