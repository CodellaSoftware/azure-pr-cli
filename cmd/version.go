package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display version information for azure-pr-cli`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("azure-pr-cli %s\n", Version)
		if verbose {
			fmt.Printf("Git Commit: %s\n", GitCommit)
			fmt.Printf("Build Date: %s\n", BuildDate)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
