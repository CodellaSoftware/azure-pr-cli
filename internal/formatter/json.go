package formatter

import (
	"encoding/json"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(prs []models.PullRequest, dateFormat string, options map[string]string) (string, error) {
	output, err := json.MarshalIndent(prs, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}
