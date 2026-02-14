package git

import (
	"testing"
)

func TestExtractRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:user/repo",
			expected: "repo",
		},
		{
			name:     "SSH protocol URL",
			url:      "ssh://git@github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "unknown",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoNameFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("extractRepoNameFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// Note: ExtractStagedChanges() requires an actual git repository
// For testing, we would need to:
// 1. Create a mock git environment
// 2. Use dependency injection to mock exec.Command
// 3. Use integration tests with a real git repo
//
// Example of what a mock-based test might look like:
//
// func TestExtractStagedChanges_MockedGit(t *testing.T) {
//     // This would require refactoring to use an interface for git commands
//     // mockGit := &MockGitClient{
//     //     diff: "mock diff content",
//     //     branch: "main",
//     // }
//     // result, err := mockGit.GetStagedChanges()
//     // if err != nil {
//     //     t.Fatalf("unexpected error: %v", err)
//     // }
//     // ... assertions
// }
