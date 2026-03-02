# Azure DevOps PR CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![CI](https://github.com/CodellaSoftware/azure-pr-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/CodellaSoftware/azure-pr-cli/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A professional CLI tool for fetching and displaying Pull Requests from Azure DevOps repositories with Excel and Word export capabilities.

## Features

- 🔍 Fetch PRs from Azure DevOps repositories
- 📅 Filter by current month or custom date ranges
- 📊 Multiple output formats: Table, JSON, CSV, XLSX, DOCX
- 📈 **Excel XLSX export with clickable hyperlinks**
- 📝 **Word DOCX export via custom template** with clickable hyperlinks
- 🔐 Secure authentication via PAT or .env files
- ✅ Comprehensive test coverage
- 🏗️ Built with Cobra CLI framework
- 📁 Automatic format detection from file extensions
- 🔄 **Multiple repository support** - fetch PRs from multiple repos in one command

## Excel XLSX Features

The XLSX export format provides professional Excel spreadsheets with:

- **Clickable hyperlinks** in the "PR LINK" column that open PRs directly in your browser
- **Auto-fit columns** for optimal readability
- **Professional formatting** with headers and data properly organized
- **Default filename** `pull-requests.xlsx` when no output file is specified
- **Extension-based format detection** - save as `.xlsx` for automatic XLSX output

## Word DOCX Template Features

The DOCX export fills a Word document template you provide:

- **Clickable hyperlinks** for `{{url}}` and `{{link}}` placeholders — formatted to match your template's font/style
- **Document-level placeholders** replaced once at the document level (e.g. report header/footer)
- **Table row template** — one row in your table acts as the template; it is duplicated once per PR
- **Default filename** `pull-requests.docx` when no output file is specified
- **Extension-based format detection** — save as `.docx` to trigger automatic DOCX output

### Creating a DOCX Template

1. Create a `.docx` file in Word with your desired layout
2. Add a table with one data row containing `{{placeholder}}` markers in each cell
3. The first table row containing a PR-level placeholder (e.g. `{{title}}`) is used as the template row — it is replaced by one expanded row per PR
4. Optionally add document-level placeholders anywhere in the document outside the table

**PR-level placeholders** (placed inside the template table row):

| Placeholder | Value |
|---|---|
| `{{index}}` | Sequential row number (1-based) |
| `{{id}}` | Pull request ID |
| `{{repo}}` | Repository name |
| `{{title}}` | Pull request title |
| `{{author}}` | Author display name |
| `{{status}}` | PR status |
| `{{created}}` | Creation date |
| `{{completed}}` | Completion/close date |
| `{{source}}` | Source branch (refs/heads/ stripped) |
| `{{target}}` | Target branch (refs/heads/ stripped) |
| `{{merge_status}}` | Merge status |
| `{{url}}` | Clickable hyperlink to the PR |
| `{{link}}` | Clickable hyperlink with display text `LINK TO PR (#N)` |

**Document-level placeholders** (placed anywhere outside the template row, replaced once):

| Placeholder | Value |
|---|---|
| `{{dateFrom}}` | Report start date |
| `{{dateTo}}` | Report end date |
| `{{reportCreationDate}}` | Last day of the reporting month |

## Multiple Repository Support

Fetch pull requests from multiple repositories in a single command using comma-separated repository names:

```bash
# Fetch PRs from multiple repositories
azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3

# Works with all output formats
azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3 --format json
azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3 --output-file multi-repo-report.xlsx

# Combine with date ranges and filters
azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --from 2024-01-01 --to 2024-01-31 --status all
```

**Features:**
- **Sorted output** - Results are sorted by repository name, then by completion date (newest first)
- **Index column** - Each PR includes a sequential index number across all repositories
- **Unified reporting** - All PRs from specified repositories are combined into a single report
- **Error handling** - Continues processing other repositories if one fails

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Development](#development)
- [Testing](#testing)
- [Project Structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

## Installation

### From Source

```bash
git clone https://github.com/CodellaSoftware/azure-pr-cli.git
cd azure-pr-cli
make build
```

### Using Go Install

```bash
go install github.com/CodellaSoftware/azure-pr-cli@latest
```

## Configuration

### Azure DevOps Personal Access Token (PAT)

1. Navigate to Azure DevOps → User Settings → Personal Access Tokens
2. Create a new token with **Code (Read)** permissions
3. Copy the generated token

### Environment Variables

All configuration options can be set via environment variables, allowing you to configure the tool without command-line flags.

| Environment Variable | Flag | Description | Default |
|---------------------|------|-------------|---------|
| `AZURE_DEVOPS_PAT` | `--pat` | Personal Access Token | (required) |
| `AZURE_DEVOPS_ORG` | `-o, --organization` | Azure DevOps organization | (required) |
| `AZURE_DEVOPS_PROJECT` | `-p, --project` | Azure DevOps project | (required) |
| `AZURE_DEVOPS_REPOSITORIES` | `-r, --repository` | Repository name(s), comma-separated | (required) |
| `AZURE_DEVOPS_STATUS` | `--status` | PR status filter | `completed` |
| `AZURE_DEVOPS_FORMAT` | `-f, --format` | Output format | `xlsx` |
| `AZURE_DEVOPS_OUTPUT_FILE` | `--output-file` | Output file path | `pull-requests.xlsx` |
| `AZURE_DEVOPS_DATE_FORMAT` | `--date-format` | Date format (Go format) | `02.01.2006` |
| `AZURE_DEVOPS_DELIMITER` | `--delimiter` | CSV delimiter | `;` |
| `AZURE_DEVOPS_COLUMNS` | `--columns` | Columns to display | `index,repo,title,completed,url` |
| `AZURE_DEVOPS_TEMPLATE` | `--template` | Path to .docx template file | (required for docx) |

**Note**: Command-line flags take precedence over environment variables.

Create a `.env` file in the project root for persistent configuration:

```bash
# .env - Required settings
AZURE_DEVOPS_PAT=your-personal-access-token
AZURE_DEVOPS_ORG=your-organization
AZURE_DEVOPS_PROJECT=your-project
AZURE_DEVOPS_REPOSITORIES=repo1,repo2,repo3

# Optional - Output settings
AZURE_DEVOPS_FORMAT=table
AZURE_DEVOPS_COLUMNS=index,repo,title,author,status,completed,url
AZURE_DEVOPS_DATE_FORMAT=2006-01-02

# Optional - Filters
AZURE_DEVOPS_STATUS=completed
```

The application will automatically load the `.env` file if it exists.

## Usage

**Default Behavior**: By default, the tool fetches **completed PRs from the current month** and saves them as an Excel file (`pull-requests.xlsx`).

### Basic Commands

```bash
# List PRs from a single repository and save as XLSX (default behavior)
azure-pr-cli list -o myorg -p myproject -r myrepo

# List PRs from MULTIPLE repositories (comma-separated)
azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3

# Using environment variables
export AZURE_DEVOPS_ORG="myorg"
export AZURE_DEVOPS_PROJECT="myproject"
azure-pr-cli list -r myrepo

# With PAT from environment
export AZURE_DEVOPS_PAT="your-token"
azure-pr-cli list -o myorg -p myproject -r myrepo
```

### Advanced Usage

```bash
# Custom date range for single repository
azure-pr-cli list -o myorg -p myproject -r myrepo --from 2024-01-01 --to 2024-01-31

# Custom date range for MULTIPLE repositories
azure-pr-cli list -o myorg -p myproject -r repo1,repo2,repo3 --from 2024-01-01 --to 2024-01-31

# Output formats
azure-pr-cli list -o myorg -p myproject -r myrepo --format table  # Console table
azure-pr-cli list -o myorg -p myproject -r myrepo --format json   # JSON output
azure-pr-cli list -o myorg -p myproject -r myrepo --format csv    # CSV output

# Save to specific files (format auto-detected from extension)
azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --output-file prs.csv
azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --output-file report.xlsx

# DOCX Word report from template
azure-pr-cli list -o myorg -p myproject -r myrepo --format docx --template template.docx
# DOCX with custom output file (format auto-detected from .docx extension)
azure-pr-cli list -o myorg -p myproject -r myrepo --template template.docx --output-file report.docx
# DOCX via environment variable
export AZURE_DEVOPS_TEMPLATE="template.docx"
azure-pr-cli list -o myorg -p myproject -r myrepo --format docx

# Custom date format (Go time format, default: 02.01.2006)
azure-pr-cli list -o myorg -p myproject -r myrepo --date-format "2006-01-02"

# CSV with custom delimiter (default: ;)
azure-pr-cli list -o myorg -p myproject -r myrepo --format csv --delimiter ","

# Filter by PR status across multiple repositories
azure-pr-cli list -o myorg -p myproject -r repo1,repo2 --status all  # active, completed, abandoned, all

# Verbose output
azure-pr-cli list -o myorg -p myproject -r myrepo -v
```

### Output Formats

| Format | Description | Default File | Features |
|--------|-------------|--------------|----------|
| **xlsx** | Excel spreadsheet | `pull-requests.xlsx` | Clickable hyperlinks, auto-fit columns |
| **docx** | Word document (template-based) | `pull-requests.docx` | Custom layout, clickable hyperlinks, document-level placeholders |
| table | Console table | stdout | Human-readable, colored output |
| json | JSON array | stdout | Machine-readable |
| csv | CSV file | stdout | Spreadsheet compatible |

**Note**: Both XLSX and DOCX formats include clickable hyperlinks that open PRs directly in your browser.

### Command Reference

```bash
# Show help
azure-pr-cli --help
azure-pr-cli list --help

# Show version
azure-pr-cli version
```

### Command Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--organization` | `-o` | Azure DevOps organization | Required |
| `--project` | `-p` | Azure DevOps project | Required |
| `--repository` | `-r` | **Repository name(s) - comma-separated for multiple repos** | Required |
| `--from` | | Start date (YYYY-MM-DD) | Start of current month |
| `--to` | | End date (YYYY-MM-DD) | End of current month |
| `--status` | | PR status: active, completed, abandoned, all | completed |
| `--format` | `-f` | Output format: table, json, csv, xlsx, docx | xlsx |
| `--output-file` | | Save to file (format from extension) | pull-requests.xlsx |
| `--template` | | Path to .docx template (required for docx format) | From env/AZURE_DEVOPS_TEMPLATE |
| `--date-format` | | Date format (Go time format) | 02.01.2006 |
| `--delimiter` | | CSV delimiter | ; |
| `--columns` | | Columns to display (see `--list-columns`) | index,repo,title,completed,url |
| `--list-columns` | | List available columns and exit | |
| `--pat` | | Personal Access Token | From env/AZURE_DEVOPS_PAT |
| `--verbose` | `-v` | Enable verbose output for debugging | false |

### Configurable Columns

You can customize which columns are displayed in the output using the `--columns` flag or `AZURE_DEVOPS_COLUMNS` environment variable.

**Available columns:**

| Column | Header | Description |
|--------|--------|-------------|
| `index` | # | Row number |
| `id` | PR ID | Pull request ID |
| `repo` | REPO NAME | Repository name |
| `title` | PR NAME | Pull request title |
| `author` | AUTHOR | PR author display name |
| `status` | STATUS | PR status (active, completed, abandoned) |
| `created` | CREATED DATE | Date when PR was created |
| `completed` | COMPLETION DATE | Date when PR was completed/closed |
| `source` | SOURCE BRANCH | Source branch name |
| `target` | TARGET BRANCH | Target branch name |
| `merge_status` | MERGE STATUS | Merge status (succeeded, conflicts, etc.) |
| `url` | PR URL | Web URL to the pull request |
| `link` | PR LINK | Clickable hyperlink (for XLSX/DOCX) |

**Examples:**

```bash
# Show author and status columns
azure-pr-cli list -r myrepo --columns "index,repo,title,author,status,completed,url"

# Minimal output - just index and title
azure-pr-cli list -r myrepo --columns "index,title"

# Include branch information
azure-pr-cli list -r myrepo --columns "index,repo,title,source,target,completed"

# List all available columns
azure-pr-cli list --list-columns
```

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile)

### Setup

```bash
# Clone the repository
git clone https://github.com/CodellaSoftware/azure-pr-cli.git
cd azure-pr-cli

# Install dependencies
go mod download

# Build
make build

# Run
./azure-pr-cli list -o myorg -p myproject -r myrepo
```

### Project Structure

```
azure-pr-cli/
├── cmd/                    # Command definitions
│   ├── root.go            # Root command
│   ├── list.go            # List command with XLSX support
│   └── version.go         # Version command
├── internal/              # Private application code
│   ├── client/           # Azure DevOps API client
│   │   ├── client.go
│   │   └── client_test.go
│   ├── config/           # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── formatter/        # Output formatters
│   │   ├── columns.go    # Column definitions and parsing
│   │   ├── table.go      # Table formatter
│   │   ├── json.go       # JSON formatter
│   │   ├── csv.go        # CSV formatter
│   │   ├── xlsx.go       # Excel XLSX formatter with hyperlinks
│   │   ├── docx.go       # Word DOCX formatter with template support
│   │   └── formatter_test.go
│   └── models/           # Data models
│       └── pullrequest.go
├── pkg/                  # Public libraries
│   └── azuredevops/     # Azure DevOps utilities
├── .github/             # GitHub configuration
│   └── workflows/       # CI/CD workflows
├── main.go              # Application entry point
├── go.mod               # Go module definition
├── go.sum               # Go dependencies checksum
├── Makefile             # Build automation
├── README.md            # This file
├── LICENSE              # License file
└── .gitignore          # Git ignore rules
```

## Testing

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
make test-coverage
```

### Run Specific Tests

```bash
go test ./internal/client -v
go test ./internal/formatter -v
go test ./internal/config -v
```

### Generate Coverage Report

```bash
make test-coverage
```

## Building

### Build for Current Platform

```bash
make build
```

### Build for All Platforms

```bash
make build-all
```

This creates binaries for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Install Locally

```bash
make install
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Maintain test coverage above 80%
- Follow Go best practices and idioms
- Use `gofmt` for code formatting
- Update documentation for new features

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra)
- Uses [tablewriter](https://github.com/olekukonko/tablewriter) for beautiful tables
- Uses [excelize](https://github.com/xuri/excelize) for Excel XLSX generation with hyperlinks
- Inspired by Azure DevOps CLI

## Support

For issues, questions, or contributions, please:
- Open an issue on GitHub
- Check existing issues and discussions
- Review the documentation

## Troubleshooting

### Common Issues

**Authentication Errors**
- Ensure your PAT has "Code (Read)" permissions
- Check that `AZURE_DEVOPS_PAT` environment variable is set correctly
- Verify the `.env` file exists and contains the correct PAT

**Repository Not Found**
- Verify the repository name is spelled correctly
- Ensure you have access to the repository in Azure DevOps
- Check that the organization and project names are correct

**No Pull Requests Found**
- Try expanding the date range with `--from` and `--to` flags
- Use `--status all` to include active and abandoned PRs
- Check if the repository actually has any PRs in the specified time period

**Network/Connection Issues**
- Verify internet connectivity
- Check Azure DevOps service status
- Ensure your firewall allows HTTPS connections to `*.visualstudio.com`

**Verbose Mode**
Use the `--verbose` flag for detailed debugging information:
```bash
azure-pr-cli list -o myorg -p myproject -r myrepo -v
```
