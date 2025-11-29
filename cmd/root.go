package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   = "1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"

	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "azure-pr-cli",
	Short: "Azure DevOps Pull Request CLI Tool",
	Long: `A professional CLI tool for fetching and displaying Pull Requests 
from Azure DevOps repositories.

This tool helps you quickly view your PRs in a clean, tabular format
with support for filtering by date ranges and various output formats.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
