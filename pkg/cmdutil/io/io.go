package io

import (
	"io"
	"os"

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

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Ensure the file is flushed to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	log.Debug().Msgf("copied file from %s -> %s", source, destination)

	return nil
}
