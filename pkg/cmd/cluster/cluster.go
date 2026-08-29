package cluster

import (
	"context"
	"fmt"
	"time"

	clusterpkg "github.com/coreeng/corectl/pkg/cluster"
	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/spf13/cobra"
)

func NewCmd(runtime *platformruntime.Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "cluster", Short: "Work with Portal-managed clusters", RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() }}
	cmd.AddCommand(listCmd(runtime), connectCmd(runtime), operationCmd(runtime, "install"), operationCmd(runtime, "convert"), operationCmd(runtime, "upgrade"), operationCmd(runtime, "repair"))
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

func operationCmd(runtime *platformruntime.Runtime, operation string) *cobra.Command {
	var kubeContext string
	var timeout time.Duration
	cmd := &cobra.Command{Use: operation + " <id>", Short: operationDescription(operation), Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := runtime.AuthenticatedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		remote, err := client.Cluster(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get Portal cluster: %w", err)
		}
		if err := runtime.Installer.Verify(ctx, remote.KubeSystemNamespaceUID, kubeContext); err != nil {
			return err
		}
		baselines, err := reportBaselines(ctx, client, remote.ID, operation)
		if err != nil {
			return err
		}
		plan, err := client.ClusterPlan(ctx, remote.ID, operation)
		if err != nil {
			return fmt.Errorf("request Portal %s plan: %w", operation, err)
		}
		if err := validatePlan(remote, plan, operation); err != nil {
			return err
		}
		if err := runtime.Installer.Apply(ctx, plan, kubeContext, timeout); err != nil {
			return err
		}
		for _, enrollment := range []*portal.EnrollmentMetadata{plan.Enrollment, plan.ControlEnrollment} {
			if enrollment == nil {
				continue
			}
			if err := confirmEnrollment(ctx, runtime, client, plan.ClusterID, *enrollment, baselines[enrollment.Role]); err != nil {
				return fmt.Errorf("helm completed but Portal could not confirm %s: %w", enrollment.Role, err)
			}
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s Cluster System for %s\n", operationResult(operation), remote.ID)
		return err
	}}
	cmd.Flags().StringVar(&kubeContext, "context", "", "Existing kubeconfig context that directly reaches this cluster")
	_ = cmd.MarkFlagRequired("context")
	cmd.Flags().DurationVar(&timeout, "timeout", clusterpkg.DefaultInstallTimeout, "Wait timeout for Helm and Portal confirmation")
	return cmd
}

func validatePlan(remote portal.Cluster, plan portal.InstallationPlan, operation string) error {
	if plan.ClusterID != remote.ID || plan.Operation != operation {
		return fmt.Errorf("portal returned a %q plan for cluster %q", plan.Operation, plan.ClusterID)
	}
	if plan.Generation != remote.Generation {
		return fmt.Errorf("cluster registration changed while requesting the %s plan; retry the command", operation)
	}
	return nil
}

func operationDescription(operation string) string {
	return map[string]string{
		"install": "Install the Portal-authorized Cluster System",
		"convert": "Convert a connected cluster to managed",
		"upgrade": "Upgrade Cluster System without rotating credentials",
		"repair":  "Repair Cluster System and rotate required credentials",
	}[operation]
}

func operationResult(operation string) string {
	return map[string]string{"install": "Installed", "convert": "Converted", "upgrade": "Upgraded", "repair": "Repaired"}[operation]
}

func reportBaselines(ctx context.Context, client *portal.Client, clusterID, operation string) (map[string]string, error) {
	if operation == "upgrade" {
		return map[string]string{}, nil
	}
	roles := []string{"runtime-agent"}
	switch operation {
	case "convert":
		roles = []string{"control-agent"}
	case "repair", "install":
		roles = append(roles, "control-agent")
	}
	result := make(map[string]string, len(roles))
	for _, role := range roles {
		report, err := client.AgentReport(ctx, clusterID, role)
		if err != nil {
			return nil, fmt.Errorf("read %s report baseline: %w", role, err)
		}
		result[role] = report.ReportedAt
	}
	return result, nil
}

func confirmEnrollment(ctx context.Context, runtime *platformruntime.Runtime, client *portal.Client, clusterID string, enrollment portal.EnrollmentMetadata, baseline string) error {
	for {
		attempt, err := client.EnrollmentAttemptStatus(ctx, clusterID, enrollment.Role, enrollment.AttemptID)
		if err != nil {
			return fmt.Errorf("check enrollment attempt: %w", err)
		}
		switch attempt.Status {
		case "pending":
			if err := runtime.Sleep(ctx, 3*time.Second); err != nil {
				return err
			}
			continue
		case "consumed":
		case "expired", "superseded", "unconfirmed":
			return fmt.Errorf("enrollment attempt is %s", attempt.Status)
		default:
			return fmt.Errorf("unknown enrollment attempt status %q", attempt.Status)
		}
		break
	}
	for {
		report, err := client.AgentReport(ctx, clusterID, enrollment.Role)
		if err != nil {
			return fmt.Errorf("read agent report: %w", err)
		}
		if report.Fresh && laterReport(report.ReportedAt, baseline) {
			return nil
		}
		if err := runtime.Sleep(ctx, 3*time.Second); err != nil {
			return err
		}
	}
}

func laterReport(reportedAt, baseline string) bool {
	reported, err := time.Parse(time.RFC3339Nano, reportedAt)
	if err != nil {
		return false
	}
	if baseline == "" {
		return true
	}
	previous, err := time.Parse(time.RFC3339Nano, baseline)
	return err == nil && reported.After(previous)
}
