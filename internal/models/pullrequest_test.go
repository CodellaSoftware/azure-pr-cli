package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPullRequest_GetWebURL(t *testing.T) {
	pr := PullRequest{
		ID: 123,
		Repository: Repository{
			Name: "my-repo",
		},
	}

	url := pr.GetWebURL("myorg", "myproject")
	expected := "https://myorg.visualstudio.com/myproject/_git/my-repo/pullrequest/123"
	assert.Equal(t, expected, url)
}

func TestPullRequest_IsCompleted(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"completed", true},
		{"active", false},
		{"abandoned", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			pr := PullRequest{Status: tt.status}
			assert.Equal(t, tt.expected, pr.IsCompleted())
		})
	}
}

func TestPullRequest_IsActive(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"active", true},
		{"completed", false},
		{"abandoned", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			pr := PullRequest{Status: tt.status}
			assert.Equal(t, tt.expected, pr.IsActive())
		})
	}
}

func TestPullRequest_IsAbandoned(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"abandoned", true},
		{"completed", false},
		{"active", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			pr := PullRequest{Status: tt.status}
			assert.Equal(t, tt.expected, pr.IsAbandoned())
		})
	}
}

func TestPullRequest_FormatCompletionDate(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name       string
		closedDate time.Time
		dateFormat string
		expected   string
	}{
		{
			name:       "with custom format",
			closedDate: testTime,
			dateFormat: "02.01.2006",
			expected:   "15.01.2024",
		},
		{
			name:       "with empty format uses default",
			closedDate: testTime,
			dateFormat: "",
			expected:   "2024-01-15 10:30:45",
		},
		{
			name:       "with zero time returns N/A",
			closedDate: time.Time{},
			dateFormat: "02.01.2006",
			expected:   "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PullRequest{ClosedDate: tt.closedDate}
			result := pr.FormatCompletionDate(tt.dateFormat)
			assert.Equal(t, tt.expected, result)
		})
	}
}
