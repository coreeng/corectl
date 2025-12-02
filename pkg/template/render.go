package template

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kluctl/go-jinja2"
)

// toYamlFilter is Python code that defines a to_yaml filter for Jinja2 templates.
// This allows templates to render objects as YAML using {{ config | to_yaml }}.
// It handles Jinja2's Undefined type to avoid pickle errors when the value is not defined.
const toYamlFilter = `
import yaml
from jinja2 import Undefined

def to_yaml(value, indent=2, default_flow_style=False):
    if value is None or isinstance(value, Undefined):
        return ""
    return yaml.dump(value, default_flow_style=default_flow_style, indent=indent, allow_unicode=True)
`

// toJsonFilter is Python code that defines a to_json filter for Jinja2 templates.
// This allows templates to render objects as JSON using {{ config | to_json }}.
// It handles Jinja2's Undefined type to avoid pickle errors when the value is not defined.
const toJsonFilter = `
import json
from jinja2 import Undefined

def to_json(value, indent=None, sort_keys=False):
    if value is None or isinstance(value, Undefined):
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

	// Jinja2 strips trailing newlines by default. Walk through rendered files
	// and ensure they end with a newline for POSIX compliance.
	if err := ensureTrailingNewlines(targetPath); err != nil {
		return err
	}

	return nil
}

// ensureTrailingNewlines walks through all files in the directory and ensures
// each file ends with a newline character.
func ensureTrailingNewlines(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Skip empty files or files that already end with newline
		if len(content) == 0 || content[len(content)-1] == '\n' {
			return nil
		}

		// Append newline and write back
		content = append(content, '\n')
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(path, content, info.Mode())
	})
}
