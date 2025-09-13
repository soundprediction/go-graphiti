package graphiti

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version info - these would be set by build flags in production
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print version, commit, and build date information for Go-Graphiti",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Go-Graphiti\n")
		fmt.Printf("Version:    %s\n", version)
		fmt.Printf("Commit:     %s\n", commit)
		fmt.Printf("Build Date: %s\n", buildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}