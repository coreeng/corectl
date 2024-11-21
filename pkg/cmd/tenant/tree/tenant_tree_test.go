package tree

import (
	"testing"

	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/stretchr/testify/assert"
)

func TestCreateTreeView(t *testing.T) {

	tenants := []tenant.Tenant{
		{Name: "root"},
	}
	tree := CreateTreeView(tenants)
	assert.Equal(t, tree.String(), "root\n")

	tenants = append(tenants, tenant.Tenant{Name: "a", Parent: "root"})
	tenants = append(tenants, tenant.Tenant{Name: "b", Parent: "root"})
	tenants = append(tenants, tenant.Tenant{Name: "c", Parent: "b"})
	tenants = append(tenants, tenant.Tenant{Name: "d", Parent: "c"})

	tree = CreateTreeView(tenants)
	assert.Equal(t, tree.String(), "root\n├── a\n└── b\n    └── c\n        └── d\n")
}
