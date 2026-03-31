package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "supermodel",
	Short: "Give your AI coding agent a map of your codebase",
	Long: `Supermodel connects AI coding agents to the Supermodel API,
providing call graphs, dead code detection, and blast radius analysis.

See https://supermodeltools.com for documentation.`,
	SilenceUsage: true,
}

// Execute is the entry point called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
