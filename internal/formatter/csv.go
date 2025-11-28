package formatter

import (
	"bytes"
	"encoding/csv"

	"github.com/yourusername/azure-pr-cli/internal/models"
)

// CSVFormatter formats PRs as CSV
type CSVFormatter struct{}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter() *CSVFormatter {
	return &CSVFormatter{}
}

// Format formats pull requests as CSV
func (f *CSVFormatter) Format(prs []models.PullRequest) (string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"REPO NAME", "PR NAME", "PR COMPLETION DATE", "PR URL"}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	// Write data
	for _, pr := range prs {
		record := []string{
			pr.Repository.Name,
			pr.Title,
			pr.FormatCompletionDate(),
			pr.URL,
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
