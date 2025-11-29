# Azure DevOps PR CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A professional CLI tool for fetching and displaying Pull Requests from Azure DevOps repositories with Excel export capabilities.

## Features

- 🔍 Fetch PRs from Azure DevOps repositories
- 📅 Filter by current month or custom date ranges
- 📊 Multiple output formats: Table, JSON, CSV, XLSX
- 📈 **Excel XLSX export with clickable hyperlinks**
- 🔐 Secure authentication via PAT or .env files
- ✅ Comprehensive test coverage
- 🏗️ Built with Cobra CLI framework
- 📁 Automatic format detection from file extensions

## Excel XLSX Features

The XLSX export format provides professional Excel spreadsheets with:

- **Clickable hyperlinks** in the "PR LINK" column that open PRs directly in your browser
- **Auto-fit columns** for optimal readability
- **Professional formatting** with headers and data properly organized
- **Default filename** `pull-requests.xlsx` when no output file is specified
- **Extension-based format detection** - save as `.xlsx` for automatic XLSX output

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
git clone https://github.com/yourusername/azure-pr-cli.git
cd azure-pr-cli
make build
```

### Using Go Install

```bash
go install github.com/yourusername/azure-pr-cli@latest
```

## Configuration

### Azure DevOps Personal Access Token (PAT)

1. Navigate to Azure DevOps → User Settings → Personal Access Tokens
2. Create a new token with **Code (Read)** permissions
3. Copy the generated token

### Environment Variables

You can set the following environment variables:

```bash
export AZURE_DEVOPS_PAT="your-personal-access-token"
export AZURE_DEVOPS_ORG="your-organization"
export AZURE_DEVOPS_PROJECT="your-project"
```

Alternatively, create a `.env` file in the project root with the same variables:

```bash
# .env
AZURE_DEVOPS_PAT=your-personal-access-token
AZURE_DEVOPS_ORG=your-organization
AZURE_DEVOPS_PROJECT=your-project
```

The application will automatically load the `.env` file if it exists.

## Usage

### Basic Commands

```bash
# List PRs and save as XLSX (default behavior)
azure-pr-cli list -o myorg -p myproject -r myrepo

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
# Custom date range
azure-pr-cli list -o myorg -p myproject -r myrepo --from 2024-01-01 --to 2024-01-31

# Output formats
azure-pr-cli list -o myorg -p myproject -r myrepo --format table  # Console table
azure-pr-cli list -o myorg -p myproject -r myrepo --format json   # JSON output
azure-pr-cli list -o myorg -p myproject -r myrepo --format csv    # CSV output

# Save to specific files (format auto-detected from extension)
azure-pr-cli list -o myorg -p myproject -r myrepo --output-file prs.csv
azure-pr-cli list -o myorg -p myproject -r myrepo --output-file report.xlsx

# Custom date format (Go time format, default: 02.01.2006)
azure-pr-cli list -o myorg -p myproject -r myrepo --date-format "2006-01-02"

# CSV with custom delimiter (default: ;)
azure-pr-cli list -o myorg -p myproject -r myrepo --format csv --delimiter ","

# Filter by PR status
azure-pr-cli list -o myorg -p myproject -r myrepo --status all  # active, completed, abandoned, all

# Verbose output
azure-pr-cli list -o myorg -p myproject -r myrepo -v
```

### Output Formats

| Format | Description | Default File | Features |
|--------|-------------|--------------|----------|
| **xlsx** | Excel spreadsheet | `pull-requests.xlsx` | Clickable hyperlinks, auto-fit columns |
| table | Console table | stdout | Human-readable, colored output |
| json | JSON array | stdout | Machine-readable |
| csv | CSV file | stdout | Spreadsheet compatible |

**Note**: XLSX format includes clickable hyperlinks in the "PR LINK" column that open directly in your browser.

### Command Reference

```bash
# Show help
azure-pr-cli --help
azure-pr-cli list --help

# Show version
azure-pr-cli version
```

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile)

### Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/azure-pr-cli.git
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
│   │   ├── table.go      # Table formatter
│   │   ├── json.go       # JSON formatter
│   │   ├── csv.go        # CSV formatter
│   │   ├── xlsx.go       # Excel XLSX formatter with hyperlinks
│   │   └── formatter_test.go
│   └── models/           # Data models
│       └── pullrequest.go
├── pkg/                  # Public libraries
│   └── azuredevops/     # Azure DevOps utilities
├── test/                # Additional test files
│   ├── fixtures/        # Test fixtures
│   └── integration/     # Integration tests
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
```

### Run Integration Tests

```bash
make test-integration
```

### Generate Coverage Report

```bash
make coverage-html
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
