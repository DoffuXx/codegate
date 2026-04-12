# CodeGate

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://go.dev/)

AI-powered git commit auditing tool that analyzes your staged changes using local or cloud AI providers to identify potential issues before you commit.

## Features

- 🔍 **Automated Code Review**: Analyzes staged changes for bugs, security issues, performance problems, and more
- 🤖 **Multiple AI Providers**: Support for local LLMs (LM Studio) with plans for OpenAI and Claude
- 🌐 **Markdown Preview by Default**: Analysis opens automatically in your browser or configured markdown viewer
- 📊 **Multiple Output Formats**: Table, JSON, and Markdown to stdout (opt-in with `--no-preview` or `--format`)
- ⚙️ **Configurable Focus Areas**: Security, performance, bugs, maintainability, style, documentation
- 🔧 **Language Detection**: Automatic programming language detection for context-aware analysis
- 🎨 **Beautiful Terminal Output**: Color-coded severity levels and formatted tables

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/DoffuXx/codegate.git
cd codegate

# Build the binary
go build -o codegate .

# Move to PATH (optional)
sudo mv codegate /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/DoffuXx/codegate@latest
```

## Quick Start

1. **Setup LM Studio** (or another AI provider):
   - Download and run [LM Studio](https://lmstudio.ai/)
   - Load a model (e.g., Qwen, Llama, Mistral)
   - Start the local server (default: `http://localhost:1234`)

2. **Stage your changes**:

   ```bash
   git add .
   ```

3. **Run the audit** (opens markdown preview in browser by default):
   ```bash
   codegate review
   ```

## Usage

### Basic Review

```bash
# Analyze all staged changes (opens markdown preview in browser by default)
codegate review

# Disable preview mode and show table output in terminal
codegate review --no-preview
```

### Focus on Specific Areas

```bash
# Focus on security and bugs only
codegate review --focus security,bugs

# Available focus areas: performance, security, bugs, maintainability, style, documentation
```

### Different Output Formats

```bash
# Table output in terminal (disables preview)
codegate review --no-preview

# JSON output (disables preview)
codegate review --format json

# Markdown output to stdout (disables preview)
codegate review --format markdown
```

### Provider Selection

```bash
# Use a specific provider (when multiple are configured)
codegate review --provider lmstudio
```

### Custom Markdown Viewer

```bash
# Configure a custom markdown viewer (e.g., glow, mdcat) in ~/.codegate.yaml
output:
  preview_command: "glow"  # CLI markdown viewer instead of browser

# Or set via environment variable
export CODEGATE_OUTPUT_PREVIEW_COMMAND="mdcat"
```

### Verbose Output

```bash
# Enable debug logging
codegate review --verbose
```

## Configuration

Create a configuration file at `~/.codegate.yaml`:

```yaml
# AI Provider Configuration
providers:
  lmstudio:
    enabled: true
    config:
      base_url: http://localhost:1234
      max_tokens: 2048
      temperature: 0.1
      timeout_seconds: 120
      enable_streaming: true

# Audit Settings
audit:
  default_provider: lmstudio
  default_focus:
    - bugs
    - security
    - performance
  auto_detect_language: true

# Output Settings
output:
  format: table # table, json, or markdown (only applies when preview is disabled)
  show_suggestions: true
  group_by_severity: true
  preview_command: "" # optional: custom markdown viewer (e.g., "glow", "mdcat")
                      # if not set, uses system default (xdg-open/open/start)
```

### Environment Variables

Override config values using environment variables with the `CODEGATE_` prefix:

```bash
export CODEGATE_PROVIDERS_LMSTUDIO_CONFIG_BASE_URL="http://localhost:8080"
export CODEGATE_AUDIT_DEFAULT_PROVIDER="lmstudio"
```

## Adding New AI Providers

CodeGate uses a provider factory pattern. To add a new provider:

1. Create a new package in `internal/providers/<name>/`
2. Implement the `AIProvider` interface
3. Register the provider in your `init()` function
4. Add a blank import to `internal/providers/registry.go`

See the LM Studio implementation in `internal/providers/lmstudio/` for reference.

## Architecture

```
codegate/
├── cmd/              # CLI commands (Cobra)
├── internal/         # Internal packages
│   ├── config/       # Configuration management
│   ├── git/          # Git operations
│   └── providers/    # AI provider implementations
├── pkg/              # Public packages
│   ├── interfaces/   # Provider interfaces
│   ├── models/       # Data models
│   └── registry/     # Provider registry
└── templates/        # Embedded AI prompt templates
```

For detailed architecture documentation, see [CLAUDE.md](CLAUDE.md).

## Development

### Building

```bash
go build -o codegate .
```

### Running Tests

```bash
go test ./...
```

### Running Without Building

```bash
go run . review
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Roadmap

- [ ] Add OpenAI provider
- [ ] Add Anthropic Claude provider
- [ ] Add GitHub Copilot integration
- [ ] Git hooks integration (pre-commit)
- [ ] CI/CD pipeline integration
- [ ] Auto-fix suggestions
- [ ] Interactive mode
- [ ] Historical analysis tracking

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Powered by [Viper](https://github.com/spf13/viper) for configuration
- Uses [LM Studio](https://lmstudio.ai/) for local AI inference

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/DoffuXx/codegate).
