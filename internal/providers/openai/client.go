package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"codegate/pkg/interfaces"
	"codegate/pkg/models"
	"codegate/pkg/prompts"
	"codegate/pkg/registry"
	"codegate/templates"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

const (
	maxPayloadSize = 10 * 1024 * 1024 // 10MB

	// GPT-4o pricing (per 1M tokens, as of 2024)
	costPerInputToken  = 5.0 / 1_000_000  // $5 per 1M input tokens
	costPerOutputToken = 15.0 / 1_000_000 // $15 per 1M output tokens
)

// Client implements the AIProvider interface for OpenAI
type Client struct {
	config     Config
	httpClient *http.Client
}

func init() {
	registry.RegisterProvider("openai", NewProvider)
}

// NewProvider creates a new OpenAI provider instance from the given config
func NewProvider(config interfaces.ProviderConfig) (interfaces.AIProvider, error) {
	oaiConfig := DefaultConfig()

	if config.Config != nil {
		configBytes, err := json.Marshal(config.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := json.Unmarshal(configBytes, &oaiConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OpenAI config: %w", err)
		}
	}

	httpClient := &http.Client{
		Timeout: time.Duration(oaiConfig.TimeoutSeconds) * time.Second,
	}

	return &Client{
		config:     oaiConfig,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Name() string {
	return "OpenAI"
}

// ValidateConfig checks the config and verifies connectivity to the OpenAI API
func (c *Client) ValidateConfig() error {
	if err := c.config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endpoint := c.config.GetEndpoint("v1/models")
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OpenAI at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid OpenAI API key (401 Unauthorized)")
	case http.StatusForbidden:
		return fmt.Errorf("OpenAI API key lacks required permissions (403 Forbidden)")
	default:
		return fmt.Errorf("OpenAI API returned unexpected status %d", resp.StatusCode)
	}
}

// Audit sends the staged diff to OpenAI and returns an AuditResponse
func (c *Client) Audit(ctx context.Context, request models.AuditRequest) (*models.AuditResponse, error) {
	startTime := time.Now()

	prompt, err := prompts.BuildAuditPrompt(request, c.config.SystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	payload := c.buildAPIPayload(prompt)

	response, err := c.sendRequest(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	auditResponse, err := c.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	auditResponse.Metadata = models.ProviderMetadata{
		Provider:   c.Name(),
		Model:      c.config.Model,
		Duration:   time.Since(startTime),
		TokensUsed: len(strings.Fields(response)),
		Cost:       c.EstimateCost(request),
	}

	return auditResponse, nil
}

func (c *Client) buildAPIPayload(prompt string) map[string]interface{} {
	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": c.config.SystemPrompt,
		},
		{
			"role":    "user",
			"content": prompt,
		},
	}

	payload := c.config.ToRequestConfig()
	payload["messages"] = messages
	return payload
}

func (c *Client) sendRequest(ctx context.Context, payload map[string]interface{}) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("request cancelled: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	if len(payloadBytes) > maxPayloadSize {
		return "", fmt.Errorf("payload too large: %d bytes", len(payloadBytes))
	}

	endpoint := c.config.GetEndpoint("v1/chat/completions")
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("User-Agent", "codegate/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	if c.config.EnableStreaming {
		return c.handleStreamingResponse(resp)
	}
	return c.handleStandardResponse(resp)
}

func (c *Client) handleStreamingResponse(resp *http.Response) (string, error) {
	scanner := bufio.NewScanner(resp.Body)
	var result strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || line == "data: [DONE]" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			line = line[6:]
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if chunk.Error != nil {
			return "", fmt.Errorf("streaming API error: %s", chunk.Error.Message)
		}

		if len(chunk.Choices) > 0 {
			result.WriteString(chunk.Choices[0].Delta.Content)
			if chunk.Choices[0].FinishReason != nil {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return result.String(), nil
}

func (c *Client) handleStandardResponse(resp *http.Response) (string, error) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResponse.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// parseResponse uses tiered parsing: direct JSON → repair → fallback
func (c *Client) parseResponse(response string) (*models.AuditResponse, error) {
	cleanResponse := strings.TrimSpace(response)

	// Preview mode: return raw markdown directly
	if viper.GetBool("preview") {
		return &models.AuditResponse{
			Summary:             cleanResponse,
			Issues:              []models.AuditIssue{},
			Suggestions:         []string{},
			ProposedCommitTitle: "chore: code review updates",
			ProposedCommitBody:  "Applied code review suggestions",
			Metadata: models.ProviderMetadata{
				Provider: c.Name(),
				Model:    c.config.Model,
			},
		}, nil
	}

	// Try direct JSON parse
	if gjson.Valid(cleanResponse) {
		if result, err := c.extractAuditResponse(cleanResponse); err == nil {
			return result, nil
		}
	}

	// Attempt simple repairs
	repairedResponse := c.simpleJSONRepair(cleanResponse)
	if gjson.Valid(repairedResponse) {
		if result, err := c.extractAuditResponse(repairedResponse); err == nil {
			return result, nil
		}
	}

	// Final fallback
	return c.createFallbackResponse(cleanResponse), nil
}

func (c *Client) simpleJSONRepair(jsonStr string) string {
	start := strings.Index(jsonStr, "{")
	if start == -1 {
		return jsonStr
	}
	end := strings.LastIndex(jsonStr, "}")
	if end == -1 || end <= start {
		return jsonStr
	}
	jsonStr = jsonStr[start : end+1]

	jsonStr = strings.ReplaceAll(jsonStr, "\n", "\\n")
	jsonStr = strings.ReplaceAll(jsonStr, "\r", "\\r")
	jsonStr = strings.ReplaceAll(jsonStr, "\t", "\\t")

	re := regexp.MustCompile(`,\s*([}\]])`)
	jsonStr = re.ReplaceAllString(jsonStr, "$1")

	openBraces := strings.Count(jsonStr, "{")
	closeBraces := strings.Count(jsonStr, "}")
	if openBraces > closeBraces {
		jsonStr += strings.Repeat("}", openBraces-closeBraces)
	}

	return jsonStr
}

func (c *Client) extractAuditResponse(jsonStr string) (*models.AuditResponse, error) {
	var auditResponse models.AuditResponse

	auditResponse.Summary = gjson.Get(jsonStr, "summary").String()
	auditResponse.ProposedCommitTitle = gjson.Get(jsonStr, "proposed_commit_title").String()
	auditResponse.ProposedCommitBody = gjson.Get(jsonStr, "proposed_commit_body").String()

	gjson.Get(jsonStr, "issues").ForEach(func(_, item gjson.Result) bool {
		issue := models.AuditIssue{
			Severity:       models.IssueSeverity(item.Get("severity").String()),
			Category:       models.AuditFocus(item.Get("category").String()),
			Title:          item.Get("title").String(),
			Description:    item.Get("description").String(),
			LineNumber:     int(item.Get("line_number").Int()),
			FileName:       item.Get("file_name").String(),
			CodeSnippet:    item.Get("code_snippet").String(),
			RecommendedFix: item.Get("recommended_fix").String(),
		}
		auditResponse.Issues = append(auditResponse.Issues, issue)
		return true
	})

	gjson.Get(jsonStr, "suggestions").ForEach(func(_, item gjson.Result) bool {
		auditResponse.Suggestions = append(auditResponse.Suggestions, item.String())
		return true
	})

	c.normalizeAuditResponse(&auditResponse)
	return &auditResponse, nil
}

func (c *Client) createFallbackResponse(response string) *models.AuditResponse {
	_ = templates.Preview // ensure embed is referenced
	return &models.AuditResponse{
		Summary: "Analysis completed with parsing issues - check full response for details",
		Issues: []models.AuditIssue{
			{
				Severity:    models.SeverityInfo,
				Category:    models.FocusBugs,
				Title:       "Response parsing incomplete",
				Description: "Some analysis results may be missing due to response format issues",
			},
		},
		Suggestions:         []string{"Consider reviewing the full AI response for detailed suggestions"},
		ProposedCommitTitle: "chore: code review and improvements",
		ProposedCommitBody:  "Applied code review suggestions based on automated analysis.",
	}
}

func (c *Client) normalizeAuditResponse(response *models.AuditResponse) {
	if strings.TrimSpace(response.Summary) == "" {
		response.Summary = "Code analysis completed"
	}
	if strings.TrimSpace(response.ProposedCommitTitle) == "" {
		response.ProposedCommitTitle = "chore: update configuration"
	}
	if strings.TrimSpace(response.ProposedCommitBody) == "" {
		response.ProposedCommitBody = "Updates application configuration with minor changes"
	}

	for i := range response.Issues {
		issue := &response.Issues[i]
		if !isValidSeverity(issue.Severity) {
			issue.Severity = models.SeverityMedium
		}
		if !isValidCategory(issue.Category) {
			issue.Category = models.FocusBugs
		}
		if strings.TrimSpace(issue.Title) == "" {
			issue.Title = "Code Review Finding"
		}
		if strings.TrimSpace(issue.Description) == "" {
			issue.Description = "No detailed description provided"
		}
		if issue.LineNumber < 0 {
			issue.LineNumber = 0
		}
	}

	if response.Suggestions == nil {
		response.Suggestions = []string{}
	}
}

func isValidSeverity(severity models.IssueSeverity) bool {
	switch severity {
	case models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow, models.SeverityInfo:
		return true
	default:
		return false
	}
}

func isValidCategory(category models.AuditFocus) bool {
	switch category {
	case models.FocusPerformance, models.FocusSecurity, models.FocusBugs, models.FocusMaintainability, models.FocusStyle, models.FocusDocumentation:
		return true
	default:
		return false
	}
}

// GetCapabilities returns what this provider supports
func (c *Client) GetCapabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsStreaming: c.config.EnableStreaming,
		MaxTokens:         c.config.MaxTokens,
		SupportedLanguages: []string{
			"javascript", "typescript", "python", "go", "java", "php", "c++", "c#", "rust", "ruby",
			"swift", "kotlin", "scala", "haskell", "elixir", "erlang",
		},
		SupportedFoci: []models.AuditFocus{
			models.FocusPerformance,
			models.FocusSecurity,
			models.FocusBugs,
			models.FocusMaintainability,
			models.FocusStyle,
			models.FocusDocumentation,
		},
		RequiresAuth: true,
		IsLocal:      false,
		HasCost:      true,
	}
}

// EstimateCost provides a rough cost estimate based on diff size
func (c *Client) EstimateCost(request models.AuditRequest) float64 {
	// Rough estimate: ~1 token per 4 chars for input, fixed output budget
	inputTokens := float64(len(request.Diff)+len(c.config.SystemPrompt)) / 4.0
	outputTokens := float64(c.config.MaxTokens) / 2.0 // assume ~half used
	return (inputTokens * costPerInputToken) + (outputTokens * costPerOutputToken)
}

// AuditRaw returns the raw string response from the API without parsing
func (c *Client) AuditRaw(ctx context.Context, request models.AuditRequest) (string, error) {
	prompt, err := prompts.BuildAuditPrompt(request, c.config.SystemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	payload := c.buildAPIPayload(prompt)
	return c.sendRequest(ctx, payload)
}

func (c *Client) Close() error {
	return nil
}
