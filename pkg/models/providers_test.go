package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuditIssueUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedFix    string
		shouldError    bool
	}{
		{
			name: "String recommended_fix",
			jsonInput: `{
				"severity": "high",
				"category": "bugs",
				"title": "Test Issue",
				"description": "Test description",
				"recommended_fix": "Fix the bug"
			}`,
			expectedFix: "Fix the bug",
			shouldError: false,
		},
		{
			name: "Array of strings recommended_fix",
			jsonInput: `{
				"severity": "medium",
				"category": "security",
				"title": "Test Issue",
				"description": "Test description",
				"recommended_fix": ["Line 1", "Line 2", "Line 3"]
			}`,
			expectedFix: "Line 1\nLine 2\nLine 3",
			shouldError: false,
		},
		{
			name: "Empty recommended_fix",
			jsonInput: `{
				"severity": "low",
				"category": "style",
				"title": "Test Issue",
				"description": "Test description",
				"recommended_fix": ""
			}`,
			expectedFix: "",
			shouldError: false,
		},
		{
			name: "Null recommended_fix",
			jsonInput: `{
				"severity": "info",
				"category": "documentation",
				"title": "Test Issue",
				"description": "Test description",
				"recommended_fix": null
			}`,
			expectedFix: "",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var issue AuditIssue
			err := json.Unmarshal([]byte(tt.jsonInput), &issue)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if issue.RecommendedFix != tt.expectedFix {
				t.Errorf("RecommendedFix = %q, want %q", issue.RecommendedFix, tt.expectedFix)
			}
		})
	}
}

func TestAuditResponseToMarkdown(t *testing.T) {
	response := &AuditResponse{
		Summary: "Test summary",
		Issues: []AuditIssue{
			{
				Severity:       SeverityHigh,
				Category:       FocusSecurity,
				Title:          "SQL Injection",
				Description:    "Potential SQL injection vulnerability",
				FileName:       "database.go",
				LineNumber:     42,
				CodeSnippet:    "query := \"SELECT * FROM users WHERE id = \" + userId",
				RecommendedFix: "Use prepared statements",
			},
		},
		Suggestions:         []string{"Add input validation", "Use ORM"},
		ProposedCommitTitle: "fix: address security vulnerabilities",
		ProposedCommitBody:  "Fixed SQL injection issue",
	}

	markdown := response.ToMarkdown()

	// Test that markdown contains expected sections
	expectedSections := []string{
		"Test summary",
		"SQL Injection",
		"database.go:42",
		"Add input validation",
		"fix: address security vulnerabilities",
	}

	for _, section := range expectedSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Markdown missing expected section: %q", section)
		}
	}
}

func TestGetRandomQuote(t *testing.T) {
	response := &AuditResponse{}

	// Test that it returns a quote
	quote := response.GetRandomQuote()
	if quote == "" {
		t.Error("GetRandomQuote() returned empty string")
	}

	// Test that multiple calls can return different quotes
	// (this is probabilistic, but with 10 quotes, we should get variety)
	quotes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		quotes[response.GetRandomQuote()] = true
	}

	if len(quotes) < 2 {
		t.Error("GetRandomQuote() appears to not be random")
	}
}

func TestIssueSeverityConstants(t *testing.T) {
	// Test that severity constants are defined correctly
	severities := []IssueSeverity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityInfo,
	}

	for _, sev := range severities {
		if sev == "" {
			t.Errorf("Severity constant is empty")
		}
	}
}

func TestAuditFocusConstants(t *testing.T) {
	// Test that focus constants are defined correctly
	focuses := []AuditFocus{
		FocusPerformance,
		FocusSecurity,
		FocusBugs,
		FocusMaintainability,
		FocusStyle,
		FocusDocumentation,
	}

	for _, focus := range focuses {
		if focus == "" {
			t.Errorf("Focus constant is empty")
		}
	}
}
