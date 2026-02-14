package providers

import (
	"fmt"

	"codegate/pkg/interfaces"
	"codegate/pkg/registry"
)

type (
	ProviderConfig       = interfaces.ProviderConfig
	AIProvider           = interfaces.AIProvider
	ProviderCapabilities = interfaces.ProviderCapabilities
)

// ProviderConstructor is a function signature for creating new provider instances
type ProviderConstructor func(config ProviderConfig) (AIProvider, error)

// CreateProvider instantiates a provider based on its configuration
func CreateProvider(config ProviderConfig) (AIProvider, error) {
	registryProviders := registry.GetProviders()
	constructorInterface, exists := registryProviders[config.Type]
	if !exists {
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}

	constructor := constructorInterface.(func(ProviderConfig) (AIProvider, error))

	provider, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider '%s': %w", config.Type, err)
	}

	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider '%s' configuration invalid: %w", config.Type, err)
	}

	return provider, nil
}

// GetAvailableProviders returns a list of all registered provider names
func GetAvailableProviders() []string {
	registryProviders := registry.GetProviders()
	names := make([]string, 0, len(registryProviders))
	for name := range registryProviders {
		names = append(names, name)
	}
	return names
}

// GetProviderCapabilities returns the capabilities of a specific provider type
func GetProviderCapabilities(providerType string) (ProviderCapabilities, error) {
	tempConfig := ProviderConfig{Type: providerType, Enabled: true}
	provider, err := CreateProvider(tempConfig)
	if err != nil {
		return ProviderCapabilities{}, err
	}
	defer provider.Close()
	return provider.GetCapabilities(), nil
}
