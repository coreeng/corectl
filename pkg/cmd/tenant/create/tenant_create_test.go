package create

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	corectltenant "github.com/coreeng/corectl/pkg/tenant"
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
