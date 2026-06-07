// Package openai implements the AIProvider interface for OpenAI API
package openai

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config holds OpenAI-specific configuration options
type Config struct {
	APIKey         string  `mapstructure:"api_key" json:"api_key"`
	BaseURL        string  `mapstructure:"base_url" json:"base_url"`
	Model          string  `mapstructure:"model" json:"model"`
	MaxTokens      int     `mapstructure:"max_tokens" json:"max_tokens"`
	Temperature    float64 `mapstructure:"temperature" json:"temperature"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds" json:"timeout_seconds"`
	SystemPrompt   string  `mapstructure:"system_prompt" json:"system_prompt"`
	EnableStreaming bool    `mapstructure:"enable_streaming" json:"enable_streaming"`
}

// DefaultConfig returns sensible defaults for OpenAI
func DefaultConfig() Config {
	return Config{
		APIKey:         os.Getenv("OPENAI_API_KEY"),
		BaseURL:        "https://api.openai.com",
		Model:          "gpt-4o",
		MaxTokens:      4096,
		Temperature:    0.1,
		TimeoutSeconds: 120,
		EnableStreaming: true,
		SystemPrompt: `You are an expert code reviewer. Analyze the provided git diff and respond with VALID JSON only.

REQUIRED JSON STRUCTURE:
{
    "summary": "Brief analysis summary",
    "issues": [
        {
            "severity": "high|medium|low|info",
            "category": "bugs|security|performance|style",
            "title": "Issue title",
            "description": "Detailed explanation",
            "line_number": 0,
            "file_name": "",
            "code_snippet": "",
            "recommended_fix": "How to fix this"
        }
    ],
    "suggestions": ["General suggestion 1", "General suggestion 2"],
    "proposed_commit_title": "fix: brief description",
    "proposed_commit_body": "What changed and why"
}

CRITICAL RULES:
1. Respond ONLY with valid JSON
2. Escape all strings properly (\n for newlines, \" for quotes)
3. No trailing commas
4. Include all required fields even if empty
5. Keep descriptions concise but informative`,
	}
}

// GetEndpoint constructs the full API endpoint URL for a specific path
func (c *Config) GetEndpoint(path string) string {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	path = strings.TrimLeft(path, "/")
	return fmt.Sprintf("%s/%s", baseURL, path)
}

// ToRequestConfig converts this config to a format suitable for API requests
func (c *Config) ToRequestConfig() map[string]interface{} {
	return map[string]interface{}{
		"model":       c.Model,
		"max_tokens":  c.MaxTokens,
		"temperature": c.Temperature,
		"stream":      c.EnableStreaming,
	}
}

// Validate checks the config for required fields and valid values
func (c *Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("api_key is required (or set OPENAI_API_KEY env var)")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid base_url format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("base_url must use http or https scheme")
	}

	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model is required")
	}

	if c.MaxTokens <= 0 || c.MaxTokens > 128000 {
		return fmt.Errorf("max_tokens must be between 1 and 128000")
	}

	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0")
	}

	if c.TimeoutSeconds <= 0 || c.TimeoutSeconds > 3600 {
		return fmt.Errorf("timeout_seconds must be between 1 and 3600")
	}

	if strings.TrimSpace(c.SystemPrompt) == "" {
		return fmt.Errorf("system_prompt cannot be empty")
	}

	return nil
}
