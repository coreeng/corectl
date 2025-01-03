package cmd

import (
	"github.com/coreeng/corectl/pkg/cmd/root"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/logger"
)

func Run() int {
	cfg, err := config.DiscoverConfig()
	if err != nil {
		logger.Error().Msgf("Error: %v", err)
		return 10
	}

	rootCmd := root.NewRootCmd(cfg)

	err = rootCmd.Execute()

	if err != nil {
		logger.Error().Msgf("Error: %v", err)
		return 1
	}
	return 0
}
