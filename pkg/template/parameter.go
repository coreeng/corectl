package template

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ImplicitParameters = []Parameter{
	{
		Name:        "name",
		Description: "application name",
		Type:        StringParamType,
		Optional:    false,
	},
	{
		Name:        "tenant",
		Description: "tenant used to deploy the application",
		Type:        StringParamType,
		Optional:    false,
	},
	{
		Name:        "working_directory",
		Description: "working directory where application is located",
		Type:        StringParamType,
		Optional:    false,
		Default:     "./",
	},
	{
		Name:        "version_prefix",
		Description: "version prefix for application",
		Type:        StringParamType,
		Optional:    false,
		Default:     "v",
	},
}

type Parameter struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Type        ParameterType `yaml:"type"`
	Default     string        `yaml:"default"`
	Optional    bool          `yaml:"optional"`
}

type ParameterType string

var (
	StringParamType ParameterType = "string"
	IntParamType    ParameterType = "int"
)

func (p Parameter) ValidateAndMap(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value != "" {
		mappedValue, err := p.Type.ValidateAndMap(value)
		if err != nil {
			return nil, err
		}
		return mappedValue, nil
	}
	if p.Default != "" {
		mappedDefaultValue, err := p.Type.ValidateAndMap(p.Default)
		if err != nil {
			return nil, err
		}
		return mappedDefaultValue, nil
	}
	if !p.Optional {
		return nil, errors.New("required")
	}
	return nil, nil
}

func (t ParameterType) ValidateAndMap(value string) (any, error) {
	switch t {
	case StringParamType:
		return value, nil
	case IntParamType:
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.New("integer is expected")
		}
		return intValue, nil
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", t)
	}
}
