package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [project-id]",
	Short: "Show status of projects",
	Long:  `Display the status of running or completed projects.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   showStatus,
}

func showStatus(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// Show all projects
		fmt.Println("📊 All Projects:")
		fmt.Println("\nNo projects yet. Run 'glive <github-url>' to get started!")
	} else {
		// Show specific project
		projectID := args[0]
		fmt.Printf("📊 Project Status: %s\n", projectID)
		fmt.Println("\n⚠️  Status tracking will be functional once Go is installed and the project is built.")
	}
}
