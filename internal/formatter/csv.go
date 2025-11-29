package formatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yourusername/azure-pr-cli/internal/models"
)

// CSVFormatter formats PRs as CSV
type CSVFormatter struct{}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter() *CSVFormatter {
	return &CSVFormatter{}
}

// Format formats pull requests as CSV
func (f *CSVFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) (string, error) {
	var buf bytes.Buffer

	// Get delimiter from options, default to ";"
	delimiter := ";"
	if d, ok := options["delimiter"]; ok && d != "" {
		delimiter = d
	}

	// Write header manually to ensure consistent formatting
	header := fmt.Sprintf("REPO NAME%sPR NAME%sPR COMPLETION DATE%sPR URL%sPR LINK\n", delimiter, delimiter, delimiter, delimiter)
	buf.WriteString(header)

	// Write data rows manually
	for _, pr := range prs {
		webURL := pr.GetWebURL(options["org"], options["project"])
		// Escape quotes in title for CSV
		escapedTitle := strings.ReplaceAll(pr.Title, `"`, `""`)
		escapedRepo := strings.ReplaceAll(pr.Repository.Name, `"`, `""`)

		// For the link field, use plain text
		linkText := fmt.Sprintf("LINK TO PR (#%d)", pr.ID)

		// Construct CSV row manually with proper quoting
		row := fmt.Sprintf(`"%s"%s"%s"%s"%s"%s"%s"%s"%s"`+"\n",
			escapedRepo, delimiter,
			escapedTitle, delimiter,
			pr.FormatCompletionDate(dateFormat), delimiter,
			webURL, delimiter,
			linkText)
		buf.WriteString(row)
	}

	return buf.String(), nil
}
