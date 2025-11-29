package models

import (
	"fmt"
	"time"
)

// PullRequest represents an Azure DevOps pull request
type PullRequest struct {
	ID            int        `json:"pullRequestId"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	CreatedBy     User       `json:"createdBy"`
	CreationDate  time.Time  `json:"creationDate"`
	ClosedDate    time.Time  `json:"closedDate"`
	Repository    Repository `json:"repository"`
	SourceRefName string     `json:"sourceRefName"`
	TargetRefName string     `json:"targetRefName"`
	MergeStatus   string     `json:"mergeStatus"`
	URL           string     `json:"url"`
}

// User represents an Azure DevOps user
type User struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	ID          string `json:"id"`
}

// Repository represents an Azure DevOps repository
type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// PRListResponse represents the API response for listing PRs
type PRListResponse struct {
	Value []PullRequest `json:"value"`
	Count int           `json:"count"`
}

// GetWebURL returns the web URL for the pull request
func (pr *PullRequest) GetWebURL(organization, project string) string {
	// Construct web URL instead of returning API URL
	// Format: https://{org}.visualstudio.com/{project}/_git/{repo}/pullrequest/{id}
	return fmt.Sprintf("https://%s.visualstudio.com/%s/_git/%s/pullrequest/%d",
		organization, project, pr.Repository.Name, pr.ID)
}

// IsCompleted returns true if the PR is completed
func (pr *PullRequest) IsCompleted() bool {
	return pr.Status == "completed"
}

// IsActive returns true if the PR is active
func (pr *PullRequest) IsActive() bool {
	return pr.Status == "active"
}

// IsAbandoned returns true if the PR is abandoned
func (pr *PullRequest) IsAbandoned() bool {
	return pr.Status == "abandoned"
}

// FormatCompletionDate returns a formatted completion date string
func (pr *PullRequest) FormatCompletionDate(dateFormat string) string {
	if pr.ClosedDate.IsZero() {
		return "N/A"
	}
	if dateFormat == "" {
		dateFormat = "2006-01-02 15:04:05"
	}
	return pr.ClosedDate.Format(dateFormat)
}
