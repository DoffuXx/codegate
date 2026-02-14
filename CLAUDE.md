# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`codegate` is an AI-powered git commit auditing tool that analyzes staged changes and provides code review feedback. It uses AI providers (currently LM Studio, with plans for OpenAI/Claude) to identify bugs, security issues, performance problems, and other code quality concerns before committing.

## Build & Run Commands

### Using Makefile (Recommended)
```bash
# Build with version information
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean

# Format and vet code
make fmt
make vet

# See all available targets
make help
```

### Manual Build
```bash
# Simple build
go build -o codegate .

# Build with version information
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o codegate .
```

### Run without building
```bash
go run . review
```

### First-time Setup
```bash
# Initialize configuration file
./codegate init

# Or specify custom output path
./codegate init -o /path/to/config.yaml
```

### Check Version
```bash
./codegate version
```

### Run with flags
```bash
# Analyze with specific focus areas
./codegate review --focus security,bugs

# Different output formats
./codegate review --format json
./codegate review --format markdown

# Generate markdown preview in browser
./codegate review --preview

# Use different provider (when implemented)
./codegate review --provider lmstudio

# Enable verbose logging
./codegate review --verbose
```

### Install dependencies
```bash
go mod download
go mod tidy
```

## Architecture

### Core Design Pattern: Provider Factory with Registry

The application uses a **provider factory pattern** with a global registry for AI provider implementations:

1. **Provider Registration** (`pkg/registry/registry.go`): Global map where providers register themselves
2. **Provider Interface** (`pkg/interfaces/provider.go`): `AIProvider` interface defines the contract
3. **Provider Factory** (`internal/providers/providers.go`): `CreateProvider()` looks up and instantiates providers
4. **Side-Effect Registration** (`internal/providers/registry.go`): Import providers with blank imports (`_ "codegate/internal/providers/lmstudio"`) to trigger their `init()` functions

When adding a new provider (e.g., OpenAI, Claude):
1. Create package in `internal/providers/<name>/`
2. Implement `AIProvider` interface
3. Register in `init()` function via `registry.RegisterProvider()`
4. Add blank import to `internal/providers/registry.go`

### Key Data Flow

```
User runs review command
  ↓
cmd/review.go extracts CLI flags
  ↓
internal/config/manager.go loads config (Viper)
  ↓
internal/git/diff.go extracts staged changes via `git diff --cached`
  ↓
internal/providers/providers.go creates AI provider instance
  ↓
Provider builds prompt + calls AI API (streaming or standard)
  ↓
Response parsed to models.AuditResponse
  ↓
cmd/review.go displays results (table/json/markdown)
```

### Package Structure

- **`cmd/`**: Cobra commands (root, review)
- **`internal/`**: Internal application code
  - `config/`: Configuration management (Viper wrapper)
  - `git/`: Git operations (diff extraction, branch info)
  - `providers/`: AI provider implementations
    - `registry.go`: Blank imports for provider registration
    - `providers.go`: Factory functions
    - `lmstudio/`: LM Studio provider implementation
- **`pkg/`**: Shared/public packages
  - `interfaces/`: Interface definitions (`AIProvider`)
  - `models/`: Data models (`AuditRequest`, `AuditResponse`, `AuditIssue`)
  - `registry/`: Global provider registry
- **`templates/`**: Embedded templates for AI prompts (using `//go:embed`)

### Configuration System

Configuration uses Viper with a hierarchical structure:
- Default config file: `$HOME/.codegate.yaml`
- Can override with `--config` flag
- Environment variables prefixed with `CODEGATE_` override config
- Defaults set in `cmd/root.go:setDefaults()`

Example config structure:
```yaml
providers:
  lmstudio:
    enabled: true
    config:
      base_url: http://localhost:1234
      max_tokens: 2048
      temperature: 0.1
audit:
  default_provider: lmstudio
  default_focus: [bugs, security, performance]
output:
  format: table
  show_suggestions: true
```

### Preview Mode

The `--preview` flag triggers a different workflow:
1. Uses markdown template from `templates/preview.md` (embedded)
2. AI generates markdown directly (not structured JSON)
3. Saves to `/tmp/codegate-<timestamp>.md`
4. Opens in system browser (platform-specific: `xdg-open`, `open`, `start`)

### Response Parsing Strategy

LM Studio client uses tiered parsing (`internal/providers/lmstudio/client.go:parseResponse`):
1. Try parsing raw JSON with `gjson`
2. Apply simple repairs (trim, balance braces, remove trailing commas)
3. Extract fields with `gjson` (robust partial parsing)
4. Fallback to basic response if parsing fails completely
5. Normalize response (default empty fields, validate enums)

This handles imperfect JSON from local LLMs.

## Important Implementation Details

### Adding New AI Providers

Follow the LM Studio implementation pattern:
1. Create `internal/providers/<name>/` directory
2. Create `config.go` with provider-specific configuration
3. Create `client.go` implementing `AIProvider` interface:
   - `Name() string`
   - `ValidateConfig() error`
   - `Audit(ctx, request) (*AuditResponse, error)`
   - `GetCapabilities() ProviderCapabilities`
   - `EstimateCost(request) float64`
   - `Close() error`
4. In `init()`, call `registry.RegisterProvider("name", NewProvider)`
5. Add blank import to `internal/providers/registry.go`

### Git Context Extraction

`internal/git/diff.go:ExtractStagedChanges()` returns:
- Diff from `git diff --cached`
- Current branch name
- Short commit hash (7 chars)
- Repository name (from origin URL)

This context helps AI provide more relevant feedback.

### Language Detection

`cmd/review.go:detectLanguageFromDiff()` parses file extensions from diff headers and returns the most common language (if >50% confidence). This is passed to AI for language-specific analysis.

### Templates

Templates use `//go:embed` directive in `templates/templates.go` to embed `preview.md` at compile time. Access via `templates.Preview` constant.

## Testing

### Running Tests

```bash
# Run all tests
go test ./...
make test

# Run specific package
go test ./internal/git -v
go test ./pkg/models -v

# Run with coverage
go test -cover ./...
make test-coverage
```

### Test Structure

Tests are located alongside the code they test (e.g., `config.go` → `config_test.go`). Use table-driven tests for multiple scenarios:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := functionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Existing Test Coverage

- `internal/git/diff_test.go`: Tests for git URL parsing
- `internal/providers/lmstudio/config_test.go`: Config validation tests
- `pkg/models/providers_test.go`: Model unmarshaling and markdown generation tests

### Adding Tests for New Providers

When adding a new provider, include tests for:
- Config validation
- Endpoint URL construction
- Error handling
- Response parsing
