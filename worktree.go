package claudes

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	claudecode "github.com/severity1/claude-agent-sdk-go"
)

// WorktreeInfo contains information about a git worktree for a session
type WorktreeInfo struct {
	Name   string // Generated name for the worktree
	Branch string // Git branch name
	Path   string // Full path to the worktree directory
}

// GenerateWorktreeName uses Claude SDK to generate a unique worktree name
// To use Haiku model for faster, cost-effective name generation, set:
//   export ANTHROPIC_MODEL=claude-haiku-4
func GenerateWorktreeName(ctx context.Context, prompt string, sessionID string) (string, error) {
	var generatedName string

	// Create a prompt for Claude to generate a short, descriptive name
	namePrompt := fmt.Sprintf(`Based on this task description, generate a short, descriptive name suitable for a git branch.

Task: %s

Requirements:
- Maximum 50 characters
- Use lowercase letters, numbers, and hyphens only
- Be descriptive but concise
- Start with a word describing the action (add, fix, update, refactor, etc.)
- Example formats: "add-user-authentication", "fix-memory-leak", "update-api-endpoints"

Respond with ONLY the branch name, nothing else.`, prompt)

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// Create a temporary session for name generation
		// Note: Using default model (can be configured via ANTHROPIC_MODEL env var)
		tempSessionID := fmt.Sprintf("name-gen-%s", sessionID)
		if err := client.QueryWithSession(ctx, namePrompt, tempSessionID); err != nil {
			return fmt.Errorf("failed to query Claude: %w", err)
		}

		// Collect the response
		msgChan := client.ReceiveMessages(ctx)
		var response strings.Builder

		for msg := range msgChan {
			switch m := msg.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(*claudecode.TextBlock); ok {
						response.WriteString(tb.Text)
					}
				}
			}
		}

		generatedName = strings.TrimSpace(response.String())
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("Claude SDK error: %w", err)
	}

	// Sanitize the generated name to ensure it's valid for git branch names
	generatedName = sanitizeBranchName(generatedName)

	// Append session ID suffix to ensure uniqueness
	finalName := fmt.Sprintf("%s-%s", generatedName, sessionID[:5])

	return finalName, nil
}

// sanitizeBranchName ensures the name is valid for git branch names
func sanitizeBranchName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace any whitespace or invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	name = reg.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Collapse multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Limit length to 50 characters
	if len(name) > 50 {
		name = name[:50]
		name = strings.TrimSuffix(name, "-")
	}

	// If empty after sanitization, use a default
	if name == "" {
		name = "task"
	}

	return name
}

// CreateWorktree creates a git worktree with the specified branch
func CreateWorktree(baseDir, branchName, worktreeName string) (*WorktreeInfo, error) {
	// Get the current branch as the starting point
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := strings.TrimSpace(string(output))

	// Create the worktree path
	worktreePath := filepath.Join(baseDir, worktreeName)

	// Create the new branch starting from current branch
	fullBranchName := fmt.Sprintf("claude/%s", branchName)

	// Create worktree with new branch
	cmd = exec.Command("git", "worktree", "add", "-b", fullBranchName, worktreePath, currentBranch)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	return &WorktreeInfo{
		Name:   worktreeName,
		Branch: fullBranchName,
		Path:   worktreePath,
	}, nil
}

// RemoveWorktree removes a git worktree
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}
