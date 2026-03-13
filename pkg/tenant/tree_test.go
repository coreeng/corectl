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

		// Org units are direct children of root.
		{Name: "top", Kind: "OrgUnit"},
		{Name: "bottom", Kind: "OrgUnit"},

		// Delivery units are children of their owner org unit.
		{Name: "child1", Kind: "DeliveryUnit", Owner: "top"},
		{Name: "child2", Kind: "DeliveryUnit", Owner: "top"},
		{Name: "childA", Kind: "DeliveryUnit", Owner: "bottom"},
		{Name: "childB", Kind: "DeliveryUnit", Owner: "bottom"},
	}

	node, err := GetTenantTree(tenants, coretnt.RootName)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 7)
	assert.Equal(t, items[0], coretnt.RootName)
	assert.Equal(t, items[1], "top")
	assert.Equal(t, items[2], "child1")
	assert.Equal(t, items[3], "child2")
	assert.Equal(t, items[4], "bottom")
	assert.Equal(t, items[5], "childA")
	assert.Equal(t, items[6], "childB")

	assert.Equal(t, len(lines), 7)
	assert.Equal(t, lines[0], coretnt.RootName)
	assert.Equal(t, lines[1], "├── top")
	assert.Equal(t, lines[2], "│   ├── child1")
	assert.Equal(t, lines[3], "│   └── child2")
	assert.Equal(t, lines[4], "└── bottom")
	assert.Equal(t, lines[5], "    ├── childA")
	assert.Equal(t, lines[6], "    └── childB")
}

func TestGenerateTenantTreeFromSubTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
		{Name: "top", Kind: "OrgUnit"},
		{Name: "bottom", Kind: "OrgUnit"},
		{Name: "child1", Kind: "DeliveryUnit", Owner: "top"},
		{Name: "child2", Kind: "DeliveryUnit", Owner: "top"},
		{Name: "childA", Kind: "DeliveryUnit", Owner: "bottom"},
		{Name: "childB", Kind: "DeliveryUnit", Owner: "bottom"},
	}

	node, err := GetTenantTree(tenants, "bottom")
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 3)
	assert.Equal(t, items[0], "bottom")
	assert.Equal(t, items[1], "childA")
	assert.Equal(t, items[2], "childB")

	assert.Equal(t, len(lines), 3)
	assert.Equal(t, lines[0], "bottom")
	assert.Equal(t, lines[1], "├── childA")
	assert.Equal(t, lines[2], "└── childB")
}
