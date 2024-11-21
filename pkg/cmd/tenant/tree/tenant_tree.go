package tree

import (
	"fmt"
	"strings"

	"github.com/coreeng/corectl/pkg/logger"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/tenant"
	"github.com/spf13/cobra"
)

type TenantListOpts struct {
	Streams userio.IOStreams
}

type Node struct {
	Value    string
	Children []*Node
}

type Tree struct {
	Root *Node
}

const (
	vertical   = "│   "
	branch     = "├── "
	lastBranch = "└── "
)

func NewTenantsTreeCmd(cfg *config.Config) *cobra.Command {
	opts := TenantListOpts{}
	tenantListCmd := &cobra.Command{
		Use:   "tree",
		Short: "List tenants in tree format",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Streams = userio.NewIOStreams(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
			)
			return run(cfg)
		},
	}

	config.RegisterStringParameterAsFlag(&cfg.Repositories.CPlatform, tenantListCmd.Flags())
	config.RegisterBoolParameterAsFlag(&cfg.Repositories.AllowDirty, tenantListCmd.Flags())

	return tenantListCmd
}

func run(cfg *config.Config) error {
	if !cfg.Repositories.AllowDirty.Value {
		if _, err := config.ResetConfigRepositoryState(&cfg.Repositories.CPlatform, false); err != nil {
			return err
		}
	}

	return printTreeView(cfg)
}

func NewNode(value string) *Node {
	return &Node{
		Value:    value,
		Children: make([]*Node, 0),
	}
}

func CreateTreeView(tenants []tenant.Tenant) *Tree {
	var rootTenant tenant.Tenant
	for _, t := range tenants {
		if t.Parent == "" || t.Name == "root" {
			rootTenant = t
			break
		}
	}

	root := NewNode(rootTenant.Name)
	tree := &Tree{Root: root}

	nodeMap := make(map[string]*Node)
	nodeMap[root.Value] = root

	for len(nodeMap) < len(tenants) {
		for _, tenant := range tenants {
			if tenant.Name == rootTenant.Name || nodeMap[tenant.Name] != nil {
				continue
			}

			if parentNode, exists := nodeMap[tenant.Parent]; exists {
				newNode := NewNode(tenant.Name)
				parentNode.Children = append(parentNode.Children, newNode)
				nodeMap[tenant.Name] = newNode
			}
		}
	}

	return tree
}

func (t *Tree) String() string {
	if t.Root == nil {
		return "Empty tree"
	}

	var sb strings.Builder
	sb.WriteString(t.Root.Value + "\n")
	t.writeChildren(&sb, t.Root.Children, "")

	return sb.String()
}

func (t *Tree) writeChildren(sb *strings.Builder, nodes []*Node, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		if isLast {
			sb.WriteString(prefix + lastBranch + node.Value + "\n")
		} else {
			sb.WriteString(prefix + branch + node.Value + "\n")
		}

		if len(node.Children) > 0 {
			newPrefix := prefix
			if isLast {
				newPrefix += "    "
			} else {
				newPrefix += vertical
			}
			t.writeChildren(sb, node.Children, newPrefix)
		}
	}
}

func printTreeView(cfg *config.Config) error {

	tenants, err := tenant.List(tenant.DirFromCPlatformPath(cfg.Repositories.CPlatform.Value))
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}
	tenants = append(tenants, tenant.Tenant{Name: "root"})
	tree := CreateTreeView(tenants)
	logger.Debug(tree.String())
	fmt.Println(tree.String())

	return nil
}
