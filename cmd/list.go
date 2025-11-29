package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CodellaSoftware/azure-pr-cli/internal/client"
	"github.com/CodellaSoftware/azure-pr-cli/internal/config"
	"github.com/CodellaSoftware/azure-pr-cli/internal/formatter"
	"github.com/spf13/cobra"
)

var (
	organization string
	project      string
	repository   string
	pat          string
	fromDate     string
	toDate       string
	status       string
	outputFormat string
	outputFile   string
	dateFormat   string
	delimiter    string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests from Azure DevOps",
	Long: `Fetch and display pull requests from an Azure DevOps repository.

By default, this command fetches completed PRs from the current month.
You can customize the date range and status filters.`,
	Example: `  # List PRs from multiple repositories and save as XLSX (default behavior)
  azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3

  # List PRs from a single repository with custom date range
  azure-pr-cli list -o myorg -p myproject -r myrepo --from 2024-01-01 --to 2024-01-31

  # List all PRs (any status) from multiple repositories
  azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --status all

  # Output as table to console
  azure-pr-cli list -o myorg -p myproject -r myrepo --format table

  # Output as JSON
  azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --format json

  # Save as CSV file
  azure-pr-cli list -o myorg -p myproject -r myrepo --output-file prs.csv

  # Save as Excel XLSX file (with hyperlinks)
  azure-pr-cli list -o myorg -p myproject -r myrepo --output-file prs.xlsx

  # Custom date format
  azure-pr-cli list -o myorg -p myproject -r myrepo --date-format "2006-01-02"

  # CSV with custom delimiter
  azure-pr-cli list -o myorg -p myproject -r myrepo --format csv --delimiter ","`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Required flags
	listCmd.Flags().StringVarP(&organization, "organization", "o", "", "Azure DevOps organization (or set AZURE_DEVOPS_ORG)")
	listCmd.Flags().StringVarP(&project, "project", "p", "", "Azure DevOps project (or set AZURE_DEVOPS_PROJECT)")
	listCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository name(s) (comma-separated for multiple repositories)")
	if err := listCmd.MarkFlagRequired("repository"); err != nil {
		panic(err)
	}

	// Authentication
	listCmd.Flags().StringVar(&pat, "pat", "", "Personal Access Token (or set AZURE_DEVOPS_PAT)")

	// Optional filters
	listCmd.Flags().StringVar(&fromDate, "from", "", "Start date (YYYY-MM-DD), defaults to start of current month")
	listCmd.Flags().StringVar(&toDate, "to", "", "End date (YYYY-MM-DD), defaults to now")
	listCmd.Flags().StringVar(&status, "status", "completed", "PR status filter: active, completed, abandoned, all")

	// Output options
	listCmd.Flags().StringVarP(&outputFormat, "format", "f", "xlsx", "Output format: table, json, csv, xlsx (default: saves to pull-requests.xlsx)")
	listCmd.Flags().StringVar(&outputFile, "output-file", "", "Save output to specified file (format determined by extension, defaults to xlsx)")
	listCmd.Flags().StringVar(&dateFormat, "date-format", "02.01.2006", "Date format for completion dates (Go time format)")
	listCmd.Flags().StringVar(&delimiter, "delimiter", ";", "CSV delimiter character (only used with CSV format)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(organization, project, repository, pat)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Organization: %s\n", cfg.Organization)
		fmt.Fprintf(os.Stderr, "Project: %s\n", cfg.Project)
		fmt.Fprintf(os.Stderr, "Repositories: %s\n", strings.Join(cfg.Repositories, ", "))
	}

	// Parse date range
	from, to, err := parseDateRange(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("date parsing error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Date range: %s to %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
		fmt.Fprintf(os.Stderr, "Status filter: %s\n", status)
	}

	// Create Azure DevOps client
	azureClient := client.NewAzureDevOpsClient(cfg)

	// Fetch pull requests
	if verbose {
		fmt.Fprintf(os.Stderr, "Fetching pull requests...\n")
	}

	prs, err := azureClient.GetPullRequests(from, to, status)
	if err != nil {
		return fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d pull requests\n", len(prs))
	}

	// Determine output format
	actualFormat := outputFormat
	if outputFile != "" {
		// Determine format from file extension
		if strings.HasSuffix(outputFile, ".xlsx") {
			actualFormat = "xlsx"
		} else if strings.HasSuffix(outputFile, ".csv") {
			actualFormat = "csv"
		} else if strings.HasSuffix(outputFile, ".json") {
			actualFormat = "json"
		} else if outputFormat == "table" {
			// Default to XLSX for file output when no specific format requested
			actualFormat = "xlsx"
		}
		// If extension doesn't match and format was explicitly set, keep it
	}

	// Validate delimiter is only used with CSV
	if delimiter != ";" && actualFormat != "csv" {
		return fmt.Errorf("--delimiter can only be used with CSV format")
	}

	// Handle XLSX format specially since it returns bytes
	if actualFormat == "xlsx" {
		xlsxFormatter := formatter.NewXLSXFormatter()
		options := map[string]string{
			"org":     cfg.Organization,
			"project": cfg.Project,
		}

		xlsxData, err := xlsxFormatter.Format(prs, dateFormat, options)
		if err != nil {
			return fmt.Errorf("XLSX formatting error: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, xlsxData, 0644); err != nil {
				return fmt.Errorf("failed to save XLSX file: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "XLSX saved to %s\n", outputFile)
			}
		} else {
			// Use default filename for XLSX
			defaultFile := "pull-requests.xlsx"
			if err := os.WriteFile(defaultFile, xlsxData, 0644); err != nil {
				return fmt.Errorf("failed to save XLSX file: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "XLSX saved to %s\n", defaultFile)
			}
		}
		return nil
	}

	var formatterInstance formatter.Formatter
	var options map[string]string

	switch actualFormat {
	case "json":
		formatterInstance = formatter.NewJSONFormatter()
		options = map[string]string{}
	case "csv":
		formatterInstance = formatter.NewCSVFormatter()
		options = map[string]string{
			"delimiter": delimiter,
			"org":       cfg.Organization,
			"project":   cfg.Project,
		}
	case "table":
		formatterInstance = formatter.NewTableFormatter()
		options = map[string]string{
			"org":     cfg.Organization,
			"project": cfg.Project,
		}
	default:
		return fmt.Errorf("unsupported output format: %s", actualFormat)
	}

	output, err := formatterInstance.Format(prs, dateFormat, options)
	if err != nil {
		return fmt.Errorf("formatting error: %w", err)
	}

	if outputFile != "" {
		// Save to file
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Output saved to %s\n", outputFile)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}

func parseDateRange(from, to string) (time.Time, time.Time, error) {
	now := time.Now()

	var fromTime, toTime time.Time
	var err error

	// Parse 'from' date
	if from == "" {
		// Default to start of current month
		fromTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		fromTime, err = time.Parse("2006-01-02", from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date format: %w", err)
		}
	}

	// Parse 'to' date
	if to == "" {
		// Default to now
		toTime = now
	} else {
		toTime, err = time.Parse("2006-01-02", to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date format: %w", err)
		}
		// Set to end of day
		toTime = time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 23, 59, 59, 0, time.UTC)
	}

	if fromTime.After(toTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("from date must be before to date")
	}

	return fromTime, toTime, nil
}
