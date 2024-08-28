package selector

import (
	"fmt"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
	coretnt "github.com/coreeng/developer-platform/pkg/tenant"
	"slices"
	"strings"
)

func Tenant(cPlatRepoPath string, overrideTenantName string, streams userio.IOStreams) (*coretnt.Tenant, error) {
	cPlatRepoPath = coretnt.DirFromCPlatformPath(cPlatRepoPath)
	existingTenants, err := coretnt.List(cPlatRepoPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load tenant configuration in path %s: %w", cPlatRepoPath, err)
	}
	inputTenant := createTenantInput(overrideTenantName, existingTenants)
	tenantOutput, err := inputTenant.GetValue(streams)
	if err != nil {
		return nil, fmt.Errorf("config repo path %s: %w", cPlatRepoPath, err)
	}
	return tenantOutput, nil
}

func Environment(cPlatRepoPath, overrideEnvName string, tenantOnboardedEnvs []string, streams userio.IOStreams) (*environment.Environment, error) {
	cPlatEnvRepoPath := environment.DirFromCPlatformRepoPath(cPlatRepoPath)
	tenantEnvs, err := getTenantEnvs(cPlatEnvRepoPath, tenantOnboardedEnvs)
	if err != nil {
		return nil, err
	}
	if len(*tenantEnvs) == 0 {
		return nil, fmt.Errorf("tenant env %s doesn't exist in tenant configuration %s", overrideEnvName, coretnt.DirFromCPlatformPath(cPlatRepoPath))
	}
	inputEnv := createEnvInputSwitch(overrideEnvName, *tenantEnvs)
	envOutput, err := inputEnv.GetValue(streams)
	if err != nil {
		return nil, fmt.Errorf("config repo path %s: %w", cPlatEnvRepoPath, err)
	}
	return envOutput, nil
}

func getTenantEnvs(cPlatEnvRepoPath string, tenantEnvNames []string) (*[]environment.Environment, error) {
	allEnvs, err := environment.List(cPlatEnvRepoPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load environment configuration: %w", err)
	}
	return sliceFilter(allEnvs, func(e environment.Environment) bool {
		return slices.Contains(tenantEnvNames, e.Environment)
	}), nil
}

func createEnvInputSwitch(defaultEnv string, environments []environment.Environment) *userio.InputSourceSwitch[string, *environment.Environment] {
	validateFn := func(env string) (*environment.Environment, error) {
		env = strings.TrimSpace(env)
		envIndex := slices.IndexFunc(environments, func(e environment.Environment) bool {
			return e.Environment == env
		})
		if envIndex < 0 {
			return nil, fmt.Errorf("cannot find %s environment, available envs: %v", defaultEnv, sliceMap(environments, func(e environment.Environment) string {
				return e.Environment
			}))
		}
		return &environments[envIndex], nil
	}
	return &userio.InputSourceSwitch[string, *environment.Environment]{
		DefaultValue: userio.AsZeroable(defaultEnv),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			envs := make([]string, len(environments))
			for i, t := range environments {
				envs[i] = t.Environment
			}
			return &userio.SingleSelect{
				Prompt: "Select environment to connect to:",
				Items:  envs,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     fmt.Sprintf("environment %s invalid", defaultEnv),
	}
}

func createTenantInput(defaultTenant string, existingTenants []coretnt.Tenant) *userio.InputSourceSwitch[string, *coretnt.Tenant] {
	var validateFq = func(e string) (*coretnt.Tenant, error) {
		inpName := strings.TrimSpace(e)
		tenantIndex := slices.IndexFunc(existingTenants, func(t coretnt.Tenant) bool {
			return t.Name == inpName
		})
		if tenantIndex < 0 {
			return nil, fmt.Errorf("cannot find %s tenant, available tenants: %v", e, sliceMap(existingTenants, func(t coretnt.Tenant) string {
				return t.Name
			}))
		}
		return &existingTenants[tenantIndex], nil
	}
	availableTenantNames := make([]string, len(existingTenants))
	for i, t := range existingTenants {
		availableTenantNames[i] = t.Name
	}
	return &userio.InputSourceSwitch[string, *coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(defaultTenant),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Tenant:",
				Items:  availableTenantNames,
			}, nil
		},
		ValidateAndMap: validateFq,
		ErrMessage:     fmt.Sprintf("tenant %s invalid", defaultTenant),
	}
}

// apply function to each element of the slice
func sliceMap[S ~[]E, E any](s S, f func(E) string) []string {
	vsm := make([]string, len(s))
	for i, v := range s {
		vsm[i] = f(v)
	}
	return vsm
}

// apply function to each element of the slice and return a slice with elements satisfying the function
func sliceFilter[T any](s []T, p func(T) bool) *[]T {
	d := &[]T{}
	for _, e := range s {
		if p(e) {
			*d = append(*d, e)
		}
	}
	return d
}
