package shell

import (
	"bytes"
	"io"
	"os"
	"os/exec"

	"github.com/phuslu/log"
)

func MoveFile(source string, destination string) error {
	log.Debug().Msgf("moving file from %s -> %s", source, destination)
	err := CopyFile(source, destination)
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

func CopyFile(source string, destination string) error {
	log.Debug().Msgf("copying file from %s -> %s", source, destination)
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	fileInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	permissions := fileInfo.Mode().Perm()

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Flush file to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	// Preserve original permissions
	err = os.Chmod(destination, permissions)
	if err != nil {
		return err
	}

	log.Debug().Msgf("copied file from %s -> %s", source, destination)

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
