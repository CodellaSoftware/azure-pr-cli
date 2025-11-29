package formatter

import (
	"bytes"
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

type TableFormatter struct{}

func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

func (f *TableFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) (string, error) {
	if len(prs) == 0 {
		return "No pull requests found for the specified criteria.\n", nil
	}

	var buf bytes.Buffer

	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"REPO NAME", "PR NAME", "PR COMPLETION DATE", "PR URL"})
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("|")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)

	for _, pr := range prs {
		completionDate := pr.FormatCompletionDate(dateFormat)

		table.Append([]string{
			pr.Repository.Name,
			pr.Title,
			completionDate,
			pr.GetWebURL(options["org"], options["project"]),
		})
	}

	table.Render()

	buf.WriteString(fmt.Sprintf("\nTotal PRs: %d\n", len(prs)))

	return buf.String(), nil
}
