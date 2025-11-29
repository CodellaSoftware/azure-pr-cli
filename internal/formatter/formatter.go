package formatter

import (
	"github.com/yourusername/azure-pr-cli/internal/models"
)

type Formatter interface {
	Format(prs []models.PullRequest, dateFormat string, options map[string]string) (string, error)
}
