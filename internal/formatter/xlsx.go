package formatter

import (
	"fmt"

	"github.com/xuri/excelize/v2"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

// XLSXFormatter formats PRs as Excel XLSX
type XLSXFormatter struct{}

// NewXLSXFormatter creates a new XLSX formatter
func NewXLSXFormatter() *XLSXFormatter {
	return &XLSXFormatter{}
}

// Format formats pull requests as XLSX bytes
func (f *XLSXFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) ([]byte, error) {
	file := excelize.NewFile()
	defer func() {
		if err := file.Close(); err != nil {
			// Handle error if needed
		}
	}()

	sheetName := "Sheet1"

	// Set headers
	headers := []string{"REPO NAME", "PR NAME", "PR COMPLETION DATE", "PR URL", "PR LINK"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		file.SetCellValue(sheetName, cell, header)
	}

	// Set data
	for row, pr := range prs {
		webURL := pr.GetWebURL(options["org"], options["project"])

		// Column A: REPO NAME
		cellA, _ := excelize.CoordinatesToCellName(1, row+2)
		file.SetCellValue(sheetName, cellA, pr.Repository.Name)

		// Column B: PR NAME
		cellB, _ := excelize.CoordinatesToCellName(2, row+2)
		file.SetCellValue(sheetName, cellB, pr.Title)

		// Column C: PR COMPLETION DATE
		cellC, _ := excelize.CoordinatesToCellName(3, row+2)
		file.SetCellValue(sheetName, cellC, pr.FormatCompletionDate(dateFormat))

		// Column D: PR URL
		cellD, _ := excelize.CoordinatesToCellName(4, row+2)
		file.SetCellValue(sheetName, cellD, webURL)

		// Column E: PR LINK (as hyperlink)
		cellE, _ := excelize.CoordinatesToCellName(5, row+2)
		linkText := fmt.Sprintf("LINK TO PR (#%d)", pr.ID)
		file.SetCellValue(sheetName, cellE, linkText)

		// Add hyperlink to the cell
		if err := file.SetCellHyperLink(sheetName, cellE, webURL, "External"); err != nil {
			return nil, fmt.Errorf("failed to set hyperlink: %w", err)
		}
	}

	// Auto-fit columns
	columns := []string{"A", "B", "C", "D", "E"}
	for _, col := range columns {
		if err := file.SetColWidth(sheetName, col, col, 20); err != nil {
			return nil, fmt.Errorf("failed to set column width: %w", err)
		}
	}

	// Save to buffer
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write XLSX to buffer: %w", err)
	}

	return buffer.Bytes(), nil
}
