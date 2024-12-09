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
//		from      Name of the tenant to start the tree from; "" to start from the top-level tenant
//
// Returns: A slice of pointers to the root nodes of the trees. If `from` has been provided, the slice will have only one item.
func GetTenantTrees(tenants []coretnt.Tenant, from string) ([]*Node, error) {
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

	rootNodes := []*Node{}
	if from != "" {
		fromNode, exists := nodeMap[from]
		if !exists {
			return nil, fmt.Errorf("no tenant found with name '%s'", from)
		}
		rootNodes = append(rootNodes, fromNode)

	} else {
		for _, tenant := range tenants {
			if tenant.Parent == coretnt.RootName {
				node, exists := nodeMap[tenant.Name]
				if !exists {
					panic("Internal inconsistency, this is a bug, please fix it")
				}
				rootNodes = append(rootNodes, node)
			}
		}
	}

	return rootNodes, nil
}

func RenderTenantTree(root *Node) []string {
	var lines []string
	buildTree(root, "", true, true, &lines)
	return lines
}

func buildTree(node *Node, prefix string, isLastChild bool, isRoot bool, lines *[]string) {
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

	out := fmt.Sprintf("%s%s%s", prefix, connector, node.Tenant.Name)
	*lines = append(*lines, out)

	if !isRoot {
		if isLastChild {
			prefix += "    "
		} else {
			prefix += "│   "
		}
	}

	for i, child := range node.Children {
		buildTree(child, prefix, i == len(node.Children)-1, false, lines)
	}
}
