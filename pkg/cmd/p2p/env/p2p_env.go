package p2penv

import (
	"github.com/coreeng/corectl/pkg/cmd/p2p/env/list"
	sync "github.com/coreeng/corectl/pkg/cmd/p2p/env/sync"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewP2PEnvCmd(cfg *config.Config) (*cobra.Command, error) {
	p2pEnvCmd := &cobra.Command{
		Use:   "env",
		Short: "Operations with p2p environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	syncCommand, err := sync.NewP2PSyncCmd(cfg)
	if err != nil {
		return nil, err
	}
	p2pEnvCmd.AddCommand(syncCommand)
	listCommand, err := list.NewP2PListCmd(cfg)
	if err != nil {
		return nil, err
	}

	p2pEnvCmd.AddCommand(listCommand)
	return p2pEnvCmd, nil
}
