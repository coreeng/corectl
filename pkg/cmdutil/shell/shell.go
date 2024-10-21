package shell

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/otiai10/copy"
	"github.com/phuslu/log"
)

func MoveFile(source string, destination string) error {
	log.Debug().Msgf("moving file from %s -> %s", source, destination)
	err := copy.Copy(source, destination)
	if err != nil {
		return err
	}

	err = os.Remove(source)
	if err != nil {
		return err
	}

	log.Debug().Msgf("moved file from %s -> %s", source, destination)

	return nil
}

func RunCommand(dir string, name string, args ...string) (string, string, error) {
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
	return stdout.String(), stderr.String(), err
}
