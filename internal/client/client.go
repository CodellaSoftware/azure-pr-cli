package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/azure-pr-cli/internal/config"
	"github.com/yourusername/azure-pr-cli/internal/models"
)

const (
	apiVersion = "7.1"
	baseURL    = "https://dev.azure.com"
)

// AzureDevOpsClient is a client for interacting with Azure DevOps API
type AzureDevOpsClient struct {
	config     *config.Config
	httpClient *http.Client
	baseURL    string
}

// NewAzureDevOpsClient creates a new Azure DevOps client
func NewAzureDevOpsClient(cfg *config.Config) *AzureDevOpsClient {
	return &AzureDevOpsClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// GetPullRequests fetches pull requests from Azure DevOps
func (c *AzureDevOpsClient) GetPullRequests(from, to time.Time, status string) ([]models.PullRequest, error) {
	url := c.buildURL(from, status)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var prResponse models.PRListResponse
	if err := json.Unmarshal(body, &prResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Filter PRs by date range and status
	filteredPRs := c.filterPRs(prResponse.Value, from, to, status)

	return filteredPRs, nil
}

// buildURL constructs the API URL
func (c *AzureDevOpsClient) buildURL(from time.Time, status string) string {
	url := fmt.Sprintf(
		"%s/%s/%s/_apis/git/repositories/%s/pullrequests?api-version=%s",
		c.baseURL,
		c.config.Organization,
		c.config.Project,
		c.config.Repository,
		apiVersion,
	)

	// Add status filter if not "all"
	if status != "all" {
		url += fmt.Sprintf("&searchCriteria.status=%s", status)
	}

	// Add date filter
	url += fmt.Sprintf("&searchCriteria.minTime=%s", from.Format(time.RFC3339))

	return url
}

// setAuthHeaders sets authentication headers
func (c *AzureDevOpsClient) setAuthHeaders(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(":" + c.config.PAT))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")
}

// filterPRs filters pull requests by date range and status
func (c *AzureDevOpsClient) filterPRs(prs []models.PullRequest, from, to time.Time, status string) []models.PullRequest {
	var filtered []models.PullRequest

	for _, pr := range prs {
		// Check date range
		var prDate time.Time
		if pr.Status == "completed" || pr.Status == "abandoned" {
			prDate = pr.ClosedDate
		} else {
			prDate = pr.CreationDate
		}

		if prDate.Before(from) || prDate.After(to) {
			continue
		}

		// Check status filter
		if status != "all" && pr.Status != status {
			continue
		}

		filtered = append(filtered, pr)
	}

	return filtered
}

// SetHTTPClient sets a custom HTTP client (useful for testing)
func (c *AzureDevOpsClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetBaseURL sets a custom base URL (useful for testing)
func (c *AzureDevOpsClient) SetBaseURL(url string) {
	c.baseURL = url
}
