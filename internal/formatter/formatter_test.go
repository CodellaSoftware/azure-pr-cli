package formatter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPRs() []models.PullRequest {
	now := time.Now()
	return []models.PullRequest{
		{
			ID:     1,
			Title:  "Add new feature",
			Status: "completed",
			Repository: models.Repository{
				Name: "test-repo",
			},
			ClosedDate: now.Add(-24 * time.Hour),
			URL:        "https://dev.azure.com/org/project/_git/repo/pullrequest/1",
		},
		{
			ID:     2,
			Title:  "Fix bug in login",
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

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	out := string(output)
	assert.Contains(t, out, "#")
	assert.Contains(t, out, "REPO NAME")
	assert.Contains(t, out, "PR NAME")
	assert.Contains(t, out, "COMPLETION DATE")
	assert.Contains(t, out, "PR URL")
	assert.Contains(t, out, "test-repo")
	assert.Contains(t, out, "Add new feature")
	assert.Contains(t, out, "Fix bug in login")
	assert.Contains(t, out, "Total PRs: 2")
}

func TestTableFormatter_EmptyList(t *testing.T) {
	formatter := NewTableFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.Contains(t, string(output), "No pull requests found")
}

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	var parsed []map[string]string
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
	assert.Equal(t, "1", parsed[0]["index"])
	assert.Equal(t, "Add new feature", parsed[0]["title"])
	assert.Equal(t, "2", parsed[1]["index"])
	assert.Equal(t, "Fix bug in login", parsed[1]["title"])
}

func TestJSONFormatter_EmptyList(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	var parsed []map[string]string
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 0)
}

func TestCSVFormatter(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	assert.Len(t, lines, 3)

	assert.Contains(t, lines[0], "#")
	assert.Contains(t, lines[0], "REPO NAME")
	assert.Contains(t, lines[0], "PR NAME")
	assert.Contains(t, lines[0], "COMPLETION DATE")
	assert.Contains(t, lines[0], "PR URL")

	assert.Contains(t, lines[1], "test-repo")
	assert.Contains(t, lines[1], "Add new feature")
	assert.Contains(t, lines[2], "Fix bug in login")
}

func TestCSVFormatter_EmptyList(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "REPO NAME")
}

func TestFormatters_AllImplementInterface(t *testing.T) {
	var _ Formatter = &TableFormatter{}
	var _ Formatter = &JSONFormatter{}
	var _ Formatter = &CSVFormatter{}
	var _ Formatter = &XLSXFormatter{}
	var _ Formatter = &DOCXFormatter{}
}

func TestXLSXFormatter(t *testing.T) {
	formatter := NewXLSXFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{"org": "testorg", "project": "testproject"})

	require.NoError(t, err)
	assert.NotEmpty(t, output)
	assert.True(t, len(output) > 0, "XLSX output should not be empty")

	assert.Equal(t, "PK", string(output[:2]), "XLSX should be a ZIP file starting with PK")
}

func TestXLSXFormatter_EmptyList(t *testing.T) {
	formatter := NewXLSXFormatter()
	prs := []models.PullRequest{}

	output, err := formatter.Format(prs, "", map[string]string{"org": "testorg", "project": "testproject"})

	require.NoError(t, err)
	assert.NotEmpty(t, output)
	assert.True(t, len(output) > 0, "XLSX output should not be empty even for empty list")
}

func TestXLSXFormatter_WithDateFormat(t *testing.T) {
	formatter := NewXLSXFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "02.01.2006", map[string]string{"org": "testorg", "project": "testproject"})

	require.NoError(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, "PK", string(output[:2]), "XLSX should be a ZIP file starting with PK")
}

func TestXLSXFormatter_MissingOptions(t *testing.T) {
	formatter := NewXLSXFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestXLSXFormatter_LongContent(t *testing.T) {
	formatter := NewXLSXFormatter()

	longTitle := strings.Repeat("Very long title with special characters: áéíóú ñ & < > \" quotes ", 10)
	prs := []models.PullRequest{
		{
			ID:          1,
			Title:       longTitle,
			Description: "Long description",
			Status:      "completed",
			Repository: models.Repository{
				Name: "test-repo",
			},
			ClosedDate: time.Now(),
			URL:        "https://dev.azure.com/org/project/_git/repo/pullrequest/1",
		},
	}

	output, err := formatter.Format(prs, "02.01.2006", map[string]string{"org": "testorg", "project": "testproject"})

	require.NoError(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, "PK", string(output[:2]), "XLSX should be a ZIP file starting with PK")
}

func TestJSONFormatter_WithDateFormat(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "2006-01-02", map[string]string{})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	var parsed []map[string]string
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
	assert.Equal(t, "1", parsed[0]["index"])
	assert.Equal(t, "2", parsed[1]["index"])
}

func TestJSONFormatter_RespectsColumns(t *testing.T) {
	formatter := NewJSONFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{"columns": "index,title"})

	require.NoError(t, err)

	var parsed []map[string]string
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)

	assert.Equal(t, "1", parsed[0]["index"])
	assert.Equal(t, "Add new feature", parsed[0]["title"])
	_, hasRepo := parsed[0]["repo"]
	assert.False(t, hasRepo, "repo column should not be present when not requested")
}

func TestCSVFormatter_WithCustomDelimiter(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{"delimiter": ","})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	assert.Len(t, lines, 3)

	assert.Contains(t, lines[0], "#,REPO NAME,PR NAME")
	assert.Contains(t, lines[1], "test-repo")
}

func TestCSVFormatter_EmptyDelimiter(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := createTestPRs()

	output, err := formatter.Format(prs, "", map[string]string{"delimiter": ""})

	require.NoError(t, err)
	assert.NotEmpty(t, output)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	assert.Len(t, lines, 3)

	assert.Contains(t, lines[0], "#;REPO NAME;PR NAME")
}

func TestFormatters_HandleSpecialCharacters(t *testing.T) {
	prs := []models.PullRequest{
		{
			ID:     1,
			Title:  "Fix: Handle \"quotes\" and, commas",
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
			output, err := tt.formatter.Format(prs, "", map[string]string{})
			require.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}

	xlsxFormatter := NewXLSXFormatter()
	xlsxOutput, err := xlsxFormatter.Format(prs, "", map[string]string{"org": "testorg", "project": "testproject"})
	require.NoError(t, err)
	assert.NotEmpty(t, xlsxOutput)
}
