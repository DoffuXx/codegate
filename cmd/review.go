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

Examples:
  git-audit review                    # Analyze with default settings
  git-audit review --focus security  # Focus only on security issues  
  git-audit review --format json     # Output results as JSON`,

	RunE: runReview,
}

// Command-specific flags for the review command
var (
	focusFlag    []string // Specific focus areas for this review
	formatFlag   string   // Output format override
	providerFlag string   // Provider override for this review
	previewFlag  bool     // Whether to generate markdown preview in browser
)

// init sets up the review command flags and adds it to the root command
func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().StringSliceVarP(&focusFlag, "focus", "f", nil,
		"focus areas: performance, security, bugs, maintainability, style, documentation")

	reviewCmd.Flags().StringVar(&formatFlag, "format", "",
		"output format: table, json, markdown (overrides config)")

	reviewCmd.Flags().StringVarP(&providerFlag, "provider", "p", "",
		"AI provider to use: lmstudio, openai, claude (overrides config)")

	reviewCmd.Flags().BoolVar(&previewFlag, "preview", false, "generate markdown and open in browser")
}

// runReview implements the main logic for the review command
func runReview(cmd *cobra.Command, args []string) error {
	viper.Set("preview", previewFlag)

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

	// Handle preview mode
	if previewFlag {
		return handlePreviewMode(response)
	}

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
	return openFileInBrowser(tmpFile)
}

// openFileInBrowser opens a file using the system's default application
func openFileInBrowser(filepath string) error {
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
		return nil // Don't return error, just inform user
	}

	return nil
}

func showProgressWithEstimate(message string, fn func()) error {
	done := make(chan bool, 1)
	errorChan := make(chan error, 1)
	start := time.Now()

	quotes := []string{
		"Analyzing code patterns",
		"Checking for potential bugs",
		"Reviewing security practices",
		"Examining performance patterns",
		"Validating code quality",
		"Scanning for improvements",
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("analysis panicked: %v", r)
				return
			}
		}()
		fn()
		done <- true
	}()

	fmt.Printf("%s ", message)

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ticker := time.NewTicker(progressUpdateRate)
	quoteTicker := time.NewTicker(quoteUpdateRate)
	defer ticker.Stop()
	defer quoteTicker.Stop()

	spinnerIndex := 0
	quoteIndex := 0

	for {
		select {
		case <-done:
			elapsed := time.Since(start)
			fmt.Printf("\r%s completed in %s%s\n",
				message, elapsed.Round(time.Second), strings.Repeat(" ", 10))
			return nil
		case err := <-errorChan:
			elapsed := time.Since(start)
			fmt.Printf("\r%s failed after %s%s\n",
				message, elapsed.Round(time.Second), strings.Repeat(" ", 10))
			return err
		case <-quoteTicker.C:
			quoteIndex = (quoteIndex + 1) % len(quotes)
		case <-ticker.C:
			elapsed := time.Since(start)
			spinnerIndex = (spinnerIndex + 1) % len(spinner)

			timeInfo := ""
			if elapsed > 5*time.Second {
				timeInfo = fmt.Sprintf(" (%s)", elapsed.Round(time.Second))
			}

			display := fmt.Sprintf("%s %s %s%s",
				message, spinner[spinnerIndex], quotes[quoteIndex], timeInfo)

			if len(display) > maxDisplayWidth {
				display = display[:maxDisplayWidth-3] + "..."
			}

			fmt.Printf("\r%-*s", maxDisplayWidth, display)
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
		green.Println("No issues found! Your code looks good.")
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
