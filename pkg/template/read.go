package template

import (
	"context"
	"errors"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

func List(templatesPath string) ([]Spec, error) {
	var specs []Spec
	readNextTemplateFn, finishFn := templatesIterator(templatesPath)
	defer finishFn()
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
	readNextTemplateFn, finishFn := templatesIterator(templatesPath)
	defer finishFn()
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

func templatesIterator(templatesPath string) (func() (Spec, bool, error), func()) {
	ctx, cancel := context.WithCancel(context.Background())
	specCh := make(chan Spec)
	errCh := make(chan error)
	go func() {
		templatesAbsPath, err := filepath.Abs(templatesPath)
		if err != nil {
			cancel()
			return
		}
		if err := fs.WalkDir(os.DirFS(templatesAbsPath), ".", func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				return fs.SkipAll
			default:
			}

			if err != nil {
				return err
			}
			if !d.IsDir() {
				return nil
			}
			filename := filepath.Join(templatesAbsPath, path, templateFilename)
			_, err = os.Stat(filename)
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			fileBytes, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			var s Spec
			if err = yaml.Unmarshal(fileBytes, &s); err != nil {
				return err
			}
			if !s.IsValid() {
				return nil
			}
			s.path = filepath.Join(templatesAbsPath, path)
			fillDefaultSpecValues(&s)

			select {
			case <-ctx.Done():
				return fs.SkipAll
			case specCh <- s:
				return filepath.SkipDir
			}
		}); err != nil {
			select {
			case <-ctx.Done():
			case errCh <- err:
			}
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
	}, cancel
}

func fillDefaultSpecValues(s *Spec) {
	for i := range s.Parameters {
		if s.Parameters[i].Type == "" {
			s.Parameters[i].Type = StringParamType
		}
	}

	allParameters := make([]Parameter, 0, len(ImplicitParameters)+len(s.Parameters))
	allParameters = append(allParameters, ImplicitParameters...)
	allParameters = append(allParameters, s.Parameters...)
	s.Parameters = allParameters
}
