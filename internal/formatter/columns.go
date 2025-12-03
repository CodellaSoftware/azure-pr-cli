package formatter

import (
	"fmt"
	"strings"

	"github.com/CodellaSoftware/azure-pr-cli/internal/models"
)

type Column struct {
	ID          string
	Header      string
	Description string
	GetValue    func(pr models.PullRequest, index int, org, project string) string
}

var DefaultColumns = "index,repo,title,completed,url,link"

var AvailableColumns = map[string]Column{
	"index": {
		ID:          "index",
		Header:      "#",
		Description: "Row number",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return fmt.Sprintf("%d", index)
		},
	},
	"id": {
		ID:          "id",
		Header:      "PR ID",
		Description: "Pull request ID",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return fmt.Sprintf("%d", pr.ID)
		},
	},
	"repo": {
		ID:          "repo",
		Header:      "REPO NAME",
		Description: "Repository name",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.Repository.Name
		},
	},
	"title": {
		ID:          "title",
		Header:      "PR NAME",
		Description: "Pull request title",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.Title
		},
	},
	"author": {
		ID:          "author",
		Header:      "AUTHOR",
		Description: "PR author display name",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.CreatedBy.DisplayName
		},
	},
	"status": {
		ID:          "status",
		Header:      "STATUS",
		Description: "PR status (active, completed, abandoned)",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.Status
		},
	},
	"created": {
		ID:          "created",
		Header:      "CREATED DATE",
		Description: "Date when PR was created",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			if pr.CreationDate.IsZero() {
				return "N/A"
			}
			return pr.CreationDate.Format("2006-01-02 15:04:05")
		},
	},
	"completed": {
		ID:          "completed",
		Header:      "COMPLETION DATE",
		Description: "Date when PR was completed/closed",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			if pr.ClosedDate.IsZero() {
				return "N/A"
			}
			return pr.ClosedDate.Format("2006-01-02 15:04:05")
		},
	},
	"source": {
		ID:          "source",
		Header:      "SOURCE BRANCH",
		Description: "Source branch name",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return strings.TrimPrefix(pr.SourceRefName, "refs/heads/")
		},
	},
	"target": {
		ID:          "target",
		Header:      "TARGET BRANCH",
		Description: "Target branch name",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return strings.TrimPrefix(pr.TargetRefName, "refs/heads/")
		},
	},
	"merge_status": {
		ID:          "merge_status",
		Header:      "MERGE STATUS",
		Description: "Merge status (succeeded, conflicts, etc.)",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.MergeStatus
		},
	},
	"url": {
		ID:          "url",
		Header:      "PR URL",
		Description: "Web URL to the pull request",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return pr.GetWebURL(org, project)
		},
	},
	"link": {
		ID:          "link",
		Header:      "PR LINK",
		Description: "Clickable hyperlink (for XLSX)",
		GetValue: func(pr models.PullRequest, index int, org, project string) string {
			return fmt.Sprintf("LINK TO PR (#%d)", pr.ID)
		},
	},
}

var ColumnOrder = []string{
	"index", "id", "repo", "title", "author", "status",
	"created", "completed", "source", "target", "merge_status", "url", "link",
}

func ParseColumns(columnsStr string) ([]Column, error) {
	if columnsStr == "" {
		columnsStr = DefaultColumns
	}

	columnNames := strings.Split(columnsStr, ",")
	var columns []Column

	for _, name := range columnNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		col, exists := AvailableColumns[name]
		if !exists {
			return nil, fmt.Errorf("unknown column: %s. Use --list-columns to see available columns", name)
		}
		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid columns specified")
	}

	return columns, nil
}

func ListColumns() string {
	var sb strings.Builder
	sb.WriteString("Available columns:\n\n")
	sb.WriteString(fmt.Sprintf("  %-15s %-20s %s\n", "NAME", "HEADER", "DESCRIPTION"))
	sb.WriteString(fmt.Sprintf("  %-15s %-20s %s\n", "----", "------", "-----------"))

	for _, name := range ColumnOrder {
		col := AvailableColumns[name]
		sb.WriteString(fmt.Sprintf("  %-15s %-20s %s\n", col.ID, col.Header, col.Description))
	}

	sb.WriteString(fmt.Sprintf("\nDefault columns: %s\n", DefaultColumns))
	sb.WriteString("\nUsage: --columns \"index,repo,title,author,status,completed,url\"\n")

	return sb.String()
}
