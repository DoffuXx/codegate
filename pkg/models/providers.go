package models

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"
)

// AuditRequest represents a request to analyze code changes
type AuditRequest struct {
	Diff        string       `json:"diff"`
	Focus       []AuditFocus `json:"focus,omitempty"`
	Language    string       `json:"language,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
}

// AuditFocus represents different types of analysis the AI should perform
type AuditFocus string

const (
	FocusPerformance     AuditFocus = "performance"
	FocusSecurity        AuditFocus = "security"
	FocusBugs            AuditFocus = "bugs"
	FocusMaintainability AuditFocus = "maintainability"
	FocusStyle           AuditFocus = "style"
	FocusDocumentation   AuditFocus = "documentation"
)

// AuditResponse represents the structured response from an AI provider
type AuditResponse struct {
	Summary             string           `json:"summary"`
	Issues              []AuditIssue     `json:"issues"`
	Suggestions         []string         `json:"suggestions"`
	ProposedCommitTitle string           `json:"proposed_commit_title"`
	ProposedCommitBody  string           `json:"proposed_commit_body"`
	Metadata            ProviderMetadata `json:"metadata"`
}

// AuditIssue represents a specific finding from the code analysis
type AuditIssue struct {
	Severity       IssueSeverity `json:"severity"`
	Category       AuditFocus    `json:"category"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	LineNumber     int           `json:"line_number,omitempty"`
	FileName       string        `json:"file_name,omitempty"`
	CodeSnippet    string        `json:"code_snippet,omitempty"`
	RecommendedFix string        `json:"recommended_fix"`
}

// IssueSeverity represents the importance level of an audit finding
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityHigh     IssueSeverity = "high"
	SeverityMedium   IssueSeverity = "medium"
	SeverityLow      IssueSeverity = "low"
	SeverityInfo     IssueSeverity = "info"
)

// ProviderMetadata contains provider-specific information about the analysis
type ProviderMetadata struct {
	Provider   string        `json:"provider"`
	Model      string        `json:"model"`
	TokensUsed int           `json:"tokens_used,omitempty"`
	Duration   time.Duration `json:"duration"`
	Cost       float64       `json:"cost,omitempty"`
}

var codeReviewQuotes = []string{
	"Code is poetry written for machines to dance to.",
	"The best code is the code that doesn't need to be written.",
	"Debugging is like being the detective in a crime movie where you are also the murderer.",
	"Clean code always looks like it was written by someone who cares.",
	"Code never lies, comments sometimes do.",
	"Simplicity is the ultimate sophistication in code.",
	"First, solve the problem. Then, write the code.",
	"The best error message is the one that never shows up.",
	"Good code is its own best documentation.",
	"Programming is the art of telling another human what one wants the computer to do.",
}

func (issue *AuditIssue) UnmarshalJSON(data []byte) error {
	type Alias AuditIssue
	aux := &struct {
		RecommendedFix interface{} `json:"recommended_fix"`
		*Alias
	}{
		Alias: (*Alias)(issue),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert recommended_fix to string regardless of AI output type
	switch v := aux.RecommendedFix.(type) {
	case string:
		issue.RecommendedFix = v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				parts = append(parts, str)
			}
		}
		issue.RecommendedFix = strings.Join(parts, "\n")
	case []string:
		issue.RecommendedFix = strings.Join(v, "\n")
	default:
		issue.RecommendedFix = ""
	}

	return nil
}

func (r *AuditResponse) GetRandomQuote() string {
	if len(codeReviewQuotes) == 0 {
		return "Happy coding!"
	}

	// Seed with current time for randomness
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(codeReviewQuotes))
	return codeReviewQuotes[index]
}
