package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"codegate/pkg/interfaces"
	"codegate/pkg/models"
)

// ── Config tests ─────────────────────────────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != "https://api.openai.com" {
		t.Errorf("expected default base_url 'https://api.openai.com', got %q", cfg.BaseURL)
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %q", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.1 {
		t.Errorf("expected default temperature 0.1, got %f", cfg.Temperature)
	}
	if cfg.TimeoutSeconds != 120 {
		t.Errorf("expected default timeout_seconds 120, got %d", cfg.TimeoutSeconds)
	}
	if cfg.SystemPrompt == "" {
		t.Error("expected non-empty default system_prompt")
	}
}

func TestConfigValidate(t *testing.T) {
	validBase := DefaultConfig()
	validBase.APIKey = "sk-test123"

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "valid config",
			mutate:  func(c *Config) {},
			wantErr: "",
		},
		{
			name:    "missing api_key",
			mutate:  func(c *Config) { c.APIKey = "" },
			wantErr: "api_key is required",
		},
		{
			name:    "whitespace-only api_key",
			mutate:  func(c *Config) { c.APIKey = "   " },
			wantErr: "api_key is required",
		},
		{
			name:    "missing base_url",
			mutate:  func(c *Config) { c.BaseURL = "" },
			wantErr: "base_url is required",
		},
		{
			name:    "invalid base_url scheme",
			mutate:  func(c *Config) { c.BaseURL = "ftp://api.openai.com" },
			wantErr: "base_url must use http or https scheme",
		},
		{
			name:    "missing model",
			mutate:  func(c *Config) { c.Model = "" },
			wantErr: "model is required",
		},
		{
			name:    "max_tokens too low",
			mutate:  func(c *Config) { c.MaxTokens = 0 },
			wantErr: "max_tokens must be between",
		},
		{
			name:    "max_tokens too high",
			mutate:  func(c *Config) { c.MaxTokens = 200000 },
			wantErr: "max_tokens must be between",
		},
		{
			name:    "temperature negative",
			mutate:  func(c *Config) { c.Temperature = -0.1 },
			wantErr: "temperature must be between",
		},
		{
			name:    "temperature too high",
			mutate:  func(c *Config) { c.Temperature = 2.5 },
			wantErr: "temperature must be between",
		},
		{
			name:    "timeout too low",
			mutate:  func(c *Config) { c.TimeoutSeconds = 0 },
			wantErr: "timeout_seconds must be between",
		},
		{
			name:    "timeout too high",
			mutate:  func(c *Config) { c.TimeoutSeconds = 9999 },
			wantErr: "timeout_seconds must be between",
		},
		{
			name:    "empty system_prompt",
			mutate:  func(c *Config) { c.SystemPrompt = "   " },
			wantErr: "system_prompt cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBase
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestGetEndpoint(t *testing.T) {
	tests := []struct {
		baseURL  string
		path     string
		expected string
	}{
		{"https://api.openai.com", "v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/", "/v1/models", "https://api.openai.com/v1/models"},
		{"http://localhost:8080", "v1/chat/completions", "http://localhost:8080/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.baseURL+"+"+tt.path, func(t *testing.T) {
			cfg := &Config{BaseURL: tt.baseURL}
			got := cfg.GetEndpoint(tt.path)
			if got != tt.expected {
				t.Errorf("GetEndpoint(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// ── NewProvider tests ─────────────────────────────────────────────────────────

func TestNewProvider_AppliesConfig(t *testing.T) {
	cfg := interfaces.ProviderConfig{
		Type:    "openai",
		Enabled: true,
		Config: map[string]interface{}{
			"api_key":         "sk-test",
			"model":           "gpt-3.5-turbo",
			"max_tokens":      float64(1024),
			"temperature":     float64(0.5),
			"timeout_seconds": float64(60),
		},
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}

	client := provider.(*Client)
	if client.config.APIKey != "sk-test" {
		t.Errorf("expected api_key 'sk-test', got %q", client.config.APIKey)
	}
	if client.config.Model != "gpt-3.5-turbo" {
		t.Errorf("expected model 'gpt-3.5-turbo', got %q", client.config.Model)
	}
	if client.config.MaxTokens != 1024 {
		t.Errorf("expected max_tokens 1024, got %d", client.config.MaxTokens)
	}
}

func TestNewProvider_UsesDefaults(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-env-key")

	provider, err := NewProvider(interfaces.ProviderConfig{Type: "openai", Enabled: true})
	if err != nil {
		t.Fatalf("NewProvider returned error: %v", err)
	}

	client := provider.(*Client)
	if client.config.Model != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %q", client.config.Model)
	}
	if client.config.APIKey != "sk-env-key" {
		t.Errorf("expected api_key from env 'sk-env-key', got %q", client.config.APIKey)
	}
}

func TestName(t *testing.T) {
	c := &Client{}
	if c.Name() != "OpenAI" {
		t.Errorf("expected Name() == 'OpenAI', got %q", c.Name())
	}
}

// ── ValidateConfig with mock server ──────────────────────────────────────────

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := DefaultConfig()
	cfg.APIKey = "sk-test"
	cfg.BaseURL = srv.URL

	client := &Client{
		config:     cfg,
		httpClient: srv.Client(),
	}
	return client, srv
}

func TestValidateConfig_Success(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"object":"list","data":[]}`)
		}
	})

	if err := client.ValidateConfig(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateConfig_Unauthorized(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	err := client.ValidateConfig()
	if err == nil || !strings.Contains(err.Error(), "invalid OpenAI API key") {
		t.Errorf("expected unauthorized error, got: %v", err)
	}
}

func TestValidateConfig_InvalidAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIKey = ""
	client := &Client{config: cfg, httpClient: &http.Client{}}

	err := client.ValidateConfig()
	if err == nil || !strings.Contains(err.Error(), "api_key is required") {
		t.Errorf("expected api_key error, got: %v", err)
	}
}

// ── Audit with mock server ────────────────────────────────────────────────────

func validAuditJSON() string {
	return `{
		"summary": "Looks good",
		"issues": [
			{
				"severity": "high",
				"category": "security",
				"title": "SQL Injection",
				"description": "User input not sanitised",
				"line_number": 42,
				"file_name": "main.go",
				"code_snippet": "db.Query(input)",
				"recommended_fix": "Use parameterised queries"
			}
		],
		"suggestions": ["Add tests"],
		"proposed_commit_title": "fix: sanitise SQL input",
		"proposed_commit_body": "Replaced raw query with parameterised version"
	}`
}

func TestAudit_StandardResponse(t *testing.T) {
	auditJSON := validAuditJSON()

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": auditJSON}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	client.config.EnableStreaming = false

	req := models.AuditRequest{Diff: "diff --git a/main.go b/main.go\n+db.Query(input)"}
	result, err := client.Audit(context.Background(), req)
	if err != nil {
		t.Fatalf("Audit returned error: %v", err)
	}

	if result.Summary != "Looks good" {
		t.Errorf("unexpected summary: %q", result.Summary)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Title != "SQL Injection" {
		t.Errorf("unexpected issue title: %q", result.Issues[0].Title)
	}
	if result.Metadata.Provider != "OpenAI" {
		t.Errorf("unexpected provider in metadata: %q", result.Metadata.Provider)
	}
}

func TestAudit_APIError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, `{"error":{"message":"rate limited","type":"rate_limit_error"}}`)
	})
	client.config.EnableStreaming = false

	_, err := client.Audit(context.Background(), models.AuditRequest{Diff: "some diff"})
	if err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestAudit_StreamingResponse(t *testing.T) {
	auditJSON := validAuditJSON()

	// Build SSE stream
	chunks := []string{}
	for i, ch := range strings.Split(auditJSON, "") {
		_ = i
		chunks = append(chunks, ch)
	}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send the whole JSON as a single SSE chunk for simplicity
		chunk := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"delta":         map[string]string{"content": auditJSON},
					"finish_reason": "stop",
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintln(w, "data: [DONE]")
		if flusher != nil {
			flusher.Flush()
		}
	})
	client.config.EnableStreaming = true

	req := models.AuditRequest{Diff: "diff --git a/main.go b/main.go\n+db.Query(input)"}
	result, err := client.Audit(context.Background(), req)
	if err != nil {
		t.Fatalf("streaming Audit returned error: %v", err)
	}

	if result.Summary != "Looks good" {
		t.Errorf("unexpected summary from streaming: %q", result.Summary)
	}
}

// ── parseResponse ─────────────────────────────────────────────────────────────

func TestParseResponse_ValidJSON(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	result, err := c.parseResponse(validAuditJSON())
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if result.Summary != "Looks good" {
		t.Errorf("unexpected summary: %q", result.Summary)
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}
}

func TestParseResponse_JSONWrappedInMarkdown(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	wrapped := "```json\n" + validAuditJSON() + "\n```"
	result, err := c.parseResponse(wrapped)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	// Should fall through to fallback or repair — either way no crash
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestParseResponse_FallbackOnGarbage(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	result, err := c.parseResponse("this is not json at all")
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if result == nil {
		t.Fatal("expected fallback result, got nil")
	}
	if result.Summary == "" {
		t.Error("fallback result should have a non-empty summary")
	}
}

func TestParseResponse_NormalizesInvalidSeverity(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	raw := `{
		"summary": "test",
		"issues": [{"severity":"extreme","category":"bugs","title":"t","description":"d"}],
		"suggestions": [],
		"proposed_commit_title": "fix: x",
		"proposed_commit_body": "y"
	}`

	result, err := c.parseResponse(raw)
	if err != nil {
		t.Fatalf("parseResponse error: %v", err)
	}
	if result.Issues[0].Severity != models.SeverityMedium {
		t.Errorf("expected severity normalized to medium, got %q", result.Issues[0].Severity)
	}
}

// ── GetCapabilities ───────────────────────────────────────────────────────────

func TestGetCapabilities(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	caps := c.GetCapabilities()

	if !caps.RequiresAuth {
		t.Error("OpenAI provider should require auth")
	}
	if caps.IsLocal {
		t.Error("OpenAI provider should not be local")
	}
	if !caps.HasCost {
		t.Error("OpenAI provider should have cost")
	}
	if caps.MaxTokens != 4096 {
		t.Errorf("expected MaxTokens 4096, got %d", caps.MaxTokens)
	}
}

// ── EstimateCost ──────────────────────────────────────────────────────────────

func TestEstimateCost_PositiveValue(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	req := models.AuditRequest{Diff: strings.Repeat("x", 10000)}
	cost := c.EstimateCost(req)
	if cost <= 0 {
		t.Errorf("expected positive cost estimate, got %f", cost)
	}
}

func TestEstimateCost_LargerDiffCostsMore(t *testing.T) {
	c := &Client{config: DefaultConfig()}
	c.config.APIKey = "sk-test"

	small := c.EstimateCost(models.AuditRequest{Diff: strings.Repeat("x", 100)})
	large := c.EstimateCost(models.AuditRequest{Diff: strings.Repeat("x", 100000)})
	if large <= small {
		t.Errorf("larger diff should cost more: small=%f, large=%f", small, large)
	}
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestClose(t *testing.T) {
	c := &Client{}
	if err := c.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
