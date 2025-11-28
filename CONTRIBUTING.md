# Contributing to Azure DevOps PR CLI

Thank you for your interest in contributing to Azure DevOps PR CLI! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples**
- **Describe the behavior you observed and what you expected**
- **Include screenshots if relevant**
- **Note your environment** (OS, Go version, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description of the suggested enhancement**
- **Explain why this enhancement would be useful**
- **List any alternative solutions you've considered**

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the coding style** used throughout the project
3. **Write tests** for new functionality
4. **Ensure the test suite passes** (`make test`)
5. **Update documentation** as needed
6. **Write clear commit messages**
7. **Create a pull request** with a clear description

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make
- Git

### Setup Instructions

```bash
# Clone your fork
git clone https://github.com/yourusername/azure-pr-cli.git
cd azure-pr-cli

# Add upstream remote
git remote add upstream https://github.com/originalowner/azure-pr-cli.git

# Install dependencies
make deps

# Run tests
make test

# Build
make build
```

## Coding Standards

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (`make fmt`)
- Run `go vet` before submitting (`make vet`)
- Use meaningful variable and function names
- Add comments for exported functions and types

### Testing

- Write unit tests for new functionality
- Aim for at least 80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

Example test structure:
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        // test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters
- Reference issues and pull requests

Example:
```
Add CSV output formatter

- Implement CSVFormatter with Format method
- Add tests for CSV formatting
- Update documentation

Fixes #123
```

## Project Structure

```
azure-pr-cli/
├── cmd/                    # Command definitions (Cobra)
├── internal/              # Private application code
│   ├── client/           # Azure DevOps API client
│   ├── config/           # Configuration management
│   ├── formatter/        # Output formatters
│   └── models/           # Data models
├── test/                 # Additional test files
└── main.go              # Application entry point
```

## Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/client -v

# Run linter
make lint
```

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update examples when changing behavior
- Keep CHANGELOG.md updated

## Release Process

Releases are managed by maintainers:

1. Update version in appropriate files
2. Update CHANGELOG.md
3. Create and push a version tag
4. GitHub Actions will build and create the release

## Questions?

Feel free to open an issue for questions or join our discussions.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
