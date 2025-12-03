package formatter

import (
	"testing"
	"time"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColumns_Default(t *testing.T) {
	cols, err := ParseColumns("")

	require.NoError(t, err)
	assert.Len(t, cols, 5)
	assert.Equal(t, "index", cols[0].ID)
	assert.Equal(t, "repo", cols[1].ID)
	assert.Equal(t, "title", cols[2].ID)
	assert.Equal(t, "completed", cols[3].ID)
	assert.Equal(t, "url", cols[4].ID)
}

func TestParseColumns_CustomColumns(t *testing.T) {
	cols, err := ParseColumns("index,author,status,title")

	require.NoError(t, err)
	assert.Len(t, cols, 4)
	assert.Equal(t, "index", cols[0].ID)
	assert.Equal(t, "author", cols[1].ID)
	assert.Equal(t, "status", cols[2].ID)
	assert.Equal(t, "title", cols[3].ID)
}

func TestParseColumns_InvalidColumn(t *testing.T) {
	_, err := ParseColumns("index,invalid_column,title")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown column: invalid_column")
}

func TestParseColumns_EmptyColumns(t *testing.T) {
	_, err := ParseColumns("  ,  ,  ")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid columns specified")
}

func TestParseColumns_WithSpaces(t *testing.T) {
	cols, err := ParseColumns("  index , repo , title  ")

	require.NoError(t, err)
	assert.Len(t, cols, 3)
	assert.Equal(t, "index", cols[0].ID)
	assert.Equal(t, "repo", cols[1].ID)
	assert.Equal(t, "title", cols[2].ID)
}

func TestListColumns(t *testing.T) {
	output := ListColumns()

	assert.Contains(t, output, "Available columns:")
	assert.Contains(t, output, "index")
	assert.Contains(t, output, "repo")
	assert.Contains(t, output, "title")
	assert.Contains(t, output, "author")
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "source")
	assert.Contains(t, output, "target")
	assert.Contains(t, output, "url")
	assert.Contains(t, output, "Default columns:")
}

func TestColumnGetValue(t *testing.T) {
	pr := models.PullRequest{
		ID:          123,
		Title:       "Test PR",
		Status:      "completed",
		Description: "Test description",
		CreatedBy: models.User{
			DisplayName: "John Doe",
			UniqueName:  "john.doe@example.com",
		},
		CreationDate:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		ClosedDate:    time.Date(2024, 1, 20, 14, 0, 0, 0, time.UTC),
		SourceRefName: "refs/heads/feature/test",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		Repository: models.Repository{
			Name: "test-repo",
		},
	}

	tests := []struct {
		columnID string
		expected string
	}{
		{"index", "5"},
		{"id", "123"},
		{"repo", "test-repo"},
		{"title", "Test PR"},
		{"author", "John Doe"},
		{"status", "completed"},
		{"source", "feature/test"},
		{"target", "main"},
		{"merge_status", "succeeded"},
	}

	for _, tt := range tests {
		t.Run(tt.columnID, func(t *testing.T) {
			col := AvailableColumns[tt.columnID]
			value := col.GetValue(pr, 5, "myorg", "myproject")
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestColumnGetValue_URL(t *testing.T) {
	pr := models.PullRequest{
		ID: 456,
		Repository: models.Repository{
			Name: "my-repo",
		},
	}

	col := AvailableColumns["url"]
	value := col.GetValue(pr, 1, "testorg", "testproject")

	assert.Equal(t, "https://testorg.visualstudio.com/testproject/_git/my-repo/pullrequest/456", value)
}

func TestColumnGetValue_Dates(t *testing.T) {
	pr := models.PullRequest{
		CreationDate: time.Date(2024, 3, 15, 9, 30, 45, 0, time.UTC),
		ClosedDate:   time.Date(2024, 3, 20, 16, 45, 30, 0, time.UTC),
	}

	createdCol := AvailableColumns["created"]
	createdValue := createdCol.GetValue(pr, 1, "", "")
	assert.Contains(t, createdValue, "2024-03-15")

	completedCol := AvailableColumns["completed"]
	completedValue := completedCol.GetValue(pr, 1, "", "")
	assert.Contains(t, completedValue, "2024-03-20")
}

func TestColumnGetValue_ZeroDates(t *testing.T) {
	pr := models.PullRequest{}

	createdCol := AvailableColumns["created"]
	createdValue := createdCol.GetValue(pr, 1, "", "")
	assert.Equal(t, "N/A", createdValue)

	completedCol := AvailableColumns["completed"]
	completedValue := completedCol.GetValue(pr, 1, "", "")
	assert.Equal(t, "N/A", completedValue)
}

func TestTableFormatter_CustomColumns(t *testing.T) {
	formatter := NewTableFormatter()
	prs := []models.PullRequest{
		{
			ID:     1,
			Title:  "Test PR",
			Status: "completed",
			CreatedBy: models.User{
				DisplayName: "Jane Doe",
			},
			Repository: models.Repository{Name: "test-repo"},
		},
	}

	options := map[string]string{
		"columns": "index,title,author,status",
		"org":     "testorg",
		"project": "testproject",
	}

	output, err := formatter.Format(prs, "", options)

	require.NoError(t, err)
	assert.Contains(t, output, "#")
	assert.Contains(t, output, "PR NAME")
	assert.Contains(t, output, "AUTHOR")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "Jane Doe")
	assert.Contains(t, output, "completed")
	assert.NotContains(t, output, "REPO NAME")
}

func TestCSVFormatter_CustomColumns(t *testing.T) {
	formatter := NewCSVFormatter()
	prs := []models.PullRequest{
		{
			ID:         1,
			Title:      "Test PR",
			Status:     "active",
			Repository: models.Repository{Name: "test-repo"},
		},
	}

	options := map[string]string{
		"columns":   "index,repo,status",
		"delimiter": ",",
	}

	output, err := formatter.Format(prs, "", options)

	require.NoError(t, err)
	assert.Contains(t, output, "#,REPO NAME,STATUS")
	assert.Contains(t, output, "test-repo")
	assert.Contains(t, output, "active")
}
