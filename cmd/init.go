package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// initCmd represents the init command that helps users set up configuration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CodeGate configuration",
	Long: `Create a sample configuration file to help you get started with CodeGate.

This command creates a .codegate.yaml file in your home directory with
sensible defaults for LM Studio. You can edit this file to customize the
behavior of CodeGate.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolP("force", "f", false, "overwrite existing config file")
	initCmd.Flags().StringP("output", "o", "", "output path (default: $HOME/.codegate.yaml)")
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	outputPath, _ := cmd.Flags().GetString("output")

	// Determine output path
	if outputPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputPath = filepath.Join(home, ".codegate.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Create sample config
	sampleConfig := `# CodeGate Configuration
# For more details, see: https://github.com/yourusername/codegate
# AI Provider Configuration
providers:
  lmstudio:
    enabled: true
    config:
      # LM Studio server URL
      base_url: http://localhost:1234
      # Maximum tokens to generate
      max_tokens: 2048
      # Temperature (0.0 = deterministic, 2.0 = creative)
      temperature: 0.1
      # Request timeout in seconds
      timeout_seconds: 120
      # Enable streaming for faster feedback
      enable_streaming: true
      # Optional: Specify model (auto-detected if empty)
      # model: "qwen2.5-coder-7b-instruct"
# Audit Settings
audit:
  # Default AI provider
  default_provider: lmstudio
  # Default focus areas
  # Options: performance, security, bugs, maintainability, style, documentation
  default_focus:
    - bugs
    - security
    - performance
  # Auto-detect programming language
  auto_detect_language: true
# Output Settings
output:
  # Default format: table, json, or markdown
  format: table
  # Show AI suggestions
  show_suggestions: true
  # Group issues by severity
  group_by_severity: true
  # Custom command for preview mode (empty = use system default)
  # Examples: "glow", "bat", "code", "vim"
  preview_command: ""
# Verbose logging (useful for debugging)
# verbose: false
`

	// Write config file
	if err := os.WriteFile(outputPath, []byte(sampleConfig), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Configuration file created: %s\n\n", outputPath)
	fmt.Println("Next steps:")
	fmt.Println("1. Start LM Studio and load a model")
	fmt.Println("2. Start the LM Studio server (default: http://localhost:1234)")
	fmt.Println("3. Stage some changes: git add <files>")
	fmt.Println("4. Run: codegate review")
	fmt.Println("\nFor more help, run: codegate --help")

	return nil
}
