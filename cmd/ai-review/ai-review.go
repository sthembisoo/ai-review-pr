package ai_review

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed prompt.tmpl
var promptTemplate string

var (
	flagBranch       string
	flagTargetBranch string
	flagRepoPath     string
)

const (
	claudeModelSonnet = "Sonnet"
)

func NewCmdAIReview() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-review",
		Short: "Perform AI code review on a git branch",
		Long: `Perform AI code review on a git branch using Claude.

Examples:
  # Review current branch against main
  ai-review --repo /path/to/repo --branch feature-branch

  # Specify target branch for diff (default: main)
  ai-review --repo /path/to/repo --branch feature-branch --target dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return start()
		},
	}

	cmd.Flags().StringVarP(&flagRepoPath, "repo", "r", ".", "Path to the git repository")
	cmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "Branch to review (defaults to current branch)")
	cmd.Flags().StringVarP(&flagTargetBranch, "target", "t", "main", "Target branch to diff against")

	return cmd
}

func start() error {
	// Resolve repo path
	repoPath, err := filepath.Abs(flagRepoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repo path: %w", err)
	}

	// Verify it's a git repository
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	fmt.Printf("Reviewing repository: %s\n", repoPath)

	// Get current branch if not specified
	branchName := flagBranch
	if branchName == "" {
		branchName, err = getCurrentBranch(repoPath)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	fmt.Printf("Branch: %s (comparing against %s)\n", branchName, flagTargetBranch)

	// Get diff between current branch and target branch
	diff, err := getDiff(repoPath, branchName, flagTargetBranch)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		fmt.Println("No differences found between branches.")
		return nil
	}

	// Launch Claude Code with the review prompt
	if err := launchClaudeReview(repoPath, branchName, diff); err != nil {
		return fmt.Errorf("failed to launch Claude review: %w", err)
	}

	fmt.Println("Claude code review complete.")
	return nil
}

func getCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getDiff(repoDir string, branchName string, targetBranch string) (string, error) {
	fmt.Printf("Getting diff between %s and %s...\n", branchName, targetBranch)

	// Try with origin/ prefix first, fall back to local branch
	targetRef := "origin/" + targetBranch

	// Check if origin/target exists
	checkCmd := exec.Command("git", "rev-parse", "--verify", targetRef)
	checkCmd.Dir = repoDir
	if err := checkCmd.Run(); err != nil {
		// Fall back to local branch
		targetRef = targetBranch
	}

	// Get the diff stats first
	statsCmd := exec.Command("git", "diff", "--stat", targetRef+"...HEAD")
	statsCmd.Dir = repoDir
	statsOutput, err := statsCmd.Output()
	if err != nil {
		// Try without the three-dot syntax
		statsCmd = exec.Command("git", "diff", "--stat", targetRef)
		statsCmd.Dir = repoDir
		statsOutput, err = statsCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff stats: %w", err)
		}
	}

	// Get the full diff
	diffCmd := exec.Command("git", "diff", targetRef+"...HEAD")
	diffCmd.Dir = repoDir
	diffOutput, err := diffCmd.Output()
	if err != nil {
		// Try without the three-dot syntax
		diffCmd = exec.Command("git", "diff", targetRef)
		diffCmd.Dir = repoDir
		diffOutput, err = diffCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff: %w", err)
		}
	}

	// Combine stats and diff
	result := fmt.Sprintf("=== DIFF STATS ===\n%s\n\n=== FULL DIFF ===\n%s", string(statsOutput), string(diffOutput))
	return result, nil
}

// promptData holds the data for the review prompt template
type promptData struct {
	BranchName     string
	TargetBranch   string
	Diff           string
	ReviewFilePath string
}

func launchClaudeReview(repoDir string, branchName string, diff string) error {
	fmt.Println("Launching Claude Code for review...")

	reviewFilePath := filepath.Join(repoDir, "Review.md")

	// Build prompt from template
	data := promptData{
		BranchName:     branchName,
		TargetBranch:   flagTargetBranch,
		Diff:           diff,
		ReviewFilePath: reviewFilePath,
	}

	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var promptBuf strings.Builder
	if err := tmpl.Execute(&promptBuf, data); err != nil {
		return fmt.Errorf("failed to execute prompt template: %w", err)
	}

	// Launch claude with the prompt
	cmd := exec.Command("claude", "--model="+claudeModelSonnet, "--dangerously-skip-permissions", "-p", promptBuf.String())
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	// Open the review file after completion
	if _, err := os.Stat(reviewFilePath); err == nil {
		fmt.Printf("Opening review file: %s\n", reviewFilePath)
		openCmd := exec.Command("open", reviewFilePath)
		if err := openCmd.Run(); err != nil {
			fmt.Printf("Warning: could not open review file: %v\n", err)
		}
	}

	return nil
}
