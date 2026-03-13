package tenant

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTenantTreeWithNoRootShouldFail(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
	}

	_, err := GetTenantTree(tenants, "")
	assert.NotEqual(t, err, nil)
}

func TestGenerateTenantTreeWithOnlyTheRootTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
	}

	node, err := GetTenantTree(tenants, coretnt.RootName)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 1)
	assert.Equal(t, items[0], coretnt.RootName)

	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], coretnt.RootName)
}

func TestGenerateTenantTreeFromManyTenantsShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},

		{Name: "ou-alpha", Kind: "OrgUnit"},
		{Name: "du-alpha-1", Kind: "DeliveryUnit", Owner: "ou-alpha"},
		{Name: "du-alpha-2", Kind: "DeliveryUnit", Owner: "ou-alpha"},

		{Name: "ou-beta", Kind: "OrgUnit"},
		{Name: "du-beta-1", Kind: "DeliveryUnit", Owner: "ou-beta"},
		{Name: "du-beta-2", Kind: "DeliveryUnit", Owner: "ou-beta"},
		{Name: "du-beta-3", Kind: "DeliveryUnit", Owner: "ou-beta"},
	}

	node, err := GetTenantTree(tenants, coretnt.RootName)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, 8, len(items))
	assert.Equal(t, coretnt.RootName, items[0])
	assert.Equal(t, "ou-alpha", items[1])
	assert.Equal(t, "du-alpha-1", items[2])
	assert.Equal(t, "du-alpha-2", items[3])
	assert.Equal(t, "ou-beta", items[4])
	assert.Equal(t, "du-beta-1", items[5])
	assert.Equal(t, "du-beta-2", items[6])
	assert.Equal(t, "du-beta-3", items[7])

	assert.Equal(t, 8, len(lines))
	assert.Equal(t, coretnt.RootName, lines[0])
	assert.Equal(t, "├── ou-alpha", lines[1])
	assert.Equal(t, "│   ├── du-alpha-1", lines[2])
	assert.Equal(t, "│   └── du-alpha-2", lines[3])
	assert.Equal(t, "└── ou-beta", lines[4])
	assert.Equal(t, "    ├── du-beta-1", lines[5])
	assert.Equal(t, "    ├── du-beta-2", lines[6])
	assert.Equal(t, "    └── du-beta-3", lines[7])
}

func TestGenerateTenantTreeFromSubTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},

		{Name: "ou-alpha", Kind: "OrgUnit"},
		{Name: "du-alpha-1", Kind: "DeliveryUnit", Owner: "ou-alpha"},
		{Name: "du-alpha-2", Kind: "DeliveryUnit", Owner: "ou-alpha"},

		{Name: "ou-beta", Kind: "OrgUnit"},
		{Name: "du-beta-1", Kind: "DeliveryUnit", Owner: "ou-beta"},
		{Name: "du-beta-2", Kind: "DeliveryUnit", Owner: "ou-beta"},
	}

	node, err := GetTenantTree(tenants, "ou-beta")
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, 3, len(items))
	assert.Equal(t, "ou-beta", items[0])
	assert.Equal(t, "du-beta-1", items[1])
	assert.Equal(t, "du-beta-2", items[2])

	assert.Equal(t, 3, len(lines))
	assert.Equal(t, "ou-beta", lines[0])
	assert.Equal(t, "├── du-beta-1", lines[1])
	assert.Equal(t, "└── du-beta-2", lines[2])
}
