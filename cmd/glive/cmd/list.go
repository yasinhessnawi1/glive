package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  `List all projects managed by GLive.`,
	Run:   listProjects,
}

func listProjects(cmd *cobra.Command, args []string) {
	fmt.Println("📋 Projects:")
	fmt.Println("\nNo projects yet. Run 'glive <github-url>' to get started!")
	fmt.Println("\n⚠️  Project listing will be functional once Go is installed and the project is built.")
}
