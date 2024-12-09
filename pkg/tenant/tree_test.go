package tenant

import (
	"testing"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTenantTreeWithNoTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
	}

	nodes, err := GetTenantTrees(tenants, "")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(nodes), 0)
}

func TestGenerateTenantTreeFromJustOneTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
		{Name: "top-level", Parent: coretnt.RootName},
	}

	nodes, err := GetTenantTrees(tenants, "")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(nodes), 1)

	lines := RenderTenantTree(nodes[0])
	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], "top-level")
}

func TestGenerateTenantTreeFromSingleTopLevelTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},
		{Name: "top", Parent: coretnt.RootName},
		{Name: "child1", Parent: "top"},
		{Name: "child11", Parent: "child1"},
		{Name: "child12", Parent: "child1"},
		{Name: "child2", Parent: "top"},
		{Name: "child21", Parent: "child2"},
		{Name: "child22", Parent: "child2"},
	}

	nodes, err := GetTenantTrees(tenants, "")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(nodes), 1)

	lines := RenderTenantTree(nodes[0])
	assert.Equal(t, len(lines), 7)
	assert.Equal(t, lines[0], "top")
	assert.Equal(t, lines[1], "├── child1")
	assert.Equal(t, lines[2], "│   ├── child11")
	assert.Equal(t, lines[3], "│   └── child12")
	assert.Equal(t, lines[4], "└── child2")
	assert.Equal(t, lines[5], "    ├── child21")
	assert.Equal(t, lines[6], "    └── child22")
}

func TestGenerateTenantTreeFromMultipleTopLevelTenantsShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},

		{Name: "top", Parent: coretnt.RootName},
		{Name: "child1", Parent: "top"},
		{Name: "child11", Parent: "child1"},

		{Name: "bottom", Parent: coretnt.RootName},
		{Name: "final", Parent: "bottom"},
	}

	nodes, err := GetTenantTrees(tenants, "")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(nodes), 2)

	lines := RenderTenantTree(nodes[0])
	assert.Equal(t, len(lines), 3)
	assert.Equal(t, lines[0], "top")
	assert.Equal(t, lines[1], "└── child1")
	assert.Equal(t, lines[2], "    └── child11")

	lines = RenderTenantTree(nodes[1])
	assert.Equal(t, len(lines), 2)
	assert.Equal(t, lines[0], "bottom")
	assert.Equal(t, lines[1], "└── final")
}

func TestGenerateTenantTreeFromSubTenantShouldSucceed(t *testing.T) {
	tenants := []coretnt.Tenant{
		{Name: coretnt.RootName},

		{Name: "top", Parent: coretnt.RootName},
		{Name: "child1", Parent: "top"},
		{Name: "child11", Parent: "child1"},
		{Name: "child12", Parent: "child1"},
		{Name: "child2", Parent: "top"},
		{Name: "child21", Parent: "child2"},
		{Name: "child22", Parent: "child2"},

		{Name: "bottom", Parent: coretnt.RootName},
		{Name: "childA", Parent: "bottom"},
		{Name: "childAA", Parent: "childA"},
		{Name: "childB", Parent: "bottom"},
	}

	nodes, err := GetTenantTrees(tenants, "bottom")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(nodes), 1)

	lines := RenderTenantTree(nodes[0])
	assert.Equal(t, len(lines), 4)
	assert.Equal(t, lines[0], "bottom")
	assert.Equal(t, lines[1], "├── childA")
	assert.Equal(t, lines[2], "│   └── childAA")
	assert.Equal(t, lines[3], "└── childB")
}
