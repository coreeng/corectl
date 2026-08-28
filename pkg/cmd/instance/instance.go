package instance

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/spf13/cobra"
)

func NewCmd(runtime *platformruntime.Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "instance", Short: "Manage Core Platform instances", RunE: help}
	var instanceURL string
	addCmd := &cobra.Command{Use: "add <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := runtime.Instances.Add(args[0], instanceURL); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added instance %s\n", args[0])
		return nil
	}}
	addCmd.Flags().StringVar(&instanceURL, "url", "", "Portal URL for the Core Platform instance")
	_ = addCmd.MarkFlagRequired("url")
	cmd.AddCommand(
		addCmd,
		&cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
			instances, err := runtime.Instances.List()
			if err != nil {
				return err
			}
			current, err := runtime.Instances.Resolve("")
			if err != nil {
				return err
			}
			for _, item := range instances {
				marker := " "
				if item.Name == current.Name {
					marker = "*"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\t%s\n", marker, item.Name, item.Origin)
			}
			return nil
		}},
		&cobra.Command{Use: "current", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := runtime.Instances.Resolve("")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", selected.Name, selected.Origin)
			return nil
		}},
		&cobra.Command{Use: "use <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			if err := runtime.Instances.Use(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Using instance %s\n", args[0])
			return nil
		}},
		&cobra.Command{Use: "remove <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			if err := runtime.Instances.Remove(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed instance %s\n", args[0])
			return nil
		}},
	)
	return cmd
}

func help(cmd *cobra.Command, args []string) error { return cmd.Help() }
