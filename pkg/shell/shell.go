package shell

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"
)

func RunCommand(dir string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	if dir == "." {
		path, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		cmd.Dir = path
	} else {
		cmd.Dir = dir
	}
	logger.Info("shell: running command",
		zap.String("command", name),
		zap.String("directory", dir),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Debug("shell: command failed",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()),
		)
	}
	return stdout.String(), stderr.String(), err
}
