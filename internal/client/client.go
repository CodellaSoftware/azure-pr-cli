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

type AzureDevOpsClient struct {
	config     *config.Config
	httpClient *http.Client
	baseURL    string
}

func NewAzureDevOpsClient(cfg *config.Config) *AzureDevOpsClient {
	return &AzureDevOpsClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

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

	filteredPRs := c.filterPRs(prResponse.Value, from, to, status)

	return filteredPRs, nil
}

func (c *AzureDevOpsClient) buildURL(from time.Time, status string) string {
	url := fmt.Sprintf(
		"%s/%s/%s/_apis/git/repositories/%s/pullrequests?api-version=%s",
		c.baseURL,
		c.config.Organization,
		c.config.Project,
		c.config.Repository,
		apiVersion,
	)

	if status != "all" {
		url += fmt.Sprintf("&searchCriteria.status=%s", status)
	}

	url += fmt.Sprintf("&searchCriteria.minTime=%s", from.Format(time.RFC3339))

	return url
}

func (c *AzureDevOpsClient) setAuthHeaders(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(":" + c.config.PAT))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")
}

func (c *AzureDevOpsClient) filterPRs(prs []models.PullRequest, from, to time.Time, status string) []models.PullRequest {
	var filtered []models.PullRequest

	for _, pr := range prs {
		var prDate time.Time
		if pr.Status == "completed" || pr.Status == "abandoned" {
			prDate = pr.ClosedDate
		} else {
			prDate = pr.CreationDate
		}

		if prDate.Before(from) || prDate.After(to) {
			continue
		}

		if status != "all" && pr.Status != status {
			continue
		}

		filtered = append(filtered, pr)
	}

	return filtered
}

func (c *AzureDevOpsClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

func (c *AzureDevOpsClient) SetBaseURL(url string) {
	c.baseURL = url
}
