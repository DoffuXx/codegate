package interfaces

import (
	"context"

	"codegate/pkg/models"
)

type ProviderConfig struct {
	Type    string                 `mapstructure:"type"`
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

type AIProvider interface {
	Name() string
	ValidateConfig() error
	Audit(ctx context.Context, request models.AuditRequest) (*models.AuditResponse, error)
	GetCapabilities() ProviderCapabilities
	EstimateCost(request models.AuditRequest) float64
	Close() error
}

type ProviderCapabilities struct {
	SupportsStreaming  bool                `json:"supports_streaming"`
	MaxTokens          int                 `json:"max_tokens"`
	SupportedLanguages []string            `json:"supported_languages"`
	SupportedFoci      []models.AuditFocus `json:"supported_foci"`
	RequiresAuth       bool                `json:"requires_auth"`
	IsLocal            bool                `json:"is_local"`
	HasCost            bool                `json:"has_cost"`
}
