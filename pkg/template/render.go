package template

import (
	"path/filepath"

	"github.com/kluctl/go-jinja2"
)

// toYamlFilter is Python code that defines a to_yaml filter for Jinja2 templates.
// This allows templates to render objects as YAML using {{ config | to_yaml }}.
const toYamlFilter = `
import yaml

def to_yaml(value, indent=2, default_flow_style=False):
    if value is None:
        return ""
    return yaml.dump(value, default_flow_style=default_flow_style, indent=indent, allow_unicode=True).rstrip('\n')
`

// toJsonFilter is Python code that defines a to_json filter for Jinja2 templates.
// This allows templates to render objects as JSON using {{ config | to_json }}.
const toJsonFilter = `
import json

def to_json(value, indent=None, sort_keys=False):
    if value is None:
        return "null"
    return json.dumps(value, indent=indent, sort_keys=sort_keys, ensure_ascii=False)
`

func Render(t *FulfilledTemplate, targetPath string) error {
	j2, err := jinja2.NewJinja2(t.Spec.Name, 1,
		jinja2.WithFilter("to_yaml", toYamlFilter),
		jinja2.WithFilter("to_json", toJsonFilter),
	)
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
