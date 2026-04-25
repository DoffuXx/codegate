// Package providers - registry imports all provider implementations
// This ensures all providers are registered when the providers package is used
package providers

import (
	// Import all provider implementations for their side effects
	// Each provider's init() function registers itself with the factory
	_ "codegate/internal/providers/lmstudio"
	// Future providers would be added here:
	_ "codegate/internal/providers/openai"
	// _ "codegate/internal/providers/claude"
)
