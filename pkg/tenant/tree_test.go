package tenant

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTenantTreeWithNoRootShouldFail(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.DefaultRootName},
	}

	_, err := GetTenantTree(tenants, "")
	assert.NotEqual(t, err, nil)
}

func TestGenerateTenantTreeWithOnlyTheRootTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.DefaultRootName},
	}

	node, err := GetTenantTree(tenants, coretnt.DefaultRootName)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 1)
	assert.Equal(t, items[0], coretnt.DefaultRootName)

	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], coretnt.DefaultRootName)
}

func TestGenerateTenantTreeFromManyTenantsShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: "myroot"},

		{Name: "top", Parent: "myroot"},
		{Name: "child1", Parent: "top"},
		{Name: "child11", Parent: "child1"},
		{Name: "child12", Parent: "child1"},
		{Name: "child2", Parent: "top"},
		{Name: "child21", Parent: "child2"},
		{Name: "child22", Parent: "child2"},

		{Name: "bottom", Parent: "myroot"},
		{Name: "childA", Parent: "bottom"},
		{Name: "childAA", Parent: "childA"},
		{Name: "childB", Parent: "bottom"},
	}

	node, err := GetTenantTree(tenants, "myroot")
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 12)
	assert.Equal(t, items[0], "myroot")
	assert.Equal(t, items[1], "top")
	assert.Equal(t, items[2], "child1")
	assert.Equal(t, items[3], "child11")
	assert.Equal(t, items[4], "child12")
	assert.Equal(t, items[5], "child2")
	assert.Equal(t, items[6], "child21")
	assert.Equal(t, items[7], "child22")
	assert.Equal(t, items[8], "bottom")
	assert.Equal(t, items[9], "childA")
	assert.Equal(t, items[10], "childAA")
	assert.Equal(t, items[11], "childB")

	assert.Equal(t, len(lines), 12)
	assert.Equal(t, lines[0], "myroot")
	assert.Equal(t, lines[1], "├── top")
	assert.Equal(t, lines[2], "│   ├── child1")
	assert.Equal(t, lines[3], "│   │   ├── child11")
	assert.Equal(t, lines[4], "│   │   └── child12")
	assert.Equal(t, lines[5], "│   └── child2")
	assert.Equal(t, lines[6], "│       ├── child21")
	assert.Equal(t, lines[7], "│       └── child22")
	assert.Equal(t, lines[8], "└── bottom")
	assert.Equal(t, lines[9], "    ├── childA")
	assert.Equal(t, lines[10], "    │   └── childAA")
	assert.Equal(t, lines[11], "    └── childB")
}

func TestGenerateTenantTreeFromSubTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.DefaultRootName},

		{Name: "top", Parent: coretnt.DefaultRootName},
		{Name: "child1", Parent: "top"},
		{Name: "child11", Parent: "child1"},
		{Name: "child12", Parent: "child1"},
		{Name: "child2", Parent: "top"},
		{Name: "child21", Parent: "child2"},
		{Name: "child22", Parent: "child2"},

		{Name: "bottom", Parent: coretnt.DefaultRootName},
		{Name: "childA", Parent: "bottom"},
		{Name: "childAA", Parent: "childA"},
		{Name: "childB", Parent: "bottom"},
	}

	node, err := GetTenantTree(tenants, "bottom")
	assert.Equal(t, err, nil)
	assert.NotEqual(t, node, nil)

	items, lines := RenderTenantTree(node)

	assert.Equal(t, len(items), 4)
	assert.Equal(t, items[0], "bottom")
	assert.Equal(t, items[1], "childA")
	assert.Equal(t, items[2], "childAA")
	assert.Equal(t, items[3], "childB")

	assert.Equal(t, len(lines), 4)
	assert.Equal(t, lines[0], "bottom")
	assert.Equal(t, lines[1], "├── childA")
	assert.Equal(t, lines[2], "│   └── childAA")
	assert.Equal(t, lines[3], "└── childB")
}
