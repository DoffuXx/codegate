package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// cfgFile holds the path to the configuration file
	// This allows users to specify custom config locations
	cfgFile string

	// version information - set by main.go
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands
// This is the foundation that all other commands build upon
var rootCmd = &cobra.Command{
	Use:   "codegate",
	Short: "AI-powered code review before you commit",
	Long: `CodeGate analyzes your staged changes using AI to identify potential
issues before you commit. It supports multiple AI providers including
local LLMs (LM Studio) and cloud services (OpenAI, Claude).

CodeGate integrates seamlessly with your git workflow and can be
configured as a pre-commit hook for automated code review.`,

	// This function runs for commands that don't match any subcommands
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, show the help
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately
// This is called by main.go and serves as the entry point for the CLI
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo allows main.go to inject build-time version information
// This is a clean way to pass build metadata into the CLI commands
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d

	// Update the version command with the actual build information
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}

// init function sets up the CLI configuration and global flags
// This runs automatically when the package is imported
func init() {
	// Set up configuration loading before any commands run
	cobra.OnInitialize(initConfig)

	// Add global flags that work across all subcommands
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.codegate.yaml)")

	// Add a verbose flag for debugging
	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"enable verbose output for debugging")

	// Bind the verbose flag to viper so it can be used in config
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables
// This function sets up the configuration system using Viper
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag if provided
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory for default config location
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search for config in home directory
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".codegate")
	}

	// Enable reading from environment variables
	// This allows users to override config with ENV vars like CODEGATE_PROVIDER
	viper.SetEnvPrefix("CODEGATE")
	viper.AutomaticEnv()

	// Attempt to read the configuration file
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
	} else {
		// If no config file exists, we'll use defaults
		// This is normal for first-time users
		if viper.GetBool("verbose") {
			fmt.Fprintf(os.Stderr, "No config file found, using defaults\n")
		}
	}

	// Set up default configuration values
	setDefaults()
}

// setDefaults establishes sensible default values for all configuration options
// This ensures the tool works out of the box without requiring configuration
func setDefaults() {
	// Default AI provider configuration
	viper.SetDefault("providers.lmstudio.enabled", true)
	viper.SetDefault("providers.lmstudio.config.base_url", "http://localhost:1234")
	viper.SetDefault("providers.lmstudio.config.max_tokens", 2048)
	viper.SetDefault("providers.lmstudio.config.temperature", 0.1)
	viper.SetDefault("providers.lmstudio.config.timeout_seconds", 120)
	viper.SetDefault("providers.lmstudio.config.enable_streaming", true)

	// Default audit settings
	viper.SetDefault("audit.default_provider", "lmstudio")
	viper.SetDefault("audit.default_focus", []string{"bugs", "security", "performance"})
	viper.SetDefault("audit.auto_detect_language", true)

	// Default output settings
	viper.SetDefault("output.format", "table")
	viper.SetDefault("output.show_suggestions", true)
	viper.SetDefault("output.group_by_severity", true)
}
