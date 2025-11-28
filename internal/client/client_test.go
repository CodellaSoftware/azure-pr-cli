package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/azure-pr-cli/internal/config"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

func TestNewAzureDevOpsClient(t *testing.T) {
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repository:   "testrepo",
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)

	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.config)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, baseURL, client.baseURL)
}

func TestGetPullRequests_Success(t *testing.T) {
	// Create test data
	now := time.Now()
	testPRs := []models.PullRequest{
		{
			ID:          1,
			Title:       "Test PR 1",
			Status:      "completed",
			CreationDate: now.Add(-48 * time.Hour),
			ClosedDate:  now.Add(-24 * time.Hour),
			Repository: models.Repository{
				Name: "testrepo",
			},
		},
		{
			ID:          2,
			Title:       "Test PR 2",
			Status:      "completed",
			CreationDate: now.Add(-72 * time.Hour),
			ClosedDate:  now.Add(-12 * time.Hour),
			Repository: models.Repository{
				Name: "testrepo",
			},
		},
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "_apis/git/repositories")
		assert.Contains(t, r.URL.Query().Get("api-version"), apiVersion)

		// Check authorization header
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.Contains(t, authHeader, "Basic")

		response := models.PRListResponse{
			Value: testPRs,
			Count: len(testPRs),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with test configuration
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repository:   "testrepo",
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)
	client.SetBaseURL(server.URL)

	// Execute test
	from := now.Add(-168 * time.Hour) // 7 days ago
	to := now
	prs, err := client.GetPullRequests(from, to, "completed")

	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, "Test PR 1", prs[0].Title)
	assert.Equal(t, "Test PR 2", prs[1].Title)
}

func TestGetPullRequests_APIError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repository:   "testrepo",
		PAT:          "invalidtoken",
	}

	client := NewAzureDevOpsClient(cfg)
	client.SetBaseURL(server.URL)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API returned status 401")
}

func TestGetPullRequests_InvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repository:   "testrepo",
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)
	client.SetBaseURL(server.URL)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestBuildURL(t *testing.T) {
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repository:   "testrepo",
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		status         string
		expectedContains []string
	}{
		{
			name:   "completed status",
			status: "completed",
			expectedContains: []string{
				"testorg",
				"testproject",
				"testrepo",
				"searchCriteria.status=completed",
				"searchCriteria.minTime=2024-01-01T00:00:00Z",
			},
		},
		{
			name:   "all status",
			status: "all",
			expectedContains: []string{
				"testorg",
				"testproject",
				"testrepo",
				"searchCriteria.minTime=2024-01-01T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.buildURL(from, tt.status)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, url, expected)
			}

			if tt.status == "all" {
				assert.NotContains(t, url, "searchCriteria.status=")
			}
		})
	}
}

func TestFilterPRs(t *testing.T) {
	now := time.Now()

	prs := []models.PullRequest{
		{
			ID:          1,
			Title:       "Completed PR in range",
			Status:      "completed",
			CreationDate: now.Add(-48 * time.Hour),
			ClosedDate:  now.Add(-24 * time.Hour),
		},
		{
			ID:          2,
			Title:       "Completed PR out of range",
			Status:      "completed",
			CreationDate: now.Add(-168 * time.Hour),
			ClosedDate:  now.Add(-144 * time.Hour),
		},
		{
			ID:          3,
			Title:       "Active PR in range",
			Status:      "active",
			CreationDate: now.Add(-12 * time.Hour),
			ClosedDate:  time.Time{},
		},
		{
			ID:          4,
			Title:       "Abandoned PR in range",
			Status:      "abandoned",
			CreationDate: now.Add(-36 * time.Hour),
			ClosedDate:  now.Add(-30 * time.Hour),
		},
	}

	client := &AzureDevOpsClient{}
	from := now.Add(-72 * time.Hour)
	to := now

	tests := []struct {
		name           string
		status         string
		expectedCount  int
		expectedIDs    []int
	}{
		{
			name:          "filter completed",
			status:        "completed",
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
		{
			name:          "filter all",
			status:        "all",
			expectedCount: 3,
			expectedIDs:   []int{1, 3, 4},
		},
		{
			name:          "filter active",
			status:        "active",
			expectedCount: 1,
			expectedIDs:   []int{3},
		},
		{
			name:          "filter abandoned",
			status:        "abandoned",
			expectedCount: 1,
			expectedIDs:   []int{4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := client.filterPRs(prs, from, to, tt.status)

			assert.Len(t, filtered, tt.expectedCount)

			for i, expectedID := range tt.expectedIDs {
				assert.Equal(t, expectedID, filtered[i].ID)
			}
		})
	}
}
