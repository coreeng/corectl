package template

import "slices"

const templateFilename = "template.yaml"

type FulfilledTemplate struct {
	Spec      *Spec
	Arguments []Argument
}

type Argument struct {
	Name  string
	Value any
}

type Spec struct {
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description"`
	SkeletonPath string         `yaml:"skeletonPath"`
	Parameters   []Parameter    `yaml:"parameters"`
	Config       map[string]any `yaml:"config"`
	path         string         `yaml:"-"`
}

func (t *Spec) IsValid() bool {
	return t.Name != ""
}

func (t *Spec) GetParameter(name string) *Parameter {
	paramI := slices.IndexFunc(t.Parameters, func(p Parameter) bool {
		return p.Name == name
	})
	if paramI < 0 {
		return nil
	}
	return &t.Parameters[paramI]
}
