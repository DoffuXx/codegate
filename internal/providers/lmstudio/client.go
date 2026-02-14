package lmstudio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"codegate/pkg/interfaces"
	"codegate/pkg/models"
	"codegate/pkg/registry"
	"codegate/templates"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

const (
	maxPayloadSize = 10 * 1024 * 1024 // 10MB
)

// getDebugLogPath returns the path for debug logging
func getDebugLogPath() string {
	return filepath.Join(os.TempDir(), "codegate-debug.log")
}

// Client implements the AIProvider interface for LM Studio
type Client struct {
	config     Config
	httpClient *http.Client
}

func init() {
	registry.RegisterProvider("lmstudio", NewProvider)
}

func NewProvider(config interfaces.ProviderConfig) (interfaces.AIProvider, error) {
	lmConfig := DefaultConfig()

	if config.Config != nil {
		configBytes, err := json.Marshal(config.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := json.Unmarshal(configBytes, &lmConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal LM Studio config: %w", err)
		}
	}

	httpClient := &http.Client{
		Timeout: time.Duration(lmConfig.TimeoutSeconds) * time.Second,
	}

	return &Client{
		config:     lmConfig,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Name() string {
	return "LM Studio"
}

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to LM Studio at %s: %w", c.config.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LM Studio server returned status %d", resp.StatusCode)
	}

	if c.config.Model == "" {
		if err := c.autoDetectModel(ctx); err != nil {
			return fmt.Errorf("failed to auto-detect model: %w", err)
		}
	}

	return nil
}

func (c *Client) autoDetectModel(ctx context.Context) error {
	endpoint := c.config.GetEndpoint("v1/models")
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return err
	}

	if len(modelsResp.Data) == 0 {
		return fmt.Errorf("no models available in LM Studio")
	}

	c.config.Model = modelsResp.Data[0].ID
	return nil
}

func (c *Client) Audit(ctx context.Context, request models.AuditRequest) (*models.AuditResponse, error) {
	startTime := time.Now()

	prompt, err := c.buildPrompt(request)
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
		Cost:       0,
	}

	return auditResponse, nil
}

// buildPrompt constructs a specialized prompt for code analysis
func (c *Client) buildPrompt(request models.AuditRequest) (string, error) {
	var promptBuilder strings.Builder

	isPreview := viper.GetBool("preview")

	if isPreview {
		template, err := c.loadPreviewTemplate()
		if err != nil {
			return "", fmt.Errorf("failed to load preview template: %w", err)
		}
		promptBuilder.WriteString("You are an AI-powered code reviewer with expertise across all programming languages. Provide thorough, actionable code analysis.\n\n")
		promptBuilder.WriteString("Analyze the provided git diff and create a comprehensive markdown report following this EXACT structure:\n\n")
		promptBuilder.WriteString(template)
	} else {
		promptBuilder.WriteString(c.config.SystemPrompt)
	}

	// Add focus areas if specified
	if len(request.Focus) > 0 {
		promptBuilder.WriteString("\n\nFocus specifically on these areas:\n")
		for _, focus := range request.Focus {
			promptBuilder.WriteString(fmt.Sprintf("- %s\n", focus))
		}
		promptBuilder.WriteString("\n")
	}

	// Add language context if detected
	if request.Language != "" {
		promptBuilder.WriteString(fmt.Sprintf("The code is primarily %s.\n\n", request.Language))
	}

	// Add the git diff
	promptBuilder.WriteString("Here is the git diff to analyze:\n\n")
	promptBuilder.WriteString("```diff\n")
	promptBuilder.WriteString(request.Diff)
	promptBuilder.WriteString("\n```\n\n")

	// Final instructions for JSON mode
	if !isPreview {
		promptBuilder.WriteString(`Respond ONLY with valid JSON in the specified format.`)
	}

	return promptBuilder.String(), nil
}

// loadPreviewTemplate loads the markdown template for preview mode
func (c *Client) loadPreviewTemplate() (string, error) {
	return templates.Preview, nil
}

func (c *Client) buildAPIPayload(prompt string) map[string]interface{} {
	messages := []map[string]interface{}{
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

// Simplified parseResponse - try direct JSON first, then simple fallback
func (c *Client) parseResponse(response string) (*models.AuditResponse, error) {
	cleanResponse := strings.TrimSpace(response)

	// Check if we're in preview mode
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

	// Log for debugging if verbose
	if viper.GetBool("verbose") {
		c.logParsingAttempt("ORIGINAL", cleanResponse)
	}

	// Try parsing as-is first
	if gjson.Valid(cleanResponse) {
		if result, err := c.extractAuditResponse(cleanResponse); err == nil {
			return result, nil
		}
	}

	// Simple repair attempt
	repairedResponse := c.simpleJSONRepair(cleanResponse)

	if viper.GetBool("verbose") {
		c.logParsingAttempt("REPAIRED", repairedResponse)
	}

	if gjson.Valid(repairedResponse) {
		if result, err := c.extractAuditResponse(repairedResponse); err == nil {
			return result, nil
		}
	}

	// Fallback to pattern extraction
	return c.createFallbackResponse(cleanResponse), nil
}

// Simplified JSON repair - only basic fixes
func (c *Client) simpleJSONRepair(jsonStr string) string {
	// Extract JSON boundaries
	start := strings.Index(jsonStr, "{")
	if start == -1 {
		return jsonStr
	}
	end := strings.LastIndex(jsonStr, "}")
	if end == -1 || end <= start {
		return jsonStr
	}
	jsonStr = jsonStr[start : end+1]

	// Basic fixes
	jsonStr = strings.ReplaceAll(jsonStr, "\n", "\\n")
	jsonStr = strings.ReplaceAll(jsonStr, "\r", "\\r")
	jsonStr = strings.ReplaceAll(jsonStr, "\t", "\\t")

	// Remove trailing commas
	re := regexp.MustCompile(`,\s*([}\]])`)
	jsonStr = re.ReplaceAllString(jsonStr, "$1")

	// Balance braces
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

	// Parse issues array
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

	// Parse suggestions array
	gjson.Get(jsonStr, "suggestions").ForEach(func(_, item gjson.Result) bool {
		auditResponse.Suggestions = append(auditResponse.Suggestions, item.String())
		return true
	})

	c.normalizeAuditResponse(&auditResponse)
	return &auditResponse, nil
}

func (c *Client) createFallbackResponse(response string) *models.AuditResponse {
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

func (c *Client) logParsingAttempt(stage, jsonStr string) {
	logFile := getDebugLogPath()
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	logger.Printf("=== %s ===", stage)
	logger.Printf("Valid JSON: %v", gjson.Valid(jsonStr))
	logger.Printf("Length: %d chars", len(jsonStr))

	if len(jsonStr) > 400 {
		logger.Printf("First 200 chars: %q", jsonStr[:200])
		logger.Printf("Last 200 chars: %q", jsonStr[len(jsonStr)-200:])
	} else {
		logger.Printf("Full content: %q", jsonStr)
	}
	logger.Println("=====================================")
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

func (c *Client) GetCapabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsStreaming: c.config.EnableStreaming,
		MaxTokens:         c.config.MaxTokens,
		SupportedLanguages: []string{
			"javascript", "typescript", "python", "go", "java", "php", "c++", "c#", "rust", "ruby",
		},
		SupportedFoci: []models.AuditFocus{
			models.FocusPerformance,
			models.FocusSecurity,
			models.FocusBugs,
			models.FocusMaintainability,
			models.FocusStyle,
			models.FocusDocumentation,
		},
		RequiresAuth: false,
		IsLocal:      true,
		HasCost:      false,
	}
}

func (c *Client) AuditRaw(ctx context.Context, request models.AuditRequest) (string, error) {
	prompt, err := c.buildPrompt(request)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	payload := c.buildAPIPayload(prompt)
	return c.sendRequest(ctx, payload)
}

func (c *Client) EstimateCost(request models.AuditRequest) float64 {
	return 0.0 // Local inference is free
}

func (c *Client) Close() error {
	return nil
}
