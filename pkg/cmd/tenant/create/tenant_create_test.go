package create

import (
	"testing"

	"github.com/coreeng/core-platform/pkg/environment"
	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	corectltenant "github.com/coreeng/corectl/pkg/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTenant_OrgUnitRequiresGroups(t *testing.T) {
	ou := &coretnt.Tenant{
		Name:         "test-ou",
		Kind:         "OrgUnit",
		ContactEmail: "ou@example.com",
		Environments: []string{"dev"},
		// AdminGroup / ReadOnlyGroup intentionally omitted
	}

	err := corectltenant.ValidateNewTenant(nil, ou)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin group must be present")
	require.Contains(t, err.Error(), "read only group must be present")
}

func TestValidateNewTenant_IgnoresUnrelatedExistingErrors(t *testing.T) {
	// Existing OU is invalid (missing groups), but should not block creating a valid DU.
	existing := []coretnt.Tenant{{
		Name:         "broken-ou",
		Kind:         "OrgUnit",
		ContactEmail: "ou@example.com",
		Environments: []string{"dev"},
	}}

	du := &coretnt.Tenant{
		Name:          "new-du",
		Kind:          "DeliveryUnit",
		Type:          "application",
		Owner:         "some-owner",
		ContactEmail:  "du@example.com",
		Environments:  []string{"dev"},
		AdminGroup:    "g1",
		ReadOnlyGroup: "g2",
	}

	err := corectltenant.ValidateNewTenant(existing, du)
	require.NoError(t, err)
}

// createNameInputSwitch tests

func TestCreateNameInputSwitch_ValidName(t *testing.T) {
	opt := &TenantCreateOpt{}
	input := opt.createNameInputSwitch(nil)

	result, err := input.ValidateAndMap("new-ou")

	require.NoError(t, err)
	assert.Equal(t, "new-ou", result)
}

func TestCreateNameInputSwitch_DuplicateName(t *testing.T) {
	existing := []coretnt.Tenant{{Name: "existing-ou"}}
	opt := &TenantCreateOpt{}
	input := opt.createNameInputSwitch(existing)

	result, err := input.ValidateAndMap("existing-ou")

	assert.ErrorContains(t, err, "tenant already exists")
	assert.Empty(t, result)
}

func TestCreateNameInputSwitch_RejectsRootName(t *testing.T) {
	opt := &TenantCreateOpt{}
	input := opt.createNameInputSwitch(nil)

	result, err := input.ValidateAndMap(coretnt.RootName)

	assert.ErrorContains(t, err, "tenant already exists")
	assert.Empty(t, result)
}

func TestCreateNameInputSwitch_InvalidK8SName(t *testing.T) {
	opt := &TenantCreateOpt{}
	input := opt.createNameInputSwitch(nil)

	result, err := input.ValidateAndMap("My Org!")

	assert.ErrorContains(t, err, "should be lower case and/or split by '-'")
	assert.Empty(t, result)
}

// createEnvironmentsInputSwitch tests

func TestCreateEnvironmentsInputSwitch_ValidSelection(t *testing.T) {
	envs := []environment.Environment{{Environment: "dev"}, {Environment: "prod"}}
	opt := &TenantCreateOpt{}
	input := opt.createEnvironmentsInputSwitch(envs)

	result, err := input.ValidateAndMap([]string{"dev", "prod"})

	require.NoError(t, err)
	assert.Equal(t, []string{"dev", "prod"}, result)
}

func TestCreateEnvironmentsInputSwitch_UnknownEnvironment(t *testing.T) {
	envs := []environment.Environment{{Environment: "dev"}}
	opt := &TenantCreateOpt{}
	input := opt.createEnvironmentsInputSwitch(envs)

	result, err := input.ValidateAndMap([]string{"staging"})

	assert.ErrorContains(t, err, "unknown environment: staging")
	assert.Nil(t, result)
}

func TestCreateEnvironmentsInputSwitch_EmptySelection(t *testing.T) {
	envs := []environment.Environment{{Environment: "dev"}}
	opt := &TenantCreateOpt{}
	input := opt.createEnvironmentsInputSwitch(envs)

	result, err := input.ValidateAndMap([]string{})

	assert.ErrorContains(t, err, "at least one environment must be selected")
	assert.Nil(t, result)
}

func TestCreateEnvironmentsInputSwitch_InteractivePromptItems(t *testing.T) {
	envs := []environment.Environment{{Environment: "dev"}, {Environment: "prod"}}
	opt := &TenantCreateOpt{}
	input := opt.createEnvironmentsInputSwitch(envs)

	prompt, err := input.InteractivePromptFn()

	require.NoError(t, err)
	multiSelect, ok := prompt.(*userio.MultiSelect)
	require.True(t, ok)
	assert.Equal(t, []string{"dev", "prod"}, multiSelect.Items)
}

// createPrefixInputSwitch tests

func TestCreatePrefixInputSwitch_EmptyIsAllowed(t *testing.T) {
	opt := &TenantCreateOpt{}
	input := opt.createPrefixInputSwitch()

	result, err := input.ValidateAndMap("")

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestCreatePrefixInputSwitch_ValidPrefix(t *testing.T) {
	opt := &TenantCreateOpt{}
	input := opt.createPrefixInputSwitch()

	result, err := input.ValidateAndMap("area/subarea")

	require.NoError(t, err)
	assert.Equal(t, "area/subarea", result)
}
