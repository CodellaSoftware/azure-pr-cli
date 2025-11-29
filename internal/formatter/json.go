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
	type indexedPR struct {
		Index int `json:"index"`
		models.PullRequest
	}

	indexedPRs := make([]indexedPR, len(prs))
	for i, pr := range prs {
		indexedPRs[i] = indexedPR{
			Index:       i + 1,
			PullRequest: pr,
		}
	}

	output, err := json.MarshalIndent(indexedPRs, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output), nil
}
