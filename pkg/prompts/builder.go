package prompts

import (
	"strings"

	"codegate/pkg/models"
	"codegate/templates"

	"github.com/spf13/viper"
)

// BuildAuditPrompt constructs a specialized prompt for code analysis
// Returns formatted prompt based on preview mode and request parameters
func BuildAuditPrompt(request models.AuditRequest, systemPrompt string) (string, error) {
	isPreview := viper.GetBool("preview")

	var builder strings.Builder
	builder.Grow(len(systemPrompt) + len(request.Diff) + 512)

	// Build system instructions
	if err := writeSystemInstructions(&builder, systemPrompt, isPreview); err != nil {
		return "", err
	}

	// Add contextual metadata
	writeContextualMetadata(&builder, request)

	// Add git diff
	writeDiff(&builder, request.Diff)

	// Add final instructions
	writeFinalInstructions(&builder, isPreview)

	return builder.String(), nil
}

// writeSystemInstructions writes the system prompt or preview template
func writeSystemInstructions(builder *strings.Builder, systemPrompt string, isPreview bool) error {
	if isPreview {
		builder.WriteString("You are an AI-powered code reviewer with expertise across all programming languages. Provide thorough, actionable code analysis.\n\n")
		builder.WriteString("Analyze the provided git diff and create a comprehensive markdown report following this EXACT structure:\n\n")
		builder.WriteString(templates.Preview)
	} else {
		builder.WriteString(systemPrompt)
	}
	return nil
}

// writeContextualMetadata adds focus areas and language context
func writeContextualMetadata(builder *strings.Builder, request models.AuditRequest) {
	// Add focus areas if specified
	if len(request.Focus) > 0 {
		builder.WriteString("\n\nFocus specifically on these areas:\n")
		for _, focus := range request.Focus {
			builder.WriteString("- ")
			builder.WriteString(string(focus))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Add language context if detected
	if request.Language != "" {
		builder.WriteString("The code is primarily ")
		builder.WriteString(request.Language)
		builder.WriteString(".\n\n")
	}
}

// writeDiff adds the git diff section
func writeDiff(builder *strings.Builder, diff string) {
	builder.WriteString("Here is the git diff to analyze:\n\n```diff\n")
	builder.WriteString(diff)
	builder.WriteString("\n```\n\n")
}

// writeFinalInstructions adds mode-specific final instructions
func writeFinalInstructions(builder *strings.Builder, isPreview bool) {
	if !isPreview {
		builder.WriteString("Respond ONLY with valid JSON in the specified format.")
	}
}
