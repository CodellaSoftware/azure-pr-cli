package integration

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/azure-pr-cli/internal/client"
	"github.com/yourusername/azure-pr-cli/internal/config"
)

// TestIntegrationGetPullRequests tests the actual API integration
// Run with: go test -v -run Integration ./test/integration/...
// Requires environment variables: AZURE_DEVOPS_ORG, AZURE_DEVOPS_PROJECT, AZURE_DEVOPS_PAT, TEST_REPOSITORY
func TestIntegrationGetPullRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check for required environment variables
	org := os.Getenv("AZURE_DEVOPS_ORG")
	project := os.Getenv("AZURE_DEVOPS_PROJECT")
	pat := os.Getenv("AZURE_DEVOPS_PAT")
	repo := os.Getenv("TEST_REPOSITORY")

	if org == "" || project == "" || pat == "" || repo == "" {
		t.Skip("Skipping integration test: required environment variables not set")
	}

	cfg := &config.Config{
		Organization: org,
		Project:      project,
		Repository:   repo,
		PAT:          pat,
	}

	azureClient := client.NewAzureDevOpsClient(cfg)

	// Test fetching PRs from last 30 days
	from := time.Now().Add(-30 * 24 * time.Hour)
	to := time.Now()

	prs, err := azureClient.GetPullRequests(from, to, "all")

	require.NoError(t, err, "Failed to fetch pull requests")
	assert.NotNil(t, prs, "PRs should not be nil")

	t.Logf("Found %d pull requests", len(prs))

	// Validate PR structure if any exist
	if len(prs) > 0 {
		pr := prs[0]
		assert.NotEmpty(t, pr.Title, "PR should have a title")
		assert.NotEmpty(t, pr.Status, "PR should have a status")
		assert.NotEmpty(t, pr.Repository.Name, "PR should have a repository name")
		t.Logf("Sample PR: ID=%d, Title=%s, Status=%s", pr.ID, pr.Title, pr.Status)
	}
}

// TestIntegrationAuthentication tests authentication failure scenarios
func TestIntegrationAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	org := os.Getenv("AZURE_DEVOPS_ORG")
	project := os.Getenv("AZURE_DEVOPS_PROJECT")
	repo := os.Getenv("TEST_REPOSITORY")

	if org == "" || project == "" || repo == "" {
		t.Skip("Skipping integration test: required environment variables not set")
	}

	// Use invalid PAT
	cfg := &config.Config{
		Organization: org,
		Project:      project,
		Repository:   repo,
		PAT:          "invalid-pat-token",
	}

	azureClient := client.NewAzureDevOpsClient(cfg)

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	_, err := azureClient.GetPullRequests(from, to, "completed")

	require.Error(t, err, "Should fail with invalid PAT")
	assert.Contains(t, err.Error(), "401", "Should return 401 error")
}
