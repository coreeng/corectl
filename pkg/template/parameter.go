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
		Optional:    true,
	},
	{
		Name:        "tenant",
		Description: "tenant used to deploy the application",
		Type:        StringParamType,
		Optional:    true,
	},
}

type Parameter struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Type        ParameterType `yaml:"type"`
	Optional    bool          `yaml:"optional"`
}

type ParameterType string

var (
	StringParamType ParameterType = "string"
	IntParamType    ParameterType = "int"
)

func (p Parameter) ValidateAndMap(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		if p.Optional {
			return nil, nil
		} else {
			return nil, errors.New("required")
		}
	}
	mappedValue, err := p.Type.ValidateAndMap(value)
	if err != nil {
		return nil, err
	}
	return mappedValue, nil
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
		panic(fmt.Sprintf("unsupported parameter type: %s", t))
	}
}
