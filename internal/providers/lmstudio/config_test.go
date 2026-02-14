package lmstudio

import (
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid default config",
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name: "Empty base URL",
			config: Config{
				BaseURL:        "",
				MaxTokens:      2048,
				Temperature:    0.1,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "base_url is required",
		},
		{
			name: "Invalid URL scheme",
			config: Config{
				BaseURL:        "ftp://localhost:1234",
				MaxTokens:      2048,
				Temperature:    0.1,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "must use http or https scheme",
		},
		{
			name: "Invalid max tokens - too high",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      200000,
				Temperature:    0.1,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "max_tokens must be between",
		},
		{
			name: "Invalid max tokens - zero",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      0,
				Temperature:    0.1,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "max_tokens must be between",
		},
		{
			name: "Invalid temperature - too high",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      2048,
				Temperature:    3.0,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "temperature must be between",
		},
		{
			name: "Invalid temperature - negative",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      2048,
				Temperature:    -0.5,
				TimeoutSeconds: 120,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "temperature must be between",
		},
		{
			name: "Invalid timeout - too high",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      2048,
				Temperature:    0.1,
				TimeoutSeconds: 4000,
				SystemPrompt:   "test prompt",
			},
			wantError: true,
			errorMsg:  "timeout_seconds must be between",
		},
		{
			name: "Empty system prompt",
			config: Config{
				BaseURL:        "http://localhost:1234",
				MaxTokens:      2048,
				Temperature:    0.1,
				TimeoutSeconds: 120,
				SystemPrompt:   "",
			},
			wantError: true,
			errorMsg:  "system_prompt cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "Basic endpoint",
			baseURL:  "http://localhost:1234",
			path:     "v1/models",
			expected: "http://localhost:1234/v1/models",
		},
		{
			name:     "Base URL with trailing slash",
			baseURL:  "http://localhost:1234/",
			path:     "v1/models",
			expected: "http://localhost:1234/v1/models",
		},
		{
			name:     "Path with leading slash",
			baseURL:  "http://localhost:1234",
			path:     "/v1/models",
			expected: "http://localhost:1234/v1/models",
		},
		{
			name:     "Both with slashes",
			baseURL:  "http://localhost:1234/",
			path:     "/v1/models",
			expected: "http://localhost:1234/v1/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{BaseURL: tt.baseURL}
			result := config.GetEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("GetEndpoint() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test that default config is valid
	if err := config.Validate(); err != nil {
		t.Errorf("DefaultConfig() returned invalid config: %v", err)
	}

	// Test expected defaults
	if config.BaseURL != "http://localhost:1234" {
		t.Errorf("DefaultConfig().BaseURL = %q, want %q", config.BaseURL, "http://localhost:1234")
	}

	if config.Temperature != 0.1 {
		t.Errorf("DefaultConfig().Temperature = %f, want %f", config.Temperature, 0.1)
	}

	if config.EnableStreaming != true {
		t.Errorf("DefaultConfig().EnableStreaming = %v, want %v", config.EnableStreaming, true)
	}

	if config.SystemPrompt == "" {
		t.Error("DefaultConfig().SystemPrompt should not be empty")
	}
}
