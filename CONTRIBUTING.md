# Contributing to CodeGate

Thank you for your interest in contributing to CodeGate! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, professional, and inclusive. We're all here to build something useful together.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear description** of the issue
- **Steps to reproduce** the behavior
- **Expected behavior**
- **Actual behavior**
- **Environment details** (OS, Go version, AI provider)
- **Logs** (run with `--verbose` flag)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- Use a **clear and descriptive title**
- Provide a **detailed description** of the proposed feature
- Explain **why this enhancement would be useful**
- Include **examples** of how it would work

### Pull Requests

1. **Fork the repo** and create your branch from `main`
2. **Follow the coding style** of the project
3. **Write clear commit messages**
4. **Update documentation** if you're changing functionality
5. **Add tests** if applicable
6. **Ensure all tests pass**

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Git
- LM Studio (for testing) or another AI provider

### Setup

```bash
# Clone your fork
git clone https://github.com/yourusername/codegate.git
cd codegate

# Install dependencies
go mod download

# Build
go build -o codegate .

# Run tests
go test ./...
```

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` to format your code
- Run `go vet` before committing
- Keep functions focused and small
- Add comments for exported functions and complex logic

### Project Structure

- **`cmd/`**: CLI commands only, minimal logic
- **`internal/`**: Implementation details, not importable by other projects
- **`pkg/`**: Public packages, can be imported by other projects
- **`templates/`**: Embedded templates

### Adding a New AI Provider

To add support for a new AI provider:

1. Create package in `internal/providers/<provider-name>/`
2. Create `config.go` with provider configuration struct
3. Create `client.go` implementing the `AIProvider` interface:
   ```go
   type AIProvider interface {
       Name() string
       ValidateConfig() error
       Audit(ctx context.Context, request models.AuditRequest) (*models.AuditResponse, error)
       GetCapabilities() ProviderCapabilities
       EstimateCost(request models.AuditRequest) float64
       Close() error
   }
   ```
4. Register in `init()`:
   ```go
   func init() {
       registry.RegisterProvider("provider-name", NewProvider)
   }
   ```
5. Add blank import to `internal/providers/registry.go`:
   ```go
   _ "codegate/internal/providers/<provider-name>"
   ```
6. Update README.md with provider setup instructions

### Commit Messages

Follow conventional commits:

```
type(scope): subject

body (optional)

footer (optional)
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(providers): add OpenAI provider support
fix(git): handle detached HEAD state correctly
docs(readme): update installation instructions
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/git

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### Writing Tests

- Place test files next to the code they test (`foo.go` → `foo_test.go`)
- Use table-driven tests when testing multiple scenarios
- Mock external dependencies (AI APIs, git commands)
- Test error cases, not just happy paths

## Documentation

- Update README.md if you change user-facing functionality
- Update CLAUDE.md if you change architecture
- Add godoc comments for exported functions
- Update configuration examples if you add new config options

## Review Process

1. Create a pull request with a clear description
2. Ensure CI passes (if configured)
3. Address review feedback
4. Maintainer will merge once approved

## Questions?

Feel free to open an issue with the `question` label if you have questions about contributing.

Thank you for contributing! 🎉
