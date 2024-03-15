package template

const templateFilename = "template.yaml"

type FulfilledTemplate struct {
	Spec      *Spec
	Arguments []Argument
}

type Argument struct {
	Name  string
	Type  ParameterType
	Value string
}

type Spec struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	SkeletonPath string      `yaml:"skeletonPath"`
	Parameters   []Parameter `yaml:"parameters"`
	path         string      `yaml:"-"`
}

type Parameter struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Type        ParameterType `yaml:"type"`
}

type ParameterType string

var (
	StringType ParameterType = "string"
	IntType    ParameterType = "string"
)

func (t *Spec) IsValid() bool {
	return t.Name != ""
}
