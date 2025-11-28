package formatter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

func createTestPRs() []models.PullRequest {
	now := time.Now()
	return []models.PullRequest{
		{
			ID:    1,
			Title: "Add new feature",
			Status: "completed",
			Repository: models.Repository{
				Name: "test-repo",
			},
			ClosedDate: now.Add(-24 * time.Hour),
			URL:        "https://dev.azure.com/org/project/_git/repo/pullrequest/1",
		},
		{
			ID:    2,
			Title: "Fix bug in login",
			Status: "completed",
			Repository: models.Repository{
				Name: "test-repo",
			},
			ClosedDate: now.Add(-12 * time.Hour),
			URL:        "https://dev.azure.com/org/project/_git/repo/pullrequest/2",
		},
	}
}

func TestTableFormatter(t *testing.T) {
	formatter := NewTableFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.NotEmpty(t, output)
	
	// Check that output contains expected elements
	assert.Contains(t, output, "REPO NAME")
	assert.Contains(t, output, "PR NAME")
	assert.Contains(t, output, "PR COMPLETION DATE")
	assert.Contains(t, output, "PR URL")
	assert.Contains(t, output, "test-repo")
	assert.Contains(t, output, "Add new feature")
	assert.Contains(t, output, "Fix bug in login")
	assert.Contains(t, output, "Total PRs: 2")
}

func TestTableFormatter_EmptyList(t *testing.T) {
	formatter := NewTableFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.Contains(t, output, "No pull requests found")
}

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	// Verify it's valid JSON
	var parsed []models.PullRequest
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
	assert.Equal(t, "Add new feature", parsed[0].Title)
	assert.Equal(t, "Fix bug in login", parsed[1].Title)
}

func TestJSONFormatter_EmptyList(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	// Should be valid JSON representing an empty array
	var parsed []models.PullRequest
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 0)
}

func TestCSVFormatter(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	// Check CSV structure
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3) // Header + 2 data rows

	// Check header
	assert.Contains(t, lines[0], "REPO NAME")
	assert.Contains(t, lines[0], "PR NAME")
	assert.Contains(t, lines[0], "PR COMPLETION DATE")
	assert.Contains(t, lines[0], "PR URL")

	// Check data rows
	assert.Contains(t, lines[1], "test-repo")
	assert.Contains(t, lines[1], "Add new feature")
	assert.Contains(t, lines[2], "Fix bug in login")
}

func TestCSVFormatter_EmptyList(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs)

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	// Should only have header
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "REPO NAME")
}

func TestFormatters_AllImplementInterface(t *testing.T) {
	var _ Formatter = &TableFormatter{}
	var _ Formatter = &JSONFormatter{}
	var _ Formatter = &CSVFormatter{}
}

func TestFormatters_HandleSpecialCharacters(t *testing.T) {
	prs := []models.PullRequest{
		{
			ID:    1,
			Title: "Fix: Handle \"quotes\" and, commas",
			Status: "completed",
			Repository: models.Repository{
				Name: "test-repo",
			},
			ClosedDate: time.Now(),
			URL:        "https://dev.azure.com/org/project/_git/repo/pullrequest/1",
		},
	}

	tests := []struct {
		name      string
		formatter Formatter
	}{
		{"table", NewTableFormatter()},
		{"json", NewJSONFormatter()},
		{"csv", NewCSVFormatter()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tt.formatter.Format(prs)
			require.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}
}
