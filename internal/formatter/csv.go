package formatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

type CSVFormatter struct{}

func NewCSVFormatter() *CSVFormatter {
	return &CSVFormatter{}
}

func (f *CSVFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) (string, error) {
	var buf bytes.Buffer

	delimiter := ";"
	if d, ok := options["delimiter"]; ok && d != "" {
		delimiter = d
	}

	columnsStr := options["columns"]
	if columnsStr == "" {
		columnsStr = DefaultColumns
	}

	cols, err := ParseColumns(columnsStr)
	if err != nil {
		return "", err
	}

	org := options["org"]
	project := options["project"]

	headers := make([]string, len(cols))
	for i, col := range cols {
		headers[i] = col.Header
	}
	buf.WriteString(strings.Join(headers, delimiter) + "\n")

	for i, pr := range prs {
		row := make([]string, len(cols))
		for j, col := range cols {
			var value string
			if col.ID == "completed" && dateFormat != "" {
				value = pr.FormatCompletionDate(dateFormat)
			} else if col.ID == "created" && dateFormat != "" {
				if pr.CreationDate.IsZero() {
					value = "N/A"
				} else {
					value = pr.CreationDate.Format(dateFormat)
				}
			} else {
				value = col.GetValue(pr, i+1, org, project)
			}
			escapedValue := strings.ReplaceAll(value, `"`, `""`)
			row[j] = fmt.Sprintf(`"%s"`, escapedValue)
		}
		buf.WriteString(strings.Join(row, delimiter) + "\n")
	}

	return buf.String(), nil
}
