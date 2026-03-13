package create

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
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

	tenantMap := map[string]*coretnt.Tenant{ou.Name: ou}
	err := validateTenant(tenantMap, ou)
	require.Error(t, err)
	require.Contains(t, err.Error(), "admin group must be present")
	require.Contains(t, err.Error(), "read only group must be present")
}
