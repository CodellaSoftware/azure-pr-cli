package models

import (
	"fmt"
	"time"
)

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

type User struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	ID          string `json:"id"`
}

type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PRListResponse struct {
	Value []PullRequest `json:"value"`
	Count int           `json:"count"`
}

func (pr *PullRequest) GetWebURL(organization, project string) string {
	return fmt.Sprintf("https://%s.visualstudio.com/%s/_git/%s/pullrequest/%d",
		organization, project, pr.Repository.Name, pr.ID)
}

func (pr *PullRequest) IsCompleted() bool {
	return pr.Status == "completed"
}

func (pr *PullRequest) IsActive() bool {
	return pr.Status == "active"
}

func (pr *PullRequest) IsAbandoned() bool {
	return pr.Status == "abandoned"
}

func (pr *PullRequest) FormatCompletionDate(dateFormat string) string {
	if pr.ClosedDate.IsZero() {
		return "N/A"
	}
	if dateFormat == "" {
		dateFormat = "2006-01-02 15:04:05"
	}
	return pr.ClosedDate.Format(dateFormat)
}
