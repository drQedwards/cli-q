package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/supermodeltools/cli/internal/build"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("supermodel %s (%s, %s)\n", build.Version, build.Commit, build.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = build.Version
}
