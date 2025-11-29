package formatter

import (
	"fmt"

	"github.com/xuri/excelize/v2"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

type XLSXFormatter struct{}

func NewXLSXFormatter() *XLSXFormatter {
	return &XLSXFormatter{}
}

func (f *XLSXFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
	file := excelize.NewFile()
	defer func() {
		if err := file.Close(); err != nil {
		}
	}()

	sheetName := "Sheet1"

	headers := []string{"REPO NAME", "PR NAME", "PR COMPLETION DATE", "PR URL", "PR LINK"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		file.SetCellValue(sheetName, cell, header)
	}

	for row, pr := range prs {
		webURL := pr.GetWebURL(options["org"], options["project"])

		cellA, _ := excelize.CoordinatesToCellName(1, row+2)
		file.SetCellValue(sheetName, cellA, pr.Repository.Name)

		cellB, _ := excelize.CoordinatesToCellName(2, row+2)
		file.SetCellValue(sheetName, cellB, pr.Title)

		cellC, _ := excelize.CoordinatesToCellName(3, row+2)
		file.SetCellValue(sheetName, cellC, pr.FormatCompletionDate(dateFormat))

		cellD, _ := excelize.CoordinatesToCellName(4, row+2)
		file.SetCellValue(sheetName, cellD, webURL)

		cellE, _ := excelize.CoordinatesToCellName(5, row+2)
		linkText := fmt.Sprintf("LINK TO PR (#%d)", pr.ID)
		file.SetCellValue(sheetName, cellE, linkText)

		if err := file.SetCellHyperLink(sheetName, cellE, webURL, "External"); err != nil {
			return nil, fmt.Errorf("failed to set hyperlink: %w", err)
		}
	}

	columns := []string{"A", "B", "C", "D", "E"}
	for _, col := range columns {
		if err := file.SetColWidth(sheetName, col, col, 20); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write XLSX to buffer: %w", err)
	}

	return buffer.Bytes(), nil
}
