package formatter

import (
	"encoding/json"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
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

	rows := make([]map[string]string, len(prs))
	for i, pr := range prs {
		row := make(map[string]string, len(cols))
		for _, col := range cols {
			row[col.ID] = FormatColumnValue(pr, col, i+1, dateFormat, org, project)
		}
		rows[i] = row
	}

	return json.MarshalIndent(rows, "", "  ")
}
