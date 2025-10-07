package env

import (
	"fmt"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/spf13/cobra"
)

type ActiveOpt struct {
	RepositoryLocation string
	Streams            userio.IOStreams
	All                bool
	Restricted         bool
	Quiet              bool
}

func activeCmd(cfg *config.Config) *cobra.Command {
	var opts = ActiveOpt{}
	activeCmd := &cobra.Command{
		Use:   "active <environment>",
		Short: "Show active proxies for environments",
		Long:  `This command allows you to list the active proxies for environments.`,
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				err                   error
				availableEnvironments []environment.Environment
			)
			cmd.SilenceUsage = true

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				nonInteractive = true
			}

			opts.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
				!nonInteractive,
			)

			repoParams := []config.Parameter[string]{cfg.Repositories.CPlatform}
			err = config.Update(cfg.GitHub.Token.Value, opts.Streams, cfg.Repositories.AllowDirty.Value, repoParams)
			if err != nil {
				return fmt.Errorf("failed to update config repos: %w", err)
			}

			availableEnvironments, err = environment.List(configpath.GetCorectlCPlatformDir("environments"))
			if err != nil {
				return fmt.Errorf("unable to load environments")
			}
			if len(args) > 0 {
				var selectedEnvironments []environment.Environment
				// iterate over args adding the environment when the find it
				for _, arg := range args {
					env, err := findEnvironmentByName(arg, availableEnvironments)
					if err != nil {
						// Write a message saying environment name not found and return non-zero exit code
						return fmt.Errorf("please specify a set of environments or leave black for all environments")
					}
					selectedEnvironments = append(selectedEnvironments, *env)
				}
				availableEnvironments = selectedEnvironments
				// If we have chosen to specify environments we always want to see them whether the proxy is running
				// or otherwise
				opts.All = true
				opts.Restricted = true
			}

			return active(opts, availableEnvironments)
		},
	}
	activeCmd.Flags().BoolVarP(
		&opts.All,
		"all",
		"a",
		false,
		"Show all environments not just ones with active proxies",
	)
	activeCmd.Flags().BoolVarP(
		&opts.Quiet,
		"quiet",
		"q",
		false,
		"Don't print output just set the exitcode",
	)

	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		activeCmd.Flags(),
	)
	return activeCmd
}

func active(opts ActiveOpt, environments []environment.Environment) error {
	// Call getProxyPids to get pids
	pids, err := corectlenv.GetProxyPids(environments)
	_ = err

	table := corectlenv.NewTable(opts.Streams, true)
	allHaveProxies := true
	for _, env := range environments {
		proxy, pid := "-", "-"
		if proc, exists := pids[env.Environment]; exists {
			proxy, pid = fmt.Sprintf("localhost:%d", proc.Port), fmt.Sprintf("%d", proc.Pid)
		} else {
			allHaveProxies = false
			if !opts.All {
				continue
			}
		}
		table.AppendEnv(env, proxy, pid)
	}
	if !opts.Quiet {
		table.Render()
	}
	if opts.Restricted && !allHaveProxies {
		return fmt.Errorf("not all specified environments have active proxies")
	}

	return nil
}
