package p2p

import (
	p2penv "github.com/coreeng/corectl/pkg/cmd/p2p/env"
	"github.com/coreeng/corectl/pkg/cmd/p2p/promote"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/spf13/cobra"
)

func NewP2PCmd(cfg *config.Config) (*cobra.Command, error) {
	p2pCmd := &cobra.Command{
		Use:   "p2p",
		Short: "P2P Operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}

	envCommand, err := p2penv.NewP2PEnvCmd(cfg)
	if err != nil {
		return nil, err
	}
	p2pCmd.AddCommand(envCommand)

	envCommand, err = promote.NewP2PPromoteCmd(cfg)
	if err != nil {
		return nil, err
	}
	p2pCmd.AddCommand(envCommand)

	return p2pCmd, nil
}
