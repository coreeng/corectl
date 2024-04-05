package p2p

import (
	"github.com/coreeng/corectl/pkg/cmd/p2p/create"
	"github.com/coreeng/corectl/pkg/cmd/p2p/list"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewP2PCmd(cfg *config.Config) (*cobra.Command, error) {
	p2pCmd := &cobra.Command{
		Use:   "p2p",
		Short: "Operations with p2p environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	createCommand, err := create.NewP2PCreateCmd(cfg)
	if err != nil {
		return nil, err
	}
	p2pCmd.AddCommand(createCommand)
	//p2pCmd.AddCommand(update.NewP2PUpdateCmd(cfg))
	listCommand, err := list.NewP2PListCmd(cfg)
	if err != nil {
		return nil, err
	}

	p2pCmd.AddCommand(listCommand)
	return p2pCmd, nil
}
