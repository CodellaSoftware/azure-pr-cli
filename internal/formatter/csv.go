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

	header := fmt.Sprintf("#%sREPO NAME%sPR NAME%sPR COMPLETION DATE%sPR URL%sPR LINK\n", delimiter, delimiter, delimiter, delimiter, delimiter)
	buf.WriteString(header)

	for i, pr := range prs {
		webURL := pr.GetWebURL(options["org"], options["project"])
		escapedTitle := strings.ReplaceAll(pr.Title, `"`, `""`)
		escapedRepo := strings.ReplaceAll(pr.Repository.Name, `"`, `""`)

		linkText := fmt.Sprintf("LINK TO PR (#%d)", pr.ID)

		row := fmt.Sprintf(`"%d"%s"%s"%s"%s"%s"%s"%s"%s"%s"%s"`+"\n",
			i+1, delimiter,
			escapedRepo, delimiter,
			escapedTitle, delimiter,
			pr.FormatCompletionDate(dateFormat), delimiter,
			webURL, delimiter,
			linkText)
		buf.WriteString(row)
	}

	return buf.String(), nil
}
