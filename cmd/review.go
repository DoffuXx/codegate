package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"codegate/internal/config"
	"codegate/internal/git"
	"codegate/internal/providers"
	"codegate/pkg/models"

	"github.com/fatih/color"
	"github.com/mitchellh/go-wordwrap"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	maxAnalysisTime    = 5 * time.Minute
	maxDisplayWidth    = 80
	progressUpdateRate = 100 * time.Millisecond
	quoteUpdateRate    = 3 * time.Second
)

// getDebugLogPath returns the path for debug logging
func getDebugLogPath() string {
	return filepath.Join(os.TempDir(), "codegate-debug.log")
}

// reviewCmd represents the review command that analyzes staged changes
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Analyze staged git changes for issues",
	Long: `Review analyzes your staged git changes using AI to identify potential
issues including bugs, security vulnerabilities, and performance problems.

The command extracts the current staged changes using 'git diff --cached'
and sends them to your configured AI provider for analysis.

By default, generates a markdown report and opens it in your browser/viewer.
Use --no-preview for structured table output instead.

Examples:
  git-audit review                      # Generate markdown preview (default)
  git-audit review --no-preview         # Show table output in terminal
  git-audit review --focus security     # Focus only on security issues
  git-audit review --format json        # Output results as JSON`,

	RunE: runReview,
}

// Command-specific flags for the review command
var (
	focusFlag     []string // Specific focus areas for this review
	formatFlag    string   // Output format override
	providerFlag  string   // Provider override for this review
	noPreviewFlag bool     // Disable preview mode (opt-out)
)

// init sets up the review command flags and adds it to the root command
func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().StringSliceVarP(&focusFlag, "focus", "f", nil,
		"focus areas: performance, security, bugs, maintainability, style, documentation")

	reviewCmd.Flags().StringVar(&formatFlag, "format", "",
		"output format: table, json, markdown (overrides config and disables preview)")

	reviewCmd.Flags().StringVarP(&providerFlag, "provider", "p", "",
		"AI provider to use: lmstudio, openai, claude (overrides config)")

	reviewCmd.Flags().BoolVar(&noPreviewFlag, "no-preview", false,
		"disable markdown preview, show table output instead")
}

// runReview implements the main logic for the review command
func runReview(cmd *cobra.Command, args []string) error {
	// Preview is enabled by default, disabled only if --no-preview or --format is set
	previewEnabled := !noPreviewFlag && formatFlag == ""
	viper.Set("preview", previewEnabled)

	ctx, cancel := context.WithTimeout(context.Background(), maxAnalysisTime)
	defer cancel()

	configMgr := config.NewManager()

	provider, err := getProviderForReview(configMgr)
	if err != nil {
		return fmt.Errorf("failed to initialize AI provider: %w", err)
	}
	defer provider.Close()

	gitInfo, err := git.ExtractStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to extract git diff: %w", err)
	}

	if strings.TrimSpace(gitInfo.Diff) == "" {
		fmt.Println("No staged changes found. Use 'git add' to stage files for review.")
		return nil
	}

	request, err := buildAuditRequest(gitInfo, configMgr)
	if err != nil {
		return fmt.Errorf("failed to build audit request: %w", err)
	}

	if viper.GetBool("verbose") {
		debugLogFile := getDebugLogPath()
		os.Remove(debugLogFile)
		fmt.Printf("JSON debug logging enabled: %s\n", debugLogFile)
	}

	var response *models.AuditResponse
	if err := showProgressWithEstimate("Analyzing staged changes", func() {
		response, err = provider.Audit(ctx, *request)
	}); err != nil {
		return fmt.Errorf("progress tracking failed: %w", err)
	}

	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Handle preview mode (default behavior)
	if previewEnabled {
		return handlePreviewMode(response)
	}

	// Handle explicit format flags or --no-preview
	return displayResults(response, configMgr)
}

// handlePreviewMode processes preview output and opens in browser
func handlePreviewMode(response *models.AuditResponse) error {
	var markdown string

	// Check if response contains raw markdown (when AI returns markdown directly)
	if strings.Contains(response.Summary, "# 🔍 Code Review Analysis") {
		markdown = response.Summary
	} else {
		// Convert structured response to markdown
		markdown = response.ToMarkdown()
	}

	tmpFile := filepath.Join(os.TempDir(), "codegate-"+time.Now().Format("20060102-150405")+".md")
	if err := os.WriteFile(tmpFile, []byte(markdown), 0o644); err != nil {
		return fmt.Errorf("failed to save markdown: %w", err)
	}

	fmt.Printf("Markdown saved: %s\n", tmpFile)
	return openMarkdownPreview(tmpFile)
}

// openMarkdownPreview opens a markdown file using configured or system default application
func openMarkdownPreview(filepath string) error {
	// Check for custom preview command from config
	customCommand := viper.GetString("output.preview_command")
	if customCommand != "" {
		// For CLI tools like glow, we need to run in foreground
		cmd := exec.Command(customCommand, filepath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			fmt.Printf("Custom command '%s' failed: %v\nFalling back to system default. File: %s\n",
				customCommand, err, filepath)
		} else {
			return nil // Success, don't fall back
		}
	}

	// System default fallback
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", filepath)
	case "darwin":
		cmd = exec.Command("open", filepath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filepath)
	default:
		fmt.Printf("Please open the file manually: %s\n", filepath)
		return nil
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Could not open automatically. Please open: %s\n", filepath)
		return nil
	}
	return nil
}

func showProgressWithEstimate(message string, fn func()) error {
	done := make(chan bool, 1)
	start := time.Now()

	go func() {
		fn()
		done <- true
	}()

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-done:
			fmt.Printf("\r%-80s\r✓ %s (%s)\n", "", message, time.Since(start).Round(time.Second))
			return nil
		case <-ticker.C:
			fmt.Printf("\r%s %s", spinner[i%len(spinner)], message)
			i++
		}
	}
}

// getProviderForReview determines which AI provider to use, considering flag overrides
func getProviderForReview(configMgr *config.Manager) (providers.AIProvider, error) {
	if providerFlag != "" {
		viper.Set("audit.default_provider", providerFlag)
	}
	return configMgr.GetActiveProvider()
}

// buildAuditRequest creates the request object for the AI provider
func buildAuditRequest(gitInfo *git.GitDiffInfo, configMgr *config.Manager) (*models.AuditRequest, error) {
	settings := configMgr.GetAuditSettings()

	focus := settings.DefaultFocus
	if len(focusFlag) > 0 {
		focus = make([]models.AuditFocus, 0, len(focusFlag))
		for _, f := range focusFlag {
			auditFocus := models.AuditFocus(f)
			if !isValidFocus(auditFocus) {
				return nil, fmt.Errorf("invalid focus area: %s", f)
			}
			focus = append(focus, auditFocus)
		}
	}

	language := ""
	if settings.AutoDetectLanguage {
		language = detectLanguageFromDiff(gitInfo.Diff)
	}

	return &models.AuditRequest{
		Diff:        gitInfo.Diff,
		Focus:       focus,
		Language:    language,
		MaxTokens:   settings.MaxTokens,
		Temperature: settings.Temperature,
	}, nil
}

func isValidFocus(focus models.AuditFocus) bool {
	validFoci := []models.AuditFocus{
		models.FocusPerformance, models.FocusSecurity, models.FocusBugs,
		models.FocusMaintainability, models.FocusStyle, models.FocusDocumentation,
	}
	for _, valid := range validFoci {
		if focus == valid {
			return true
		}
	}
	return false
}

// detectLanguageFromDiff attempts to determine the primary programming language
func detectLanguageFromDiff(diff string) string {
	languageMap := map[string]string{
		"js":    "javascript",
		"jsx":   "javascript",
		"ts":    "typescript",
		"tsx":   "typescript",
		"py":    "python",
		"pyw":   "python",
		"go":    "go",
		"php":   "php",
		"java":  "java",
		"kt":    "kotlin",
		"scala": "scala",
		"cpp":   "cpp",
		"cc":    "cpp",
		"cxx":   "cpp",
		"c":     "c",
		"h":     "c",
		"hpp":   "cpp",
		"rs":    "rust",
		"rb":    "ruby",
		"cs":    "csharp",
		"swift": "swift",
		"dart":  "dart",
		"sh":    "shell",
		"bash":  "shell",
		"sql":   "sql",
		"html":  "html",
		"css":   "css",
		"scss":  "scss",
		"sass":  "sass",
		"vue":   "vue",
		"yaml":  "yaml",
		"yml":   "yaml",
		"json":  "json",
		"xml":   "xml",
		"toml":  "toml",
		"ini":   "ini",
	}

	lines := strings.Split(diff, "\n")
	extensions := make(map[string]int)
	totalFiles := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filename := parts[1]
				if filename != "/dev/null" {
					filename = strings.TrimPrefix(filename, "a/")
					filename = strings.TrimPrefix(filename, "b/")

					if idx := strings.LastIndex(filename, "."); idx != -1 {
						ext := strings.ToLower(filename[idx+1:])
						extensions[ext]++
						totalFiles++
					}
				}
			}
		}
	}

	if totalFiles < 1 {
		return ""
	}

	maxCount := 0
	primaryExt := ""
	for ext, count := range extensions {
		if count > maxCount {
			maxCount = count
			primaryExt = ext
		}
	}

	confidence := float64(maxCount) / float64(totalFiles)
	if confidence < 0.5 && totalFiles > 1 {
		return ""
	}

	if language, exists := languageMap[primaryExt]; exists {
		return language
	}

	return ""
}

// displayResults shows the audit results to the user in the configured format
func displayResults(response *models.AuditResponse, configMgr *config.Manager) error {
	outputSettings := configMgr.GetOutputSettings()

	format := outputSettings.Format
	if formatFlag != "" {
		format = formatFlag
	}

	switch format {
	case "json":
		return displayJSON(response)
	case "markdown":
		return displayMarkdown(response)
	default:
		return displayTable(response, outputSettings)
	}
}

func displayJSON(response *models.AuditResponse) error {
	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

func displayMarkdown(response *models.AuditResponse) error {
	fmt.Print(response.ToMarkdown())
	return nil
}

func displayTable(response *models.AuditResponse, settings config.OutputSettings) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)

	cyan.Printf("Analysis Summary:\n")
	fmt.Println(wordwrap.WrapString(response.Summary, 80))
	fmt.Println()

	if len(response.Issues) == 0 {
		green.Println("✓ No issues found! Your code looks good.")
		displayCommitInfo(response, cyan)
		return nil
	}

	// Issues table
	data := [][]string{}
	data = append(data, []string{"#", "Severity", "Category", "Issue", "File:Line"})

	for i, issue := range response.Issues {
		location := ""
		if issue.FileName != "" {
			location = issue.FileName
			if issue.LineNumber > 0 {
				location += fmt.Sprintf(":%d", issue.LineNumber)
			}
		}

		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			getSeverityDisplay(issue.Severity),
			string(issue.Category),
			wordwrap.WrapString(issue.Title, 40),
			location,
		})
	}

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"#", "Severity", "Category", "Issue", "File:Line"}),
	)

	for i := 1; i < len(data); i++ {
		table.Append(data[i])
	}
	table.Render()

	// Detailed issues
	for i, issue := range response.Issues {
		displayDetailedIssue(issue, i+1, red, yellow)
	}

	if settings.ShowSuggestions && len(response.Suggestions) > 0 {
		yellow.Println("\n💡 Suggestions:")
		for _, suggestion := range response.Suggestions {
			fmt.Printf("   • %s\n", wordwrap.WrapString(suggestion, 76))
		}
	}

	displayCommitInfo(response, cyan)
	return nil
}

func displayDetailedIssue(issue models.AuditIssue, index int, red, yellow *color.Color) {
	fmt.Printf("\n%s Issue #%d: %s\n", getSeverityIcon(issue.Severity), index, issue.Title)

	if issue.Description != "" {
		fmt.Printf("Description:\n%s\n",
			indentText(wordwrap.WrapString(issue.Description, 76), "  "))
	}

	if issue.CodeSnippet != "" {
		red.Println("\nProblematic Code:")
		fmt.Printf("┌─────────────────────────────────────────────────┐\n")
		lines := strings.Split(strings.TrimSpace(issue.CodeSnippet), "\n")
		for _, line := range lines {
			fmt.Printf("│ %-47s │\n", line)
		}
		fmt.Printf("└─────────────────────────────────────────────────┘\n")
	}

	if issue.RecommendedFix != "" {
		yellow.Println("\nRecommended Fix:")
		fmt.Printf("┌─────────────────────────────────────────────────┐\n")
		lines := strings.Split(strings.TrimSpace(issue.RecommendedFix), "\n")
		for _, line := range lines {
			fmt.Printf("│ %-47s │\n", line)
		}
		fmt.Printf("└─────────────────────────────────────────────────┘\n")
	}
}

func displayCommitInfo(response *models.AuditResponse, cyan *color.Color) {
	if response.ProposedCommitTitle != "" {
		fmt.Println()
		cyan.Println("Suggested Commit:")
		fmt.Printf("Title: %s\n", response.ProposedCommitTitle)
		if response.ProposedCommitBody != "" {
			fmt.Printf("Body:\n%s\n", wordwrap.WrapString(response.ProposedCommitBody, 60))
		}
	}
}

func indentText(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func getSeverityDisplay(severity models.IssueSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return color.RedString("CRITICAL")
	case models.SeverityHigh:
		return color.New(color.FgRed).Sprint("HIGH")
	case models.SeverityMedium:
		return color.YellowString("MEDIUM")
	case models.SeverityLow:
		return color.BlueString("LOW")
	default:
		return "INFO"
	}
}

func getSeverityIcon(severity models.IssueSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return "[CRIT]"
	case models.SeverityHigh:
		return "[HIGH]"
	case models.SeverityMedium:
		return "[MED]"
	case models.SeverityLow:
		return "[LOW]"
	default:
		return "[INFO]"
	}
}
