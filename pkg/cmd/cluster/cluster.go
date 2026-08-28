package cluster

import (
	"fmt"
	"time"

	clusterpkg "github.com/coreeng/corectl/pkg/cluster"
	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/spf13/cobra"
)

func NewCmd(runtime *platformruntime.Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "cluster", Short: "Work with Portal-managed clusters", RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() }}
	cmd.AddCommand(listCmd(runtime), connectCmd(runtime), installCmd(runtime))
	return cmd
}

func listCmd(runtime *platformruntime.Runtime) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List clusters", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := runtime.AuthenticatedClient()
		if err != nil {
			return err
		}
		clusters, err := client.Clusters(cmd.Context())
		if err != nil {
			return fmt.Errorf("list Portal clusters: %w", err)
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "ID\tNAME\tSTATUS\tGENERATION"); err != nil {
			return err
		}
		for _, item := range clusters {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", item.ID, item.Name, item.Status, item.Generation); err != nil {
				return err
			}
		}
		return nil
	}}
}

func connectCmd(runtime *platformruntime.Runtime) *cobra.Command {
	var sourceContext string
	var switchContext bool
	cmd := &cobra.Command{Use: "connect <id>", Short: "Create and use a verified managed kubeconfig context", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, selected, err := runtime.AuthenticatedClient()
		if err != nil {
			return err
		}
		remote, err := client.Cluster(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("get Portal cluster: %w", err)
		}
		contextName, err := runtime.Connector.Connect(cmd.Context(), selected, remote, sourceContext, switchContext)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Connected cluster %s with context %s\n", remote.ID, contextName)
		return err
	}}
	cmd.Flags().StringVar(&sourceContext, "context", "", "Existing kubeconfig context that directly reaches this cluster (required on first use and generation changes)")
	cmd.Flags().BoolVar(&switchContext, "switch-context", true, "Switch the current kubeconfig context to the managed context")
	return cmd
}

func installCmd(runtime *platformruntime.Runtime) *cobra.Command {
	var kubeContext string
	var timeout time.Duration
	cmd := &cobra.Command{Use: "install <id>", Short: "Install the Portal-authorized connected-cluster agent", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := runtime.AuthenticatedClient()
		if err != nil {
			return err
		}
		plan, err := client.InstallationPlan(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("request Portal installation plan: %w", err)
		}
		if plan.ClusterID != args[0] {
			return fmt.Errorf("portal installation plan is for cluster %q, not %q", plan.ClusterID, args[0])
		}
		if err := runtime.Installer.Install(cmd.Context(), plan, kubeContext, timeout); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Installed cluster agent for %s\n", args[0])
		return err
	}}
	cmd.Flags().StringVar(&kubeContext, "context", "", "Kubeconfig context to install into")
	_ = cmd.MarkFlagRequired("context")
	cmd.Flags().DurationVar(&timeout, "timeout", clusterpkg.DefaultInstallTimeout, "Wait timeout for the Helm installation")
	return cmd
}
