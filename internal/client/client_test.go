package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CodellaSoftware/azure-pr-cli/internal/config"
	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUserID = "test-user-id-guid"

func TestNewAzureDevOpsClient(t *testing.T) {
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repositories: []string{"testrepo"},
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)

	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.config)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, baseURL, client.baseURL)
}

func TestGetPullRequests_Success(t *testing.T) {
	now := time.Now()
	testPRs := []models.PullRequest{
		{
			ID:           1,
			Title:        "Test PR 1",
			Status:       "completed",
			CreationDate: now.Add(-48 * time.Hour),
			ClosedDate:   now.Add(-24 * time.Hour),
			Repository:   models.Repository{Name: "testrepo"},
		},
		{
			ID:           2,
			Title:        "Test PR 2",
			Status:       "completed",
			CreationDate: now.Add(-72 * time.Hour),
			ClosedDate:   now.Add(-12 * time.Hour),
			Repository:   models.Repository{Name: "testrepo"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.NotEmpty(t, r.Header.Get("Authorization"))
		assert.Contains(t, r.Header.Get("Authorization"), "Basic")

		if strings.Contains(r.URL.Path, "_apis/connectionData") {
			serveConnectionData(t, w, testUserID)
			return
		}

		assert.Contains(t, r.URL.Path, "_apis/git/repositories")
		assert.Contains(t, r.URL.Query().Get("api-version"), apiVersion)
		assert.Equal(t, testUserID, r.URL.Query().Get("searchCriteria.creatorId"))

		serveJSON(t, w, models.PRListResponse{Value: testPRs, Count: len(testPRs)})
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	from := now.Add(-168 * time.Hour)
	to := now
	prs, err := client.GetPullRequests(from, to, "completed")

	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, "Test PR 2", prs[0].Title)
	assert.Equal(t, "Test PR 1", prs[1].Title)
}

func TestGetPullRequests_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API returned status 401")
}

func TestGetPullRequests_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestResolveCurrentUserID_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "_apis/connectionData")
		serveConnectionData(t, w, testUserID)
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	userID, err := client.resolveCurrentUserID()

	require.NoError(t, err)
	assert.Equal(t, testUserID, userID)
}

func TestResolveCurrentUserID_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	_, err := client.resolveCurrentUserID()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API returned status 403")
}

func TestResolveCurrentUserID_MissingID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveJSON(t, w, connectionDataResponse{})
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	_, err := client.resolveCurrentUserID()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine authenticated user ID")
}

func TestBuildURL(t *testing.T) {
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repositories: []string{"testrepo"},
		PAT:          "testtoken",
	}

	client := NewAzureDevOpsClient(cfg)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		status           string
		expectedContains []string
		notContains      []string
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
				"searchCriteria.creatorId=" + testUserID,
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
				"searchCriteria.creatorId=" + testUserID,
			},
			notContains: []string{"searchCriteria.status="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.buildURL("testrepo", from, tt.status, testUserID)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, url, expected)
			}
			for _, excluded := range tt.notContains {
				assert.NotContains(t, url, excluded)
			}
		})
	}
}

func TestFilterPRs(t *testing.T) {
	now := time.Now()

	prs := []models.PullRequest{
		{
			ID:           1,
			Title:        "Completed PR in range",
			Status:       "completed",
			CreationDate: now.Add(-48 * time.Hour),
			ClosedDate:   now.Add(-24 * time.Hour),
		},
		{
			ID:           2,
			Title:        "Completed PR out of range",
			Status:       "completed",
			CreationDate: now.Add(-168 * time.Hour),
			ClosedDate:   now.Add(-144 * time.Hour),
		},
		{
			ID:           3,
			Title:        "Active PR in range",
			Status:       "active",
			CreationDate: now.Add(-12 * time.Hour),
			ClosedDate:   time.Time{},
		},
		{
			ID:           4,
			Title:        "Abandoned PR in range",
			Status:       "abandoned",
			CreationDate: now.Add(-36 * time.Hour),
			ClosedDate:   now.Add(-30 * time.Hour),
		},
	}

	client := &AzureDevOpsClient{}
	from := now.Add(-72 * time.Hour)
	to := now

	tests := []struct {
		name          string
		status        string
		expectedCount int
		expectedIDs   []int
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

func TestGetPullRequests_MultipleRepositories(t *testing.T) {
	now := time.Now()
	testPRsRepo1 := []models.PullRequest{
		{
			ID:           1,
			Title:        "Repo1 PR 1",
			Status:       "completed",
			CreationDate: now.Add(-48 * time.Hour),
			ClosedDate:   now.Add(-24 * time.Hour),
			Repository:   models.Repository{Name: "repo1"},
		},
	}
	testPRsRepo2 := []models.PullRequest{
		{
			ID:           2,
			Title:        "Repo2 PR 1",
			Status:       "completed",
			CreationDate: now.Add(-24 * time.Hour),
			ClosedDate:   now.Add(-12 * time.Hour),
			Repository:   models.Repository{Name: "repo2"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_apis/connectionData") {
			serveConnectionData(t, w, testUserID)
			return
		}

		assert.Contains(t, r.URL.Path, "_apis/git/repositories")

		if strings.Contains(r.URL.Path, "/repositories/repo1/") {
			serveJSON(t, w, models.PRListResponse{Value: testPRsRepo1, Count: len(testPRsRepo1)})
		} else {
			assert.Contains(t, r.URL.Path, "/repositories/repo2/")
			serveJSON(t, w, models.PRListResponse{Value: testPRsRepo2, Count: len(testPRsRepo2)})
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repositories: []string{"repo1", "repo2"},
		PAT:          "testtoken",
	}
	client := NewAzureDevOpsClient(cfg)
	client.SetBaseURL(server.URL)

	from := now.Add(-7 * 24 * time.Hour)
	to := now

	prs, err := client.GetPullRequests(from, to, "completed")

	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, "repo1", prs[0].Repository.Name)
	assert.Equal(t, "Repo1 PR 1", prs[0].Title)
	assert.Equal(t, "repo2", prs[1].Repository.Name)
	assert.Equal(t, "Repo2 PR 1", prs[1].Title)
}

func TestGetPullRequests_MultipleRepositories_Error(t *testing.T) {
	testPRsRepo1 := []models.PullRequest{
		{
			ID:           1,
			Title:        "Repo1 PR 1",
			Status:       "completed",
			CreationDate: time.Now().Add(-48 * time.Hour),
			ClosedDate:   time.Now().Add(-24 * time.Hour),
			Repository:   models.Repository{Name: "repo1"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_apis/connectionData") {
			serveConnectionData(t, w, testUserID)
			return
		}

		if strings.Contains(r.URL.Path, "/repositories/repo1/") {
			serveJSON(t, w, models.PRListResponse{Value: testPRsRepo1, Count: len(testPRsRepo1)})
		} else {
			assert.Contains(t, r.URL.Path, "/repositories/repo2/")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repositories: []string{"repo1", "repo2"},
		PAT:          "testtoken",
	}
	client := NewAzureDevOpsClient(cfg)
	client.SetBaseURL(server.URL)

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch PRs for repository repo2")
	assert.Contains(t, err.Error(), "API returned status 401")
}

func TestSetHTTPClient(t *testing.T) {
	client := givenClientWithServer(nil)

	customClient := &http.Client{Timeout: 60 * time.Second}
	client.SetHTTPClient(customClient)

	assert.NotNil(t, client)
}

func TestSetBaseURL(t *testing.T) {
	client := givenClientWithServer(nil)

	client.SetBaseURL("https://custom.dev.azure.com")

	assert.NotNil(t, client)
}

func TestGetPullRequests_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	client := givenClientWithServer(server)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	_, err := client.GetPullRequests(from, to, "completed")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func givenClientWithServer(server *httptest.Server) *AzureDevOpsClient {
	cfg := &config.Config{
		Organization: "testorg",
		Project:      "testproject",
		Repositories: []string{"testrepo"},
		PAT:          "testtoken",
	}
	c := NewAzureDevOpsClient(cfg)
	if server != nil {
		c.SetBaseURL(server.URL)
	}
	return c
}

func serveConnectionData(t *testing.T, w http.ResponseWriter, userID string) {
	t.Helper()
	var data connectionDataResponse
	data.AuthenticatedUser.ID = userID
	serveJSON(t, w, data)
}

func serveJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}
