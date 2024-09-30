package render

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/template"
	"gopkg.in/yaml.v3"
)

func CollectArgsFromAllSources(
	spec *template.Spec,
	argsFile string,
	flagArgsRaw []string,
	streams userio.IOStreams,
	existingArgs []template.Argument,
) ([]template.Argument, error) {
	fileArgs, err := parseArgsFile(spec, argsFile)
	if err != nil {
		return nil, err
	}

	flagArgs, err := parseArgsFromFlags(spec, flagArgsRaw)
	if err != nil {
		return nil, err
	}

	passedArgs := make([]template.Argument, 0, len(fileArgs)+len(flagArgs))
	passedArgs = append(passedArgs, fileArgs...)
	passedArgs = append(passedArgs, flagArgs...)
	for _, arg := range existingArgs {
		if spec.GetParameter(arg.Name) != nil {
			passedArgs = append(passedArgs, arg)
		}
	}
	missingParameters := collectAllMissingParameters(spec, passedArgs)

	var promptedArgs []template.Argument
	var defaultArgs []template.Argument
	if streams.IsInteractive() {
		promptedArgs, err = promptForArgs(streams, missingParameters)
		if err != nil {
			return nil, err
		}
	} else {
		defaultArgs, err = collectDefaultArgs(missingParameters)
		if err != nil {
			return nil, err
		}
	}

	args := make([]template.Argument, 0, len(promptedArgs)+len(promptedArgs))
	args = append(args, passedArgs...)
	args = append(args, defaultArgs...)
	args = append(args, promptedArgs...)

	return args, nil
}

func collectDefaultArgs(params []template.Parameter) ([]template.Argument, error) {
	var defaultArgs []template.Argument
	for _, parameter := range params {
		if parameter.Default == "" && !parameter.Optional {
			return nil, fmt.Errorf("required argument %s is missing", parameter.Name)
		}
		if parameter.Default != "" {
			defaultValue, err := parameter.ValidateAndMap(parameter.Default)
			if err != nil {
				return nil, fmt.Errorf("default argument %s is invalid: %w", parameter.Name, err)
			}
			defaultArgs = append(defaultArgs, template.Argument{
				Name:  parameter.Name,
				Value: defaultValue,
			})
		}
	}
	return defaultArgs, nil
}

func parseArgsFromFlags(spec *template.Spec, flagArgs []string) ([]template.Argument, error) {
	args := make([]template.Argument, 0, len(flagArgs))
	for _, arg := range flagArgs {
		argName, argRawValue, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("expected format: <arg-name>=<arg-value>. got: %s", arg)
		}
		param := spec.GetParameter(argName)
		if param == nil {
			continue
		}
		argValue, err := param.ValidateAndMap(argRawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid %s arg value: %w", argName, err)
		}
		args = append(args, template.Argument{
			Name:  argName,
			Value: argValue,
		})
	}
	return args, nil
}

func parseArgsFile(spec *template.Spec, argsFile string) ([]template.Argument, error) {
	var rawArgs map[string]string
	var arguments []template.Argument

	if argsFile != "" {
		fileContent, err := os.ReadFile(argsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read args file: %w", err)
		}
		if err := yaml.Unmarshal(fileContent, &rawArgs); err != nil {
			return nil, fmt.Errorf("failed to parse args file: %w", err)
		}
	}

	for name, rawValue := range rawArgs {
		param := spec.GetParameter(name)
		if param == nil {
			continue
		}
		value, err := param.ValidateAndMap(rawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid %s arg value: %w", name, err)
		}
		arguments = append(arguments, template.Argument{
			Name:  name,
			Value: value,
		})
	}

	return arguments, nil
}

func collectAllMissingParameters(
	spec *template.Spec,
	existingArgs []template.Argument,
) []template.Parameter {
	missingParams := make([]template.Parameter, 0, len(spec.Parameters))
	for _, param := range spec.Parameters {
		argI := slices.IndexFunc(existingArgs, func(arg template.Argument) bool {
			return arg.Name == param.Name
		})
		if argI < 0 {
			missingParams = append(missingParams, param)
		}
	}
	return missingParams
}

func promptForArgs(
	streams userio.IOStreams,
	paramsToPrompt []template.Parameter,
) ([]template.Argument, error) {
	promptedArgs := make([]template.Argument, 0, len(paramsToPrompt))
	for _, param := range paramsToPrompt {
		prompt := param.Name + "(" + string(param.Type) + "):"
		if param.Description != "" {
			prompt += " " + param.Description
		}
		argInput := userio.TextInput[any]{
			Prompt:         prompt,
			ValidateAndMap: param.ValidateAndMap,
			Placeholder:    param.Default,
		}
		argValue, err := argInput.GetInput(streams)
		if err != nil {
			return nil, err
		}
		promptedArgs = append(promptedArgs, template.Argument{
			Name:  param.Name,
			Value: argValue,
		})
	}
	return promptedArgs, nil
}
