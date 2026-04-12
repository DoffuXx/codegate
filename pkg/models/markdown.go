package models

import (
	"fmt"
	"strings"
	"time"
)

// MarkdownConfig allows customization of markdown output
type MarkdownConfig struct {
	IncludeMetadata   bool `json:"include_metadata"`
	IncludeTOC        bool `json:"include_toc"`
	MaxCodeSnippetLen int  `json:"max_code_snippet_len"`
	GroupBySeverity   bool `json:"group_by_severity"`
	ShowLineNumbers   bool `json:"show_line_numbers"`
}

// DefaultMarkdownConfig returns sensible defaults
func DefaultMarkdownConfig() MarkdownConfig {
	return MarkdownConfig{
		IncludeMetadata:   true,
		IncludeTOC:        false,
		MaxCodeSnippetLen: 500,
		GroupBySeverity:   true,
		ShowLineNumbers:   true,
	}
}

// ToMarkdown generates markdown with default configuration
func (r *AuditResponse) ToMarkdown() string {
	return r.ToMarkdownWithConfig(DefaultMarkdownConfig())
}

// ToMarkdownWithConfig generates customized markdown output
func (r *AuditResponse) ToMarkdownWithConfig(config MarkdownConfig) string {
	builder := &markdownBuilder{
		config: config,
		sb:     &strings.Builder{},
	}

	return builder.build(r)
}

// markdownBuilder encapsulates markdown generation logic
type markdownBuilder struct {
	config MarkdownConfig
	sb     *strings.Builder
}

// build generates the complete markdown document
func (mb *markdownBuilder) build(r *AuditResponse) string {
	mb.writeHeader(r)
	mb.writeSummary(r)
	mb.writeIssues(r)
	mb.writeSuggestions(r)
	mb.writeCommitInfo(r)

	return mb.sb.String()
}

// writeHeader writes document header with metadata and TOC
func (mb *markdownBuilder) writeHeader(r *AuditResponse) {
	if mb.config.IncludeMetadata && r.Metadata.Provider != "" {
		mb.writeMetadataTable(r.Metadata)
	}

	if mb.config.IncludeTOC {
		mb.writeTOC(r)
	}
}

// writeMetadataTable writes analysis metadata in table format
func (mb *markdownBuilder) writeMetadataTable(metadata ProviderMetadata) {
	mb.sb.WriteString("## Analysis Details\n\n")
	mb.sb.WriteString("| Field | Value |\n")
	mb.sb.WriteString("|-------|-------|\n")
	mb.sb.WriteString(fmt.Sprintf("| **Provider** | %s |\n", metadata.Provider))
	mb.sb.WriteString(fmt.Sprintf("| **Model** | %s |\n", metadata.Model))
	mb.sb.WriteString(fmt.Sprintf("| **Duration** | %v |\n", metadata.Duration.Round(time.Millisecond)))

	if metadata.TokensUsed > 0 {
		mb.sb.WriteString(fmt.Sprintf("| **Tokens Used** | %d |\n", metadata.TokensUsed))
	}

	if metadata.Cost > 0 {
		mb.sb.WriteString(fmt.Sprintf("| **Cost** | $%.4f |\n", metadata.Cost))
	}

	mb.sb.WriteString("\n")
}

// writeTOC generates table of contents
func (mb *markdownBuilder) writeTOC(r *AuditResponse) {
	mb.sb.WriteString("## Table of Contents\n\n")
	mb.sb.WriteString("- [Summary](#summary)\n")

	if len(r.Issues) > 0 {
		mb.sb.WriteString("- [Issues Found](#issues-found)\n")
		if mb.config.GroupBySeverity {
			severityGroups := groupIssuesBySeverity(r.Issues)
			for _, severity := range getSeverityOrder() {
				if issues, exists := severityGroups[severity]; exists && len(issues) > 0 {
					anchor := strings.ToLower(strings.ReplaceAll(string(severity), " ", "-"))
					mb.sb.WriteString(fmt.Sprintf("  - [%s Issues (%d)](#%s)\n",
						strings.Title(string(severity)), len(issues), anchor))
				}
			}
		}
	}

	if len(r.Suggestions) > 0 {
		mb.sb.WriteString("- [General Suggestions](#general-suggestions)\n")
	}

	if r.ProposedCommitTitle != "" {
		mb.sb.WriteString("- [Proposed Commit Message](#proposed-commit-message)\n")
	}

	mb.sb.WriteString("\n")
}

// writeSummary writes summary section with issue statistics
func (mb *markdownBuilder) writeSummary(r *AuditResponse) {
	mb.sb.WriteString("## Summary\n\n")

	if len(r.Issues) > 0 {
		stats := getIssueStatistics(r.Issues)
		mb.sb.WriteString(fmt.Sprintf("**Issues Found:** %d total\n\n", len(r.Issues)))
		mb.writeIssueStatistics(stats)
	}

	if r.Summary != "" {
		mb.sb.WriteString(r.Summary + "\n\n")
	} else {
		mb.sb.WriteString("No detailed summary provided.\n\n")
	}
}

// writeIssueStatistics writes severity breakdown table
func (mb *markdownBuilder) writeIssueStatistics(stats map[IssueSeverity]int) {
	mb.sb.WriteString("| Severity | Count |\n")
	mb.sb.WriteString("|----------|-------|\n")

	for _, severity := range getSeverityOrder() {
		if count, exists := stats[severity]; exists && count > 0 {
			mb.sb.WriteString(fmt.Sprintf("| %s | %d |\n",
				strings.Title(string(severity)), count))
		}
	}

	mb.sb.WriteString("\n")
}

// writeIssues writes all issues section
func (mb *markdownBuilder) writeIssues(r *AuditResponse) {
	if len(r.Issues) == 0 {
		mb.sb.WriteString("## No Issues Found\n\n")
		mb.sb.WriteString("Your code follows best practices and no issues were detected.\n\n")
		return
	}

	mb.sb.WriteString("## Issues Found\n\n")

	if mb.config.GroupBySeverity {
		mb.writeIssuesGroupedBySeverity(r.Issues)
	} else {
		mb.writeIssuesSequentially(r.Issues)
	}
}

// writeIssuesGroupedBySeverity writes issues organized by severity
func (mb *markdownBuilder) writeIssuesGroupedBySeverity(issues []AuditIssue) {
	severityGroups := groupIssuesBySeverity(issues)

	for _, severity := range getSeverityOrder() {
		severityIssues, exists := severityGroups[severity]
		if !exists || len(severityIssues) == 0 {
			continue
		}

		mb.sb.WriteString(fmt.Sprintf("### %s Issues (%d)\n\n",
			strings.Title(string(severity)), len(severityIssues)))

		for i, issue := range severityIssues {
			mb.writeIssueDetails(issue, i+1)
		}
	}
}

// writeIssuesSequentially writes issues in original order
func (mb *markdownBuilder) writeIssuesSequentially(issues []AuditIssue) {
	for i, issue := range issues {
		mb.writeIssueDetails(issue, i+1)
	}
}

// writeIssueDetails writes individual issue details
func (mb *markdownBuilder) writeIssueDetails(issue AuditIssue, index int) {
	severityBadge := getSeverityBadge(issue.Severity)
	mb.sb.WriteString(fmt.Sprintf("#### %d. %s %s\n\n", index, issue.Title, severityBadge))

	// Quick info table
	mb.sb.WriteString("| | |\n")
	mb.sb.WriteString("|---|---|\n")
	mb.sb.WriteString(fmt.Sprintf("| **Severity** | %s |\n", strings.ToUpper(string(issue.Severity))))
	mb.sb.WriteString(fmt.Sprintf("| **Category** | %s |\n", strings.Title(string(issue.Category))))

	if location := buildLocationString(issue.FileName, issue.LineNumber); location != "Unknown" {
		mb.sb.WriteString(fmt.Sprintf("| **Location** | `%s` |\n", location))
	}

	mb.sb.WriteString("\n")

	if issue.Description != "" {
		mb.sb.WriteString("**Description:**\n\n")
		mb.sb.WriteString(issue.Description + "\n\n")
	}

	mb.writeCodeSnippet("Problematic Code", issue.CodeSnippet, issue.FileName)
	mb.writeCodeSnippet("Recommended Fix", issue.RecommendedFix, issue.FileName)

	mb.sb.WriteString("---\n\n")
}

// writeSuggestions writes general suggestions section
func (mb *markdownBuilder) writeSuggestions(r *AuditResponse) {
	if len(r.Suggestions) == 0 {
		return
	}

	mb.sb.WriteString("## General Suggestions\n\n")

	for i, suggestion := range r.Suggestions {
		// Clean and format each suggestion
		cleanSuggestion := mb.formatSuggestion(suggestion)
		mb.sb.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, cleanSuggestion))
	}
}

func (mb *markdownBuilder) writeCodeSnippet(title, code, fileName string) {
	if code == "" {
		return
	}

	// Handle truncation properly to maintain markdown structure
	wasTruncated := false

	if mb.config.MaxCodeSnippetLen > 0 && len(code) > mb.config.MaxCodeSnippetLen {
		code = code[:mb.config.MaxCodeSnippetLen]
		wasTruncated = true

		// Ensure we don't break in the middle of a line
		if lastNewline := strings.LastIndex(code, "\n"); lastNewline > mb.config.MaxCodeSnippetLen/2 {
			code = code[:lastNewline]
		}
	}

	lang := detectLanguageFromFilename(fileName)
	mb.sb.WriteString(fmt.Sprintf("**%s:**\n\n", title))
	mb.sb.WriteString(fmt.Sprintf("```%s\n%s", lang, code))

	// Add newline if code doesn't end with one
	if !strings.HasSuffix(code, "\n") {
		mb.sb.WriteString("\n")
	}

	// Add truncation notice before closing if truncated
	if wasTruncated {
		mb.sb.WriteString("... (truncated)\n")
	}

	mb.sb.WriteString("```\n\n")
}

// writeCommitInfo writes proposed commit message section
func (mb *markdownBuilder) writeCommitInfo(r *AuditResponse) {
	if r.ProposedCommitTitle == "" {
		return
	}

	mb.sb.WriteString("## Proposed Commit Message\n\n")

	// Write title
	mb.sb.WriteString("**Title:**\n")
	mb.sb.WriteString("```\n")
	mb.sb.WriteString(r.ProposedCommitTitle)
	mb.sb.WriteString("\n```\n\n")

	// Write body if present
	if r.ProposedCommitBody != "" {
		mb.sb.WriteString("**Body:**\n")
		mb.sb.WriteString("```\n")
		mb.sb.WriteString(r.ProposedCommitBody)
		mb.sb.WriteString("\n```\n\n")
	}
}

func (mb *markdownBuilder) formatSuggestion(suggestion string) string {
	// Remove leading/trailing whitespace
	suggestion = strings.TrimSpace(suggestion)

	// Split into lines for processing
	lines := strings.Split(suggestion, "\n")
	var formattedLines []string

	inCodeBlock := false
	codeBlockLanguage := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines at start/end but preserve them in middle
		if line == "" {
			if len(formattedLines) > 0 {
				formattedLines = append(formattedLines, "")
			}
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Starting code block
				inCodeBlock = true
				codeBlockLanguage = strings.TrimPrefix(line, "```")
				formattedLines = append(formattedLines, "")
				formattedLines = append(formattedLines, fmt.Sprintf("   ```%s", codeBlockLanguage))
			} else {
				// Ending code block
				inCodeBlock = false
				formattedLines = append(formattedLines, "   ```")
				formattedLines = append(formattedLines, "")
			}
			continue
		}

		if inCodeBlock {
			// Indent code lines
			formattedLines = append(formattedLines, "   "+line)
			continue
		}

		// Handle bullet points
		if strings.HasPrefix(line, "- ") {
			// Convert to sub-bullet (indent)
			formattedLines = append(formattedLines, "   "+line)
			continue
		}

		// Handle regular text - ensure proper spacing
		if len(formattedLines) > 0 && !strings.HasSuffix(formattedLines[len(formattedLines)-1], ":") {
			// Add space before new paragraphs unless previous line ends with colon
			lastLine := formattedLines[len(formattedLines)-1]
			if lastLine != "" && !strings.HasSuffix(lastLine, ":") {
				formattedLines = append(formattedLines, "")
			}
		}

		formattedLines = append(formattedLines, line)
	}

	// Join and clean up extra whitespace
	result := strings.Join(formattedLines, "\n")

	// Remove multiple consecutive empty lines
	result = mb.cleanupWhitespace(result)

	return result
}

// cleanupWhitespace removes excessive whitespace while preserving intentional formatting
func (mb *markdownBuilder) cleanupWhitespace(text string) string {
	// Replace multiple consecutive empty lines with single empty line
	lines := strings.Split(text, "\n")
	var cleaned []string

	emptyLineCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
			if emptyLineCount <= 1 {
				cleaned = append(cleaned, "")
			}
		} else {
			emptyLineCount = 0
			cleaned = append(cleaned, line)
		}
	}

	// Remove trailing empty lines
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	return strings.Join(cleaned, "\n")
}

// Helper functions

// getSeverityBadge returns text badge for severity level
func getSeverityBadge(severity IssueSeverity) string {
	badges := map[IssueSeverity]string{
		SeverityCritical: "`CRITICAL`",
		SeverityHigh:     "`HIGH`",
		SeverityMedium:   "`MEDIUM`",
		SeverityLow:      "`LOW`",
		SeverityInfo:     "`INFO`",
	}

	if badge, exists := badges[severity]; exists {
		return badge
	}

	return "`UNKNOWN`"
}

// getIssueStatistics counts issues by severity
func getIssueStatistics(issues []AuditIssue) map[IssueSeverity]int {
	stats := make(map[IssueSeverity]int)

	for _, issue := range issues {
		stats[issue.Severity]++
	}

	return stats
}

// groupIssuesBySeverity organizes issues by severity level
func groupIssuesBySeverity(issues []AuditIssue) map[IssueSeverity][]AuditIssue {
	groups := make(map[IssueSeverity][]AuditIssue)

	for _, issue := range issues {
		groups[issue.Severity] = append(groups[issue.Severity], issue)
	}

	return groups
}

// getSeverityOrder returns severity levels in priority order
func getSeverityOrder() []IssueSeverity {
	return []IssueSeverity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityInfo,
	}
}

// buildLocationString formats file location string
func buildLocationString(fileName string, lineNumber int) string {
	switch {
	case fileName == "" && lineNumber == 0:
		return "Unknown"
	case fileName == "":
		return fmt.Sprintf("Line %d", lineNumber)
	case lineNumber == 0:
		return fileName
	default:
		return fmt.Sprintf("%s:%d", fileName, lineNumber)
	}
}

// detectLanguageFromFilename returns language identifier for syntax highlighting
func detectLanguageFromFilename(fileName string) string {
	if fileName == "" {
		return ""
	}

	lastDot := strings.LastIndex(fileName, ".")
	if lastDot == -1 {
		return ""
	}

	ext := strings.ToLower(fileName[lastDot+1:])

	langMap := map[string]string{
		"go":         "go",
		"js":         "javascript",
		"jsx":        "jsx",
		"ts":         "typescript",
		"tsx":        "tsx",
		"php":        "php",
		"py":         "python",
		"java":       "java",
		"kt":         "kotlin",
		"cpp":        "cpp",
		"cc":         "cpp",
		"cxx":        "cpp",
		"c":          "c",
		"cs":         "csharp",
		"rb":         "ruby",
		"rs":         "rust",
		"sql":        "sql",
		"json":       "json",
		"xml":        "xml",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"sass":       "sass",
		"yaml":       "yaml",
		"yml":        "yaml",
		"toml":       "toml",
		"sh":         "bash",
		"bash":       "bash",
		"ps1":        "powershell",
		"dockerfile": "dockerfile",
		"md":         "markdown",
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return ""
}

// Maintain backward compatibility
func (r *AuditResponse) groupIssuesBySeverity() map[IssueSeverity][]AuditIssue {
	return groupIssuesBySeverity(r.Issues)
}
