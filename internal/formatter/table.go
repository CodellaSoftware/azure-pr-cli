package formatter

import (
	"bytes"
	"fmt"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/olekukonko/tablewriter"
)

type TableFormatter struct{}

func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

func (f *TableFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
	if len(prs) == 0 {
		return []byte("No pull requests found for the specified criteria.\n"), nil
	}

	columnsStr := options["columns"]
	if columnsStr == "" {
		columnsStr = DefaultColumns
	}

	cols, err := ParseColumns(columnsStr)
	if err != nil {
		return nil, err
	}

	org := options["org"]
	project := options["project"]

	var buf bytes.Buffer

	table := tablewriter.NewWriter(&buf)

	headers := make([]string, len(cols))
	for i, col := range cols {
		headers[i] = col.Header
	}
	table.SetHeader(headers)

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

	for i, pr := range prs {
		row := make([]string, len(cols))
		for j, col := range cols {
			row[j] = FormatColumnValue(pr, col, i+1, dateFormat, org, project)
		}
		table.Append(row)
	}

	table.Render()

	buf.WriteString(fmt.Sprintf("\nTotal PRs: %d\n", len(prs)))

	return buf.Bytes(), nil
}
