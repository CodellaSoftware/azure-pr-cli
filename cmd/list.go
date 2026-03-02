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
	columns      string
	listColumns  bool
	templatePath string
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
  azure-pr-cli list -o myorg -p myproject -r myrepo --format csv --delimiter ","

  # Custom columns
  azure-pr-cli list -o myorg -p myproject -r myrepo --columns "index,repo,title,author,status,completed,url"

  # List available columns
  azure-pr-cli list --list-columns

  # Output as DOCX using a template
  azure-pr-cli list -o myorg -p myproject -r myrepo --format docx --template template.docx

  # Output as DOCX with custom output file
  azure-pr-cli list -o myorg -p myproject -r myrepo --template template.docx --output-file report.docx`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&organization, "organization", "o", "", "Azure DevOps organization (or AZURE_DEVOPS_ORG)")
	listCmd.Flags().StringVarP(&project, "project", "p", "", "Azure DevOps project (or AZURE_DEVOPS_PROJECT)")
	listCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository name(s), comma-separated (or AZURE_DEVOPS_REPOSITORIES)")

	listCmd.Flags().StringVar(&pat, "pat", "", "Personal Access Token (or AZURE_DEVOPS_PAT)")

	listCmd.Flags().StringVar(&fromDate, "from", "", "Start date (YYYY-MM-DD), defaults to start of current month")
	listCmd.Flags().StringVar(&toDate, "to", "", "End date (YYYY-MM-DD), defaults to end of current month")
	listCmd.Flags().StringVar(&status, "status", "", "PR status filter: active, completed, abandoned, all (or AZURE_DEVOPS_STATUS, default: completed)")

	listCmd.Flags().StringVarP(&outputFormat, "format", "f", "", "Output format: table, json, csv, xlsx, docx (or AZURE_DEVOPS_FORMAT, default: xlsx)")
	listCmd.Flags().StringVar(&outputFile, "output-file", "", "Save output to file (or AZURE_DEVOPS_OUTPUT_FILE)")
	listCmd.Flags().StringVar(&dateFormat, "date-format", "", "Date format for completion dates (or AZURE_DEVOPS_DATE_FORMAT, default: 02.01.2006)")
	listCmd.Flags().StringVar(&delimiter, "delimiter", "", "CSV delimiter character (or AZURE_DEVOPS_DELIMITER, default: ;)")
	listCmd.Flags().StringVar(&columns, "columns", "", "Columns to display, comma-separated (or AZURE_DEVOPS_COLUMNS, default: index,repo,title,completed,url)")
	listCmd.Flags().BoolVar(&listColumns, "list-columns", false, "List available columns and exit")
	listCmd.Flags().StringVar(&templatePath, "template", "", "Path to .docx template file (required when --format docx, or AZURE_DEVOPS_TEMPLATE)")
}

func runList(cmd *cobra.Command, args []string) error {
	if listColumns {
		fmt.Println(formatter.ListColumns())
		return nil
	}

	actualStatus := config.GetValueOrEnvWithDefault(status, "AZURE_DEVOPS_STATUS", "completed")
	actualFormat := config.GetValueOrEnvWithDefault(outputFormat, "AZURE_DEVOPS_FORMAT", "xlsx")
	actualOutputFile := config.GetValueOrEnv(outputFile, "AZURE_DEVOPS_OUTPUT_FILE")
	actualDateFormat := config.GetValueOrEnvWithDefault(dateFormat, "AZURE_DEVOPS_DATE_FORMAT", "02.01.2006")
	actualDelimiter := config.GetValueOrEnvWithDefault(delimiter, "AZURE_DEVOPS_DELIMITER", ";")
	actualColumns := config.GetValueOrEnvWithDefault(columns, "AZURE_DEVOPS_COLUMNS", formatter.DefaultColumns)

	cfg, err := config.LoadConfig(organization, project, repository, pat)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Organization: %s\n", cfg.Organization)
		fmt.Fprintf(os.Stderr, "Project: %s\n", cfg.Project)
		fmt.Fprintf(os.Stderr, "Repositories: %s\n", strings.Join(cfg.Repositories, ", "))
		fmt.Fprintf(os.Stderr, "Columns: %s\n", actualColumns)
	}

	_, err = formatter.ParseColumns(actualColumns)
	if err != nil {
		return fmt.Errorf("column configuration error: %w", err)
	}

	from, to, err := parseDateRange(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("date parsing error: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Date range: %s to %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
		fmt.Fprintf(os.Stderr, "Status filter: %s\n", actualStatus)
	}

	azureClient := client.NewAzureDevOpsClient(cfg)

	if verbose {
		fmt.Fprintf(os.Stderr, "Fetching pull requests...\n")
	}

	prs, err := azureClient.GetPullRequests(from, to, actualStatus)
	if err != nil {
		return fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d pull requests\n", len(prs))
	}

	if actualOutputFile != "" {
		if strings.HasSuffix(actualOutputFile, ".xlsx") {
			actualFormat = "xlsx"
		} else if strings.HasSuffix(actualOutputFile, ".csv") {
			actualFormat = "csv"
		} else if strings.HasSuffix(actualOutputFile, ".json") {
			actualFormat = "json"
		} else if strings.HasSuffix(actualOutputFile, ".docx") {
			actualFormat = "docx"
		}
	}

	if actualFormat == "docx" {
		actualTemplatePath := config.GetValueOrEnv(templatePath, "AZURE_DEVOPS_TEMPLATE")
		if actualTemplatePath == "" {
			return fmt.Errorf("--template flag is required when using --format docx")
		}
		templatePath = actualTemplatePath
	}

	if actualDelimiter != ";" && actualFormat != "csv" {
		return fmt.Errorf("--delimiter can only be used with CSV format")
	}

	if actualFormat == "xlsx" {
		xlsxFormatter := formatter.NewXLSXFormatter()
		options := map[string]string{
			"org":     cfg.Organization,
			"project": cfg.Project,
			"columns": actualColumns,
		}

		xlsxData, err := xlsxFormatter.Format(prs, actualDateFormat, options)
		if err != nil {
			return fmt.Errorf("XLSX formatting error: %w", err)
		}

		if actualOutputFile != "" {
			if err := os.WriteFile(actualOutputFile, xlsxData, 0644); err != nil {
				return fmt.Errorf("failed to save XLSX file: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "XLSX saved to %s\n", actualOutputFile)
			}
		} else {
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

	if actualFormat == "docx" {
		lastDayOfMonth := time.Date(to.Year(), to.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		docxFormatter := formatter.NewDOCXFormatter()
		options := map[string]string{
			"template":           templatePath,
			"org":                cfg.Organization,
			"project":            cfg.Project,
			"dateFrom":           from.Format(actualDateFormat),
			"dateTo":             to.Format(actualDateFormat),
			"reportCreationDate": lastDayOfMonth.Format(actualDateFormat),
		}

		docxData, err := docxFormatter.Format(prs, actualDateFormat, options)
		if err != nil {
			return fmt.Errorf("DOCX formatting error: %w", err)
		}

		outputTarget := actualOutputFile
		if outputTarget == "" {
			outputTarget = "pull-requests.docx"
		}

		if err := os.WriteFile(outputTarget, docxData, 0644); err != nil {
			return fmt.Errorf("failed to save DOCX file: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "DOCX saved to %s\n", outputTarget)
		}

		return nil
	}

	var formatterInstance formatter.Formatter
	var options map[string]string

	switch actualFormat {
	case "json":
		formatterInstance = formatter.NewJSONFormatter()
		options = map[string]string{
			"columns": actualColumns,
		}
	case "csv":
		formatterInstance = formatter.NewCSVFormatter()
		options = map[string]string{
			"delimiter": actualDelimiter,
			"org":       cfg.Organization,
			"project":   cfg.Project,
			"columns":   actualColumns,
		}
	case "table":
		formatterInstance = formatter.NewTableFormatter()
		options = map[string]string{
			"org":     cfg.Organization,
			"project": cfg.Project,
			"columns": actualColumns,
		}
	default:
		return fmt.Errorf("unsupported output format: %s", actualFormat)
	}

	output, err := formatterInstance.Format(prs, actualDateFormat, options)
	if err != nil {
		return fmt.Errorf("formatting error: %w", err)
	}

	if actualOutputFile != "" {
		if err := os.WriteFile(actualOutputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Output saved to %s\n", actualOutputFile)
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

	if from == "" {
		fromTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		fromTime, err = time.Parse("2006-01-02", from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date format: %w", err)
		}
	}

	if to == "" {
		nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		toTime = nextMonth.Add(-time.Second)
	} else {
		toTime, err = time.Parse("2006-01-02", to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date format: %w", err)
		}
		toTime = time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 23, 59, 59, 0, time.UTC)
	}

	if fromTime.After(toTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("from date must be before to date")
	}

	return fromTime, toTime, nil
}
