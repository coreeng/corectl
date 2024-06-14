package env

import (
	"errors"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectlenv "github.com/coreeng/corectl/pkg/env"
	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var localPort int

func connectCmd(cfg *config.Config) *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect [environment]",
		Short: "Connect to an environment",
		Long:  `This command allows you to connect to a specified environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			io := userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
			)
			return connect(cfg, io)
		},
	}

	connectCmd.Flags().IntVar(&localPort, "localPort", 54808, "The local port to use for connection to the cluster")

	return connectCmd
}

func connect(cfg *config.Config, stream userio.IOStreams) error {
	envs, err := environment.List(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return err
	}

	inputEnv := createEnvInputSwitch(envs)
	envOutput, err := inputEnv.GetValue(stream)
	if err != nil {
		return err
	}

	env, err := environment.FindByName(environment.DirFromCPlatformRepoPath(cfg.Repositories.CPlatform.Value), envOutput)
	if err != nil {
		return err
	}

	ctx := context.Background()
	gcpClient, err := setupSvc(ctx)
	exec := corectlenv.NewCommand()
	if err != nil {
		return err
	}

	if err := corectlenv.Validate(ctx, env, exec, gcpClient); err != nil {
		return err
	}

	gcpEnv := env.Platform.(*environment.GCPVendor)
	if err := corectlenv.Connect(stream, env, exec, gcpEnv.ProjectId, gcpEnv.Region, localPort); err != nil {
		return err
	}

	return nil
}

func setupSvc(ctx context.Context) (*gcp.Client, error) {
	clusterClient, err := gcp.NewClusterClient(ctx)
	if err != nil {
		return nil, err
	}
	gcpClient, err := gcp.NewClient(ctx, clusterClient)
	if err != nil {
		return nil, err
	}

	return gcpClient, nil
}

func createEnvInputSwitch(environments []environment.Environment) *userio.InputSourceSwitch[string, string] {
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
		DefaultValue: userio.AsZeroable(""),
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
