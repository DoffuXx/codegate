package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GitDiffInfo contains git diff and repository information
type GitDiffInfo struct {
	Diff       string `json:"diff"`
	BranchName string `json:"branch_name"`
	RepoName   string `json:"repo_name,omitempty"`
	CommitHash string `json:"commit_hash,omitempty"`
}

// ExtractStagedChanges gets the diff of currently staged changes with git context
func ExtractStagedChanges() (*GitDiffInfo, error) {
	if err := checkGitRepository(); err != nil {
		return nil, err
	}

	info := &GitDiffInfo{}

	// Get staged changes diff (required)
	diff, err := getGitDiff()
	if err != nil {
		return nil, err
	}
	info.Diff = diff

	// Get current branch name (required)
	branch, err := getCurrentBranch()
	if err != nil {
		return nil, err
	}
	info.BranchName = branch

	// Get optional metadata (don't fail on errors)
	info.CommitHash, _ = getCurrentCommitHash()
	info.RepoName, _ = getRepositoryName()

	return info, nil
}

// ExtractStagedChangesLegacy returns only the diff string for backward compatibility
func ExtractStagedChangesLegacy() (string, error) {
	gitInfo, err := ExtractStagedChanges()
	if err != nil {
		return "", err
	}
	return gitInfo.Diff, nil
}

// checkGitRepository verifies we're in a git repository
func checkGitRepository() error {
	if err := runGitCommand("rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not in a git repository")
	}
	return nil
}

// getGitDiff gets the staged changes diff
func getGitDiff() (string, error) {
	output, err := runGitCommandOutput("diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	return output, nil
}

// getCurrentBranch gets the current branch name
func getCurrentBranch() (string, error) {
	output, err := runGitCommandOutput("branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		// Fallback for detached HEAD state
		return getDetachedHeadInfo()
	}

	return branch, nil
}

// getDetachedHeadInfo handles detached HEAD state
func getDetachedHeadInfo() (string, error) {
	shortHash, err := runGitCommandOutput("rev-parse", "--short", "HEAD")
	if err != nil {
		return "unknown", nil // Don't fail, just return unknown
	}
	return fmt.Sprintf("detached-HEAD-%s", strings.TrimSpace(shortHash)), nil
}

// getCurrentCommitHash gets the current commit hash (short version)
func getCurrentCommitHash() (string, error) {
	hash, err := runGitCommandOutput("rev-parse", "--short=7", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(hash), nil
}

// getRepositoryName extracts repository name from remote URL
func getRepositoryName() (string, error) {
	url, err := runGitCommandOutput("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return extractRepoNameFromURL(strings.TrimSpace(url)), nil
}

// extractRepoNameFromURL extracts repository name from git URL
// Handles: https://github.com/user/repo, git@github.com:user/repo, ssh://git@github.com/user/repo
func extractRepoNameFromURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Replace colon with slash for SSH URLs (git@github.com:user/repo -> git@github.com/user/repo)
	if strings.Contains(url, ":") && !strings.Contains(url, "://") {
		url = strings.Replace(url, ":", "/", 1)
	}

	// Extract last path segment
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// runGitCommand executes a git command and returns error if it fails
func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}
	return nil
}

// runGitCommandOutput executes a git command and returns output
func runGitCommandOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
