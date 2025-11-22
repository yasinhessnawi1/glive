package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <project-id>",
	Short: "Clean up a project",
	Long:  `Remove a project and rollback all changes made during setup.`,
	Args:  cobra.ExactArgs(1),
	Run:   cleanup,
}

func cleanup(cmd *cobra.Command, args []string) {
	projectID := args[0]

	fmt.Printf("🧹 Cleaning up project: %s\n", projectID)
	fmt.Println("\n⚠️  Cleanup will be functional once Go is installed and the project is built.")
	fmt.Println("\nThis will:")
	fmt.Println("  - Remove cloned repository")
	fmt.Println("  - Rollback installed dependencies (if possible)")
	fmt.Println("  - Remove project state")
}
