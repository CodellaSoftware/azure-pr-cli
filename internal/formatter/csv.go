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

func (f *CSVFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
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
		return nil, err
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
			value := FormatColumnValue(pr, col, i+1, dateFormat, org, project)
			escapedValue := strings.ReplaceAll(value, `"`, `""`)
			row[j] = fmt.Sprintf(`"%s"`, escapedValue)
		}
		buf.WriteString(strings.Join(row, delimiter) + "\n")
	}

	return buf.Bytes(), nil
}
