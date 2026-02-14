# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- MIT License for open source distribution
- Comprehensive README.md with installation and usage instructions
- CONTRIBUTING.md guide for contributors
- CLAUDE.md architecture documentation for AI assistance
- `.gitignore` for build artifacts and IDE files
- Example configuration file (`.codegate.example.yaml`)
- `version` command to display build information
- `init` command for easy first-time configuration setup
- Makefile with common development tasks (build, test, lint, etc.)
- Basic test suite:
  - `internal/git/diff_test.go` - Git URL parsing tests
  - `internal/providers/lmstudio/config_test.go` - Config validation tests
  - `pkg/models/providers_test.go` - Model serialization tests
- Version information support via build-time ldflags
- Cross-platform debug log path using `os.TempDir()`

### Changed
- Hardcoded `/tmp/` paths replaced with platform-agnostic temporary directory
- Debug log file now uses `filepath.Join(os.TempDir(), "codegate-json-debug.log")`
- System prompt is now documented and configurable via config file
- Improved error messages with more context

### Fixed
- Cross-platform compatibility for debug log paths (Windows, macOS, Linux)

## [0.1.0] - Initial Development

### Added
- Core CLI structure using Cobra
- LM Studio provider implementation
- Git diff extraction and analysis
- Multiple output formats (table, JSON, markdown)
- Browser preview mode
- Configuration management with Viper
- Streaming response support
- Language auto-detection
- Provider factory pattern with registry
