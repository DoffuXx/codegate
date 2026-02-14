package git

import (
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
	// Check if we're in a git repository first
	if err := checkGitRepository(); err != nil {
		return nil, err
	}

	info := &GitDiffInfo{}

	// Get staged changes diff
	diff, err := getGitDiff()
	if err != nil {
		return nil, err
	}
	info.Diff = diff

	// Get current branch name
	branch, err := getCurrentBranch()
	if err != nil {
		return nil, err
	}
	info.BranchName = branch

	// Get current commit hash (optional, ignore errors)
	if hash, err := getCurrentCommitHash(); err == nil {
		info.CommitHash = hash
	}

	// Get repository name (optional, ignore errors)
	if repoName, err := getRepositoryName(); err == nil {
		info.RepoName = repoName
	}

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
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not in a git repository")
	}
	return nil
}

// getGitDiff gets the staged changes diff
func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "not a git repository") {
			return "", fmt.Errorf("not in a git repository")
		}
		return "", fmt.Errorf("git command failed: %w\nOutput: %s", err, output)
	}
	return string(output), nil
}

// getCurrentBranch gets the current branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w\nOutput: %s", err, output)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		// Fallback for detached HEAD state
		return getDetachedHeadInfo()
	}

	return branch, nil
}

// getDetachedHeadInfo handles detached HEAD state
func getDetachedHeadInfo() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown", nil // Don't fail, just return unknown
	}

	shortHash := strings.TrimSpace(string(output))
	return fmt.Sprintf("detached-HEAD-%s", shortHash), nil
}

// getCurrentCommitHash gets the current commit hash
func getCurrentCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	hash := strings.TrimSpace(string(output))
	// Return short hash for readability
	if len(hash) > 7 {
		return hash[:7], nil
	}
	return hash, nil
}

// getRepositoryName extracts repository name from remote URL
func getRepositoryName() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	url := strings.TrimSpace(string(output))
	return extractRepoNameFromURL(url), nil
}

// extractRepoNameFromURL extracts repository name from git URL
func extractRepoNameFromURL(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle different URL formats:
	// https://github.com/user/repo
	// git@github.com:user/repo
	// ssh://git@github.com/user/repo

	if strings.Contains(url, "/") {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return "unknown"
}
