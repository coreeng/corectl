package utils

import (
	"bytes"
	"os/exec"

	"github.com/phuslu/log"
)

func RunCommand(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	log.Info().Msgf("Running %s in %s", name, dir)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Debug().Msgf("err %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return stdout.String(), err
}
