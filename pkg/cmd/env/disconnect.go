package env

import (
	"bytes"
	"fmt"
	"os"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmd/config/update"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/command"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/spf13/cobra"
)

func disconnectCmd(cfg *config.Config) *cobra.Command {
	opts := corectlenv.EnvConnectOpts{
		SilentExec: command.NewCommander(
			command.WithStdout(&bytes.Buffer{}),
			command.WithStderr(&bytes.Buffer{}),
		),
		Exec: command.NewCommander(
			command.WithStdout(os.Stdout),
			command.WithStderr(os.Stderr),
		),
	}
	disconnectCmd := &cobra.Command{
		Use:   "disconnect <environment>",
		Short: "Disconnect from an environment",
		Long:  `This command allows you to disconnect from a specified environment.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				err                   error
				availableEnvironments []environment.Environment
			)

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

			err = update.Update(cfg, opts.Streams)
			if err != nil {
				return fmt.Errorf("failed to update config repos: %w", err)
			}

			cmd.SilenceUsage = true
			availableEnvironments, err = environment.List(environment.DirFromCPlatformRepoPath(opts.RepositoryLocation))
			if err != nil {
				return fmt.Errorf("unable to load environments")
			}
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

			return disconnect(opts, cfg, availableEnvironments)
		},
	}

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		disconnectCmd.Flags(),
	)
	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		disconnectCmd.Flags(),
	)
	opts.RepositoryLocation = cfg.Repositories.CPlatform.Value

	return disconnectCmd
}

func disconnect(opts corectlenv.EnvConnectOpts, cfg *config.Config, environments []environment.Environment) error {
	// Call getProxyPids to get pids
	pids, err := corectlenv.GetProxyPIDs(environments)
	_ = err
	for name, pid := range pids {
		// Kill the process with the pid
		if err := corectlenv.KillProcess(name, pid.Pid, false); err != nil {
			return fmt.Errorf("[%s] failed to kill process: %w", name, err)
		}
		opts.Streams.Info(fmt.Sprintf("[%s] killed process %d on port %d", name, pid.Pid, pid.Port))
	}
	return nil
}
