package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   showVersion,
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("glive version %s (commit: %s, built: %s)\n", version, commit, date)
}
