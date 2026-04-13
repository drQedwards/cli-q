package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/supermodeltools/cli/internal/build"
	"github.com/supermodeltools/cli/internal/update"
)

func init() {
	var checkOnly bool

	c := &cobra.Command{
		Use:   "update",
		Short: "Update supermodel to the latest release",
		Long: `Checks for a newer release on GitHub and, if found, downloads and installs it.

Use --check to only print the latest available version without installing.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkOnly {
				latest, err := update.Check()
				if err != nil {
					return fmt.Errorf("check for updates: %w", err)
				}
				current := strings.TrimPrefix(build.Version, "v")
				latestClean := strings.TrimPrefix(latest, "v")
				if current == latestClean {
					fmt.Printf("supermodel %s is up to date\n", build.Version)
				} else {
					fmt.Printf("current: %s  →  latest: %s\n", build.Version, latest)
					fmt.Println("Run `supermodel update` to install.")
				}
				return nil
			}

			fmt.Println("Checking for updates…")
			updated, err := update.Run()
			if err != nil {
				return err
			}
			if updated {
				fmt.Printf("Updated to the latest version. Run `supermodel version` to confirm.\n")
			} else {
				fmt.Printf("supermodel %s is already up to date.\n", build.Version)
			}
			return nil
		},
	}

	c.Flags().BoolVar(&checkOnly, "check", false, "check for updates without installing")
	rootCmd.AddCommand(c)

	// Allow `update` to run without an API key (it's independent of auth)
	noConfigCommands["update"] = true
}
