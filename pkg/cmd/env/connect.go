package env

import (
	"errors"
	"strings"

	"github.com/coreeng/corectl/pkg/command"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type EnvConnectOpt struct {
	Port               int
	Environment        string
	RepositoryLocation string
	ProjectID          string
	Region             string
	Streams            userio.IOStreams
	Exec               command.Commander
	gcpClient          gcp.Client
}

func connectCmd(cfg *config.Config) *cobra.Command {
	opts := EnvConnectOpt{
		Exec: command.NewCommander(),
	}
	connectCmd := &cobra.Command{
		Use:   "connect <environment>",
		Short: "Connect to an environment",
		Long:  `This command allows you to connect to a specified environment.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Environment = args[0]
			}
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return connect(opts, cfg)
		},
	}

	connectCmd.Flags().IntVarP(
		&opts.Port,
		"port",
		"p",
		54808,
		"Local port to use for connection to the cluster",
	)

	config.RegisterStringParameterAsFlag(
		&cfg.Repositories.CPlatform,
		connectCmd.Flags(),
	)
	config.RegisterBoolParameterAsFlag(
		&cfg.Repositories.AllowDirty,
		connectCmd.Flags(),
	)
	opts.RepositoryLocation = cfg.Repositories.CPlatform.Value

	return connectCmd
}

func connect(opts EnvConnectOpt, cfg *config.Config) error {
	if !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, false); err != nil {
			return err
		}
	}
	envs, err := environment.List(environment.DirFromCPlatformRepoPath(opts.RepositoryLocation))
	if err != nil {
		return err
	}
	inputEnv := createEnvInputSwitch(opts, envs)
	envOutput, err := inputEnv.GetValue(opts.Streams)
	if err != nil {
		return err
	}
	opts.Environment = envOutput

	env, err := environment.FindByName(environment.DirFromCPlatformRepoPath(opts.RepositoryLocation), opts.Environment)
	if err != nil {
		return err
	}

	ctx := context.Background()
	gcpClient, err := setupSvc(ctx)

	if err != nil {
		return err
	}
	opts.gcpClient = *gcpClient

	if err := corectlenv.Validate(ctx, env, opts.Exec, &opts.gcpClient); err != nil {
		return err
	}

	gcpEnv := env.Platform.(*environment.GCPVendor)
	opts.ProjectID = gcpEnv.ProjectId
	opts.Region = gcpEnv.Region
	if err := corectlenv.Connect(opts.Streams, env, opts.Exec, opts.Port); err != nil {
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

func createEnvInputSwitch(opts EnvConnectOpt, environments []environment.Environment) *userio.InputSourceSwitch[string, string] {
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
		DefaultValue: userio.AsZeroable(opts.Environment),
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
