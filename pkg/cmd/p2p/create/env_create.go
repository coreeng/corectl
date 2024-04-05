package create

import (
	"context"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/google/go-github/v59/github"
	"github.com/spf13/cobra"
)

type EnvCreateOpts struct {
	NonInteractive bool
	Repo           string
	Name           string
	BaseDomain     string
	InternalDomain string
	Dplatform      string
	ProjectID      string
	ProjectNumber  string

	Streams userio.IOStreams
}

func NewP2PCreateCmd(cfg *config.Config) (*cobra.Command, error) {

	var opts = EnvCreateOpts{}
	var createEnvironmentsCmd = &cobra.Command{
		Use:   "create",
		Short: "Create Environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				!opts.NonInteractive,
			)
			return run(&opts, cfg)
		},
	}

	createEnvironmentsCmd.Flags().StringVarP(
		&opts.Repo,
		"repo",
		"r",
		"",
		"Application Repository")
	createEnvironmentsCmd.MarkFlagRequired("repo")
	createEnvironmentsCmd.Flags().StringVarP(
		&opts.Dplatform,
		"dplatform",
		"d",
		"",
		"Environment")
	createEnvironmentsCmd.MarkFlagRequired("dplatform")

	createEnvironmentsCmd.Flags().StringVarP(
		&opts.BaseDomain,
		"basedomain",
		"b",
		"",
		"Base Domain")
	createEnvironmentsCmd.MarkFlagRequired("basedomain")

	createEnvironmentsCmd.Flags().StringVarP(
		&opts.InternalDomain,
		"isdomain",
		"",
		"",
		"Internal Services Domain")
	createEnvironmentsCmd.MarkFlagRequired("isdomain")

	createEnvironmentsCmd.Flags().StringVarP(
		&opts.ProjectID,
		"projectid",
		"i",
		"",
		"Project ID")
	createEnvironmentsCmd.MarkFlagRequired("projectid")

	createEnvironmentsCmd.Flags().StringVarP(
		&opts.ProjectNumber,
		"projectnum",
		"n",
		"",
		"Project Number")
	createEnvironmentsCmd.MarkFlagRequired("projectnum")

	createEnvironmentsCmd.Flags().BoolVar(
		&opts.NonInteractive,
		"nonint",
		false,
		"Disable interactive inputs")

	return createEnvironmentsCmd, nil
}

func run(opts *EnvCreateOpts, cfg *config.Config) error {

	varsToCreate := []github.ActionsVariable{
		{
			Name:  "BASE_DOMAIN",
			Value: opts.BaseDomain,
		},
		{
			Name:  "INTERNAL_SERVICES_DOMAIN",
			Value: opts.InternalDomain,
		},
		{
			Name:  "DPLATFORM",
			Value: opts.Dplatform,
		},
		{
			Name:  "PROJECT_ID",
			Value: opts.ProjectID,
		},
		{
			Name:  "PROJECT_NUMBER",
			Value: opts.ProjectNumber,
		},
	}

	githubClient := github.NewClient(nil).
		WithAuthToken(cfg.GitHub.Token.Value)
	opts.Streams.Info("Creating Environment...")
	environments, _, err := githubClient.Repositories.CreateUpdateEnvironment(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.Repo,
		opts.Dplatform,
		&github.CreateUpdateEnvironment{},
	)
	if err != nil {
		return err
	}

	repository, _, err := githubClient.Repositories.Get(
		context.Background(),
		cfg.GitHub.Organization.Value,
		opts.Repo)
	opts.Streams.Info("Repo: " + *repository.Name)

	for i := range varsToCreate {
		opts.Streams.Info("Adding Var: " + varsToCreate[i].Name)
		_, err = githubClient.Actions.CreateEnvVariable(
			context.Background(),
			int(repository.GetID()),
			*environments.Name,
			&varsToCreate[i],
		)
		if err != nil {
			return err
		}
	}

	return nil
}
