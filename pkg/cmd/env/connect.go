package env

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
	"os"
	"strings"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/command"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/spf13/cobra"
)

// use os args to capture flags in commands (we can't detect -- otherwise)
func unfilteredExecuteArgs() []string {
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			return os.Args[i+1:]
		}
	}
	return []string{}
}

func findEnvironmentByName(name string, environments []environment.Environment) (*environment.Environment, error) {
	for _, env := range environments {
		if name == env.Environment {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("could not find environment: %s", name)
}

func connectCmd(cfg *config.Config) *cobra.Command {
	opts := corectlenv.EnvConnectOpts{
		SkipTunnel: true,
		SilentExec: command.NewCommander(
			command.WithStdout(&bytes.Buffer{}),
			command.WithStderr(&bytes.Buffer{}),
		),
		Exec: command.NewCommander(
			command.WithStdout(os.Stdout),
			command.WithStderr(os.Stderr),
		),
	}
	connectCmd := &cobra.Command{
		Use:   "connect <environment>",
		Short: "Connect to an environment",
		Long:  `This command allows you to connect to a specified environment.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				err                   error
				availableEnvironments []environment.Environment
				envNames              []string
			)
			cmd.SilenceUsage = true

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil || corectlenv.IsConnectChild(opts) {
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

			if len(args) > 0 {
				availableEnvironments, err = environment.List(configpath.GetCorectlCPlatformDir("environments"))
				if err == nil {
					env, err := findEnvironmentByName(args[0], availableEnvironments)
					if err != nil {
						return err
					}
					opts.Environment = env
				}
				opts.Command = unfilteredExecuteArgs()
			} else {
				return fmt.Errorf("please specify the environment as the 1st argument, one of: {%s}", strings.Join(envNames, "|"))
			}

			return connect(opts, cfg, availableEnvironments)
		},
	}

	connectCmd.Flags().IntVarP(
		&opts.Port,
		"port",
		"p",
		0,
		"Local port to use for connection to the cluster",
	)
	connectCmd.Flags().BoolVarP(
		&opts.Background,
		"background",
		"b",
		false,
		"Run in background",
	)

	connectCmd.Flags().BoolVarP(
		&opts.Force,
		"force",
		"f",
		false,
		"Force replacement of existing connection",
	)

	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		connectCmd.Flags(),
	)
	return connectCmd
}

func connect(opts corectlenv.EnvConnectOpts, cfg *config.Config, availableEnvironments []environment.Environment) error {
	inputEnv := createEnvInputSwitch(opts, availableEnvironments)
	envOutput, err := inputEnv.GetValue(opts.Streams)
	if err != nil {
		return err
	}
	env, err := findEnvironmentByName(envOutput, availableEnvironments)
	if err != nil {
		return err
	}
	opts.Environment = env

	ctx := context.Background()
	gcpClient, err := setupSvc(ctx)

	if err != nil {
		return err
	}
	opts.GcpClient = gcpClient

	if err := corectlenv.Validate(ctx, env, opts.Exec, opts.GcpClient); err != nil {
		return err
	}

	gcpEnv := env.Platform.(*environment.GCPVendor)
	opts.ProjectID = gcpEnv.ProjectId
	opts.Region = gcpEnv.Region
	if err := corectlenv.Connect(opts); err != nil {
		return err
	}

	return nil
}

func setupSvc(ctx context.Context) (*gcp.Client, error) {
	clusterClient, err := gcp.NewClusterClient(ctx)
	if err != nil {
		return nil, err
	}
	gcpClient, err := gcp.NewClient(clusterClient)
	if err != nil {
		return nil, err
	}

	return gcpClient, nil
}

func createEnvInputSwitch(opts corectlenv.EnvConnectOpts, environments []environment.Environment) *userio.InputSourceSwitch[string, string] {
	validateFn := func(s string) (string, error) {
		s = strings.TrimSpace(s)
		for _, env := range environments {
			if env.Environment == s {
				return s, nil
			}
		}
		return s, errors.New("unknown environment")
	}
	return &userio.InputSourceSwitch[string, string]{
		DefaultValue: userio.AsZeroable(opts.Environment.Environment),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			envs := make([]string, len(environments))
			for i, t := range environments {
				envs[i] = t.Environment
			}
			return &userio.SingleSelect{
				Prompt: "Select environment to connect to:",
				Items:  envs,
			}, nil
		},
		ValidateAndMap: func(s string) (string, error) {
			s, err := validateFn(s)
			if err != nil {
				return "", err
			}
			return s, nil
		},
		ErrMessage: "environment is invalid",
	}
}
