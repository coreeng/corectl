package tenant

import (
	"fmt"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
)

type Node struct {
	Tenant   *coretnt.Tenant
	Children []*Node
}

// Builds trees of tenants.
//
// Arguments:
//
//	 tenants   Tenants to build the tree(s) from
//		root   Name of the tenant to start the tree from
//
// Returns: A pointer to the root node of the tree
func GetTenantTree(tenants []coretnt.Tenant, root string) (*Node, error) {
	// Build a map of tenants indexed by name for faster access
	nodeMap := make(map[string]*Node)
	for _, tenant := range tenants {
		nodeMap[tenant.Name] = &Node{
			Tenant:   &tenant,
			Children: []*Node{},
		}
	}

	// Populate the `Children` slices
	for _, tenant := range tenants {
		parent, exists := nodeMap[tenant.Parent]
		if exists {
			parent.Children = append(parent.Children, nodeMap[tenant.Name])
		}
	}

	rootNode, exists := nodeMap[root]
	if !exists {
		return nil, fmt.Errorf("root tenant '%s' not found", root)
	}
	return rootNode, nil
}

// Renders a tree
//
// Arguments:
//
//	node   Top-level node to render the tree from
//
// Returns: The first slice is the list of tenants names. The second slice is how the corresponding line in the first slice should be rendered.
func RenderTenantTree(root *Node) ([]string, []string) {
	var lines []string
	var renderedLines []string
	buildTree(root, "", true, true, &lines, &renderedLines)
	return lines, renderedLines
}

func buildTree(node *Node, prefix string, isLastChild bool, isRoot bool, lines *[]string, renderedLines *[]string) {
	if node == nil {
		return
	}

	var connector string
	if !isRoot {
		if isLastChild {
			connector = "└── "
		} else {
			connector = "├── "
		}
	}

	*lines = append(*lines, node.Tenant.Name)
	out := fmt.Sprintf("%s%s%s", prefix, connector, node.Tenant.Name)
	*renderedLines = append(*renderedLines, out)

	if !isRoot {
		if isLastChild {
			prefix += "    "
		} else {
			prefix += "│   "
		}
	}

	for i, child := range node.Children {
		buildTree(child, prefix, i == len(node.Children)-1, false, lines, renderedLines)
	}
}
