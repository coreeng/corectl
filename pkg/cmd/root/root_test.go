package root

import (
	"fmt"
	"testing"

	configcmd "github.com/coreeng/corectl/pkg/cmd/config"
	"github.com/coreeng/corectl/pkg/cmd/update"
	"github.com/coreeng/corectl/pkg/cmd/version"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	oldLogger := logger.Log
	logger.Log = zap.NewNop()
	defer func() { logger.Log = oldLogger }()
	m.Run()
}

func TestCheckDuplicateCmds(t *testing.T) {
	config := config.NewConfig()
	fillCfgWithMockValues(config)
	rootCmd := NewRootCmd(config)

	// Check there's no duplicate commands
	cmdErr := checkDuplicateCmds(rootCmd)
	assert.Nil(t, cmdErr)

	// Code below here tests the check
	// Add duplicate commands
	duplicateCommands := []*cobra.Command{configcmd.NewConfigCmd(config), update.UpdateCmd(config), version.VersionCmd(config)}
	for _, v := range duplicateCommands {
		rootCmd.AddCommand(v)
	}
	cmdErr = checkDuplicateCmds(rootCmd)
	var err_suffix []string
	for _, v := range duplicateCommands {
		err_suffix = append(err_suffix, v.Use)
	}
	assert.ErrorContains(t, cmdErr, fmt.Sprintf("duplicate cli commands found: %s", err_suffix))

	// Remove duplicate commands
	for _, v := range duplicateCommands {
		rootCmd.RemoveCommand(v)
	}
	cmdErr = checkDuplicateCmds(rootCmd)
	assert.Nil(t, cmdErr)

	// Add duplicate nested commands
	rootCmd.Commands()[0].AddCommand(duplicateCommands[0])
	rootCmd.Commands()[0].AddCommand(duplicateCommands[0])
	cmdErr = checkDuplicateCmds(rootCmd)
	assert.ErrorContains(t, cmdErr, fmt.Sprintf("duplicate cli commands found: %s", []string{duplicateCommands[0].Use}))

	// Remove duplicate nested commands again
	rootCmd.Commands()[0].RemoveCommand((duplicateCommands[0]))
	rootCmd.Commands()[0].RemoveCommand((duplicateCommands[0]))
	cmdErr = checkDuplicateCmds(rootCmd)
	assert.Nil(t, cmdErr)
}

// A simple recursive check to just make sure commands and subcommands are unique
func checkDuplicateCmds(rootCmd *cobra.Command) (err error) {
	var duplicateCmds []string
	commandMap := make(map[string]int)
	for _, cmd := range rootCmd.Commands() {
		if cmd.Commands() != nil {
			err := checkDuplicateCmds(cmd)
			if err != nil {
				return err
			}
		}
		commandMap[cmd.Use] += 1
		if commandMap[cmd.Use] > 1 {
			duplicateCmds = append(duplicateCmds, cmd.Use)
		}
	}
	if duplicateCmds != nil {
		return fmt.Errorf("duplicate cli commands found: %s", duplicateCmds)
	}
	return nil
}

func fillCfgWithMockValues(cfg *config.Config) {
	cfg.GitHub.Token.Value = "gh_token-qwerty"
	cfg.GitHub.Organization.Value = "organization"

	cfg.Repositories.CPlatform.Value = "https://github.com/org/cplatform"
	cfg.Repositories.Templates.Value = "https://github.com/org/templates"
	cfg.Repositories.AllowDirty.Value = true

	cfg.P2P.FastFeedback.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.ExtendedTest.DefaultEnvs.Value = []string{"dev"}
	cfg.P2P.Prod.DefaultEnvs.Value = []string{"prod"}
}
