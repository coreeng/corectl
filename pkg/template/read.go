package template

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

func List(templatesPath string) ([]Spec, error) {
	var specs []Spec
	readNextTemplateFn := templatesIterator(templatesPath)
	for {
		spec, done, err := readNextTemplateFn()
		if err != nil {
			return nil, err
		}
		if done {
			return specs, nil
		}
		specs = append(specs, spec)
	}
}

func FindByName(templatesPath string, name string) (*Spec, error) {
	readNextTemplateFn := templatesIterator(templatesPath)
	for {
		spec, done, err := readNextTemplateFn()
		if err != nil {
			return nil, err
		}
		if done {
			return nil, nil
		}
		if spec.Name == name {
			return &spec, nil
		}
	}
}

func templatesIterator(templatesPath string) func() (Spec, bool, error) {
	specCh := make(chan Spec)
	errCh := make(chan error)
	go func() {
		if err := fs.WalkDir(os.DirFS(templatesPath), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				return nil
			}
			filename := filepath.Join(templatesPath, path, templateFilename)
			_, err = os.Stat(filename)
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			fileBytes, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			var t Spec
			if err = yaml.Unmarshal(fileBytes, &t); err != nil {
				return err
			}
			if !t.IsValid() {
				return nil
			}
			t.path = path
			specCh <- t
			return filepath.SkipDir
		}); err != nil {
			errCh <- err
		}
		close(specCh)
		close(errCh)
	}()

	return func() (Spec, bool, error) {
		select {
		case spec, isReceived := <-specCh:
			return spec, !isReceived, nil
		case err, isReceived := <-errCh:
			return Spec{}, !isReceived, err
		}
	}
}
