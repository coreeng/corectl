package tenant

import (
	"fmt"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
)

type Node struct {
	Tenant   *coretnt.Tenant
	Children []*Node
}

// Builds trees of tenants.
//
// Arguments:
//
//	cfg    Project configuration
//	root   Name of the tenant to start the tree from; "" to start from the top-level tenant
//
// Returns: A slice of pointers to the root nodes of the trees
func GetTenantTrees(cfg *config.Config, root string) ([]*Node, error) {
	tenants, err := coretnt.List(coretnt.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

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
	if root != "" {
		rootNode, exists := nodeMap[root]
		if !exists {
			return nil, fmt.Errorf("no tenant found with name '%s'", root)
		}
		rootNodes = append(rootNodes, rootNode)

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
