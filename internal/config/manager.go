// This package handles loading, validating, and providing access to user settings
package config

import (
	"fmt"

	"codegate/internal/providers"
	"codegate/pkg/models"

	"github.com/spf13/viper"
)

// Manager handles all configuration operations for the application
// This centralizes config logic and provides a clean interface for other components
type Manager struct {
	// We could add state here if needed for caching or validation
}

// NewManager creates a new configuration manager instance
func NewManager() *Manager {
	return &Manager{}
}

// GetActiveProvider returns the configured and initialized AI provider
// This method handles the complex logic of reading config and creating provider instances
func (m *Manager) GetActiveProvider() (providers.AIProvider, error) {
	// Get the provider type from configuration
	providerType := viper.GetString("audit.default_provider")
	if providerType == "" {
		return nil, fmt.Errorf("no default provider configured")
	}

	// Check if the provider is enabled
	enabledKey := fmt.Sprintf("providers.%s.enabled", providerType)
	if !viper.GetBool(enabledKey) {
		return nil, fmt.Errorf("provider '%s' is not enabled", providerType)
	}

	// Load the provider-specific configuration
	configKey := fmt.Sprintf("providers.%s.config", providerType)
	providerConfig := providers.ProviderConfig{
		Type:    providerType,
		Enabled: true,
		Config:  viper.GetStringMap(configKey),
	}

	// Create and return the provider instance
	provider, err := providers.CreateProvider(providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider '%s': %w", providerType, err)
	}

	return provider, nil
}

// GetAuditSettings returns the default audit configuration
// This provides consistent settings across all audit operations
func (m *Manager) GetAuditSettings() AuditSettings {
	// Convert string slice to typed slice for focus areas
	focusStrings := viper.GetStringSlice("audit.default_focus")
	focus := make([]models.AuditFocus, len(focusStrings))
	for i, f := range focusStrings {
		focus[i] = models.AuditFocus(f)
	}

	return AuditSettings{
		DefaultFocus:       focus,
		AutoDetectLanguage: viper.GetBool("audit.auto_detect_language"),
		MaxTokens:          viper.GetInt("audit.max_tokens"),
		Temperature:        viper.GetFloat64("audit.temperature"),
	}
}

// GetOutputSettings returns the configured output formatting options
func (m *Manager) GetOutputSettings() OutputSettings {
	return OutputSettings{
		Format:          viper.GetString("output.format"),
		ShowSuggestions: viper.GetBool("output.show_suggestions"),
		GroupBySeverity: viper.GetBool("output.group_by_severity"),
		Verbose:         viper.GetBool("verbose"),
	}
}

// AuditSettings holds configuration for how audits are performed
type AuditSettings struct {
	DefaultFocus       []models.AuditFocus
	AutoDetectLanguage bool
	MaxTokens          int
	Temperature        float64
}

// OutputSettings controls how results are displayed to users
type OutputSettings struct {
	Format          string // "table", "json", "markdown"
	ShowSuggestions bool
	GroupBySeverity bool
	Verbose         bool
}

// ValidateConfiguration checks that all settings are valid and providers are working
// This is useful for the config command to verify user settings
func (m *Manager) ValidateConfiguration() error {
	// Test the active provider
	provider, err := m.GetActiveProvider()
	if err != nil {
		return fmt.Errorf("active provider validation failed: %w", err)
	}
	defer provider.Close()

	// Validate audit settings
	settings := m.GetAuditSettings()
	if len(settings.DefaultFocus) == 0 {
		return fmt.Errorf("at least one focus area must be specified")
	}

	if settings.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if settings.Temperature < 0 || settings.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	return nil
}
