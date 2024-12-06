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

// Builds a tree of tenants.
//
// Arguments:
//
//	cfg    Project configuration
//	root   Name of the tenant to start the tree from; "" to start from the top-level tenant
//
// Returns: A pointer to the root node of the tree
func GetTenantTree(cfg *config.Config, root string) (*Node, error) {
	tenants, err := coretnt.List(coretnt.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	if root == "" {
		root = coretnt.RootName
	}

	// Build a map of tenants indexed by name for faster access
	nodeMap := make(map[string]*Node)
	for _, tenant := range tenants {
		nodeMap[tenant.Name] = &Node{
			Tenant:   &tenant,
			Children: []*Node{},
		}
	}

	var rootNode *Node
	for _, tenant := range tenants {
		if tenant.Parent == root {
			rootNode = nodeMap[tenant.Name]
		} else {
			parent, exists := nodeMap[tenant.Parent]
			if exists {
				parent.Children = append(parent.Children, nodeMap[tenant.Name])
			}
		}
	}

	return rootNode, nil
}

func RenderTenantTree(root *Node) []string {
	var output []string
	buildTree(root, "", true, &output)
	return output
}

func buildTree(node *Node, prefix string, isLastChild bool, output *[]string) {
	if node == nil {
		return
	}

	var connector string
	if node.Tenant.Parent != coretnt.RootName {
		if isLastChild {
			connector = "└── "
		} else {
			connector = "├── "
		}
	}

	out := fmt.Sprintf("%s%s%s", prefix, connector, node.Tenant.Name)
	*output = append(*output, out)

	childPrefix := prefix
	if node.Tenant.Parent != coretnt.RootName {
		if isLastChild {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		buildTree(child, childPrefix, i == len(node.Children)-1, output)
	}
}
