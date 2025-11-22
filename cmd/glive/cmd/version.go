package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.1.0-alpha"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   showVersion,
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("GLive version %s\n", version)
	fmt.Println("GitHub to Live - Automatically run any GitHub project")
}
