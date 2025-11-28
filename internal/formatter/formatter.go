package formatter

import (
	"github.com/yourusername/azure-pr-cli/internal/models"
)

// Formatter defines the interface for formatting PR output
type Formatter interface {
	Format(prs []models.PullRequest) (string, error)
}
