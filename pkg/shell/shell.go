package shell

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/coreeng/corectl/pkg/logger"
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
	logger.Info().Msgf("Running %s in %s", name, dir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Debug().Msgf("err %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), err
}
