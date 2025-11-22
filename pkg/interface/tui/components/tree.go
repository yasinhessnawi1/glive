package components

import (
	"strings"
)

// TreeNode represents a node in a tree
type TreeNode struct {
	Label    string
	Children []*TreeNode
	Status   string // empty, success, running, warning, error
}

// NewTreeNode creates a new tree node
func NewTreeNode(label string) *TreeNode {
	return &TreeNode{
		Label:    label,
		Children: make([]*TreeNode, 0),
	}
}

// AddChild adds a child node
func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
}

// Render renders the tree starting from this node
func (n *TreeNode) Render() string {
	return n.renderNode("", true)
}

// renderNode renders a node and its children
func (n *TreeNode) renderNode(prefix string, isLast bool) string {
	var result strings.Builder

	// Current node
	connector := "├─ "
	if isLast {
		connector = "└─ "
	}

	statusIcon := n.getStatusIcon()
	result.WriteString(prefix + connector + statusIcon + " " + n.Label + "\n")

	// Children
	childPrefix := prefix
	if isLast {
		childPrefix += "   "
	} else {
		childPrefix += "│  "
	}

	for i, child := range n.Children {
		isLastChild := i == len(n.Children)-1
		result.WriteString(child.renderNode(childPrefix, isLastChild))
	}

	return result.String()
}

// getStatusIcon returns the icon for the node status
func (n *TreeNode) getStatusIcon() string {
	switch n.Status {
	case "success", "completed":
		return "✓"
	case "running", "pending":
		return "⟳"
	case "warning", "recovered":
		return "⚠"
	case "failed", "error":
		return "✗"
	default:
		return ""
	}
}

// Tree represents a tree structure
type Tree struct {
	Root   *TreeNode
	Styles interface{} // For future styling support
}

// NewTree creates a new tree
func NewTree(root *TreeNode) *Tree {
	return &Tree{
		Root: root,
	}
}

// Render renders the entire tree
func (t *Tree) Render() string {
	if t.Root == nil {
		return ""
	}
	return t.Root.Render()
}

