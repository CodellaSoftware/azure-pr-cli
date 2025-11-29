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
		if err := file.Close(); err != nil {
		}
	}()

	sheetName := "Sheet1"

	headers := []string{"#", "REPO NAME", "PR NAME", "PR COMPLETION DATE", "PR URL", "PR LINK"}
	for col, header := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for header: %w", err)
		}
		if err := file.SetCellValue(sheetName, cell, header); err != nil {
			return nil, fmt.Errorf("failed to set header cell: %w", err)
		}
	}

	for row, pr := range prs {
		webURL := pr.GetWebURL(options["org"], options["project"])

		// Column A: #
		cellA, err := excelize.CoordinatesToCellName(1, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for index: %w", err)
		}
		if err := file.SetCellValue(sheetName, cellA, row+1); err != nil {
			return nil, fmt.Errorf("failed to set index cell: %w", err)
		}

		// Column B: REPO NAME
		cellB, err := excelize.CoordinatesToCellName(2, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for repo: %w", err)
		}
		if err := file.SetCellValue(sheetName, cellB, pr.Repository.Name); err != nil {
			return nil, fmt.Errorf("failed to set repo name cell: %w", err)
		}

		// Column C: PR NAME
		cellC, err := excelize.CoordinatesToCellName(3, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for title: %w", err)
		}
		if err := file.SetCellValue(sheetName, cellC, pr.Title); err != nil {
			return nil, fmt.Errorf("failed to set PR title cell: %w", err)
		}

		// Column D: PR COMPLETION DATE
		cellD, err := excelize.CoordinatesToCellName(4, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for date: %w", err)
		}
		if err := file.SetCellValue(sheetName, cellD, pr.FormatCompletionDate(dateFormat)); err != nil {
			return nil, fmt.Errorf("failed to set completion date cell: %w", err)
		}

		// Column E: PR URL
		cellE, err := excelize.CoordinatesToCellName(5, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for URL: %w", err)
		}
		if err := file.SetCellValue(sheetName, cellE, webURL); err != nil {
			return nil, fmt.Errorf("failed to set URL cell: %w", err)
		}

		// Column F: PR LINK
		cellF, err := excelize.CoordinatesToCellName(6, row+2)
		if err != nil {
			return nil, fmt.Errorf("failed to get cell name for link: %w", err)
		}
		linkText := fmt.Sprintf("LINK TO PR (#%d)", pr.ID)
		if err := file.SetCellValue(sheetName, cellF, linkText); err != nil {
			return nil, fmt.Errorf("failed to set link cell: %w", err)
		}

		if err := file.SetCellHyperLink(sheetName, cellF, webURL, "External"); err != nil {
			return nil, fmt.Errorf("failed to set hyperlink: %w", err)
		}
	}

	columns := []string{"A", "B", "C", "D", "E", "F"}
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
