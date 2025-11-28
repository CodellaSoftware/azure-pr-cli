package formatter

import (
	"encoding/json"

	"github.com/yourusername/azure-pr-cli/internal/models"
)

// JSONFormatter formats PRs as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format formats pull requests as JSON
func (f *JSONFormatter) Format(prs []models.PullRequest) (string, error) {
	output, err := json.MarshalIndent(prs, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}
