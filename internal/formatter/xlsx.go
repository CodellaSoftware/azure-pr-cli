package formatter

import (
	"fmt"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/xuri/excelize/v2"
)

type XLSXFormatter struct{}

func NewXLSXFormatter() *XLSXFormatter {
	return &XLSXFormatter{}
}

func (f *XLSXFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
	file := excelize.NewFile()
	defer func() {
		_ = file.Close()
	}()

	sheetName := "Sheet1"

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

	for colIdx, col := range cols {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for header: %w", err)
		}
		if err := file.SetCellValue(sheetName, cell, col.Header); err != nil {
			return nil, fmt.Errorf("failed to set header cell: %w", err)
		}
	}

	for rowIdx, pr := range prs {
		for colIdx, col := range cols {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return nil, fmt.Errorf("failed to get cell name: %w", err)
			}

			value := FormatColumnValue(pr, col, rowIdx+1, dateFormat, org, project)

			if err := file.SetCellValue(sheetName, cell, value); err != nil {
				return nil, fmt.Errorf("failed to set cell value: %w", err)
			}

			if col.ID == "link" {
				webURL := pr.GetWebURL(org, project)
				if err := file.SetCellHyperLink(sheetName, cell, webURL, "External"); err != nil {
					return nil, fmt.Errorf("failed to set hyperlink: %w", err)
				}
			}
		}
	}

	for colIdx := range cols {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get column name: %w", err)
		}
		if err := file.SetColWidth(sheetName, colName, colName, 20); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write XLSX to buffer: %w", err)
	}

	return buffer.Bytes(), nil
}
