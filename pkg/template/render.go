package template

import (
	"path/filepath"

	"github.com/kluctl/go-jinja2"
)

func Render(t *FulfilledTemplate, targetPath string) error {
	j2, err := jinja2.NewJinja2(t.Spec.Name, 1)
	if err != nil {
		return err
	}
	defer j2.Close()
	defer j2.Cleanup()

	vars := map[string]any{}
	for _, arg := range t.Arguments {
		vars[arg.Name] = arg.Value
	}
	tPath := filepath.Join(t.Spec.path, t.Spec.SkeletonPath)
	if err := j2.RenderDirectory(
		tPath,
		targetPath,
		[]string{},
		jinja2.WithGlobals(vars),
	); err != nil {
		return err
	}
	return nil
}
