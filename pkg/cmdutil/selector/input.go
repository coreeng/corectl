package selector

import (
	"fmt"
	"slices"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
)

func DeliveryUnit(cPlatRepoPath string, overrideDeliveryUnitName string, streams userio.IOStreams) (*coretnt.Tenant, error) {
	existingTenants, err := coretnt.List(cPlatRepoPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load tenant configuration in path %s: %w", cPlatRepoPath, err)
	}
	deliveryUnits := *sliceFilter(existingTenants, func(t coretnt.Tenant) bool {
		return t.Kind == "DeliveryUnit"
	})
	inputDeliveryUnit := createDeliveryUnitInput(overrideDeliveryUnitName, deliveryUnits)
	deliveryUnitOutput, err := inputDeliveryUnit.GetValue(streams)
	if err != nil {
		return nil, fmt.Errorf("config repo path %s: %w", cPlatRepoPath, err)
	}
	return deliveryUnitOutput, nil
}

func OrgUnit(cPlatRepoPath string, overrideOrgUnitName string, streams userio.IOStreams) (*coretnt.Tenant, error) {
	existingTenants, err := coretnt.List(cPlatRepoPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load tenant configuration in path %s: %w", cPlatRepoPath, err)
	}
	orgUnits := *sliceFilter(existingTenants, func(t coretnt.Tenant) bool {
		return t.Kind == "OrgUnit"
	})
	inputOrgUnit := createOrgUnitInput(overrideOrgUnitName, orgUnits)
	orgUnitOutput, err := inputOrgUnit.GetValue(streams)
	if err != nil {
		return nil, fmt.Errorf("config repo path %s: %w", cPlatRepoPath, err)
	}
	return orgUnitOutput, nil
}

func createOrgUnitInput(defaultOrgUnit string, orgUnits []coretnt.Tenant) *userio.InputSourceSwitch[string, *coretnt.Tenant] {
	validateFn := func(e string) (*coretnt.Tenant, error) {
		inpName := strings.TrimSpace(e)
		idx := slices.IndexFunc(orgUnits, func(t coretnt.Tenant) bool {
			return t.Name == inpName
		})
		if idx < 0 {
			return nil, fmt.Errorf("cannot find %s org unit, available org units: %v", e, sliceMap(orgUnits, func(t coretnt.Tenant) string {
				return t.Name
			}))
		}
		return &orgUnits[idx], nil
	}
	names := sliceMap(orgUnits, func(t coretnt.Tenant) string { return t.Name })
	return &userio.InputSourceSwitch[string, *coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(defaultOrgUnit),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Org unit:",
				Items:  names,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     fmt.Sprintf("org unit %s invalid", defaultOrgUnit),
	}
}

func Environment(cPlatRepoPath, overrideEnvName string, tenantOnboardedEnvs []string, streams userio.IOStreams) (*environment.Environment, error) {
	cPlatEnvRepoPath := configpath.GetCorectlCPlatformDir("environments")
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

func createDeliveryUnitInput(defaultDeliveryUnit string, deliveryUnits []coretnt.Tenant) *userio.InputSourceSwitch[string, *coretnt.Tenant] {
	validateFn := func(e string) (*coretnt.Tenant, error) {
		inpName := strings.TrimSpace(e)
		idx := slices.IndexFunc(deliveryUnits, func(t coretnt.Tenant) bool {
			return t.Name == inpName
		})
		if idx < 0 {
			return nil, fmt.Errorf("cannot find %s delivery unit, available delivery units: %v", e, sliceMap(deliveryUnits, func(t coretnt.Tenant) string {
				return t.Name
			}))
		}
		return &deliveryUnits[idx], nil
	}
	names := sliceMap(deliveryUnits, func(t coretnt.Tenant) string { return t.Name })
	return &userio.InputSourceSwitch[string, *coretnt.Tenant]{
		DefaultValue: userio.AsZeroable(defaultDeliveryUnit),
		InteractivePromptFn: func() (userio.InputPrompt[string], error) {
			return &userio.SingleSelect{
				Prompt: "Delivery unit:",
				Items:  names,
			}, nil
		},
		ValidateAndMap: validateFn,
		ErrMessage:     fmt.Sprintf("delivery unit %s invalid", defaultDeliveryUnit),
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
