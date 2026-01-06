package claude

import (
	"fmt"
	"os"
	"os/exec"
)

var (
	ClaudeModelHaiku  = "Haiku"
	ClaudeModelSonnet = "Sonnet"
)

func LaunchClaude(repoDir, outputFilePath, prompt string, model *string) error {
	claudeModel := ClaudeModelSonnet
	if model != nil && *model != "" {
		claudeModel = *model
	}

	cmd := exec.Command("claude", "--model="+claudeModel, "--dangerously-skip-permissions", "-p", prompt)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	// Open the output file after completion
	if _, err := os.Stat(outputFilePath); err == nil {
		fmt.Printf("Opening output file: %s\n", outputFilePath)
		openCmd := exec.Command("open", outputFilePath)
		if err := openCmd.Run(); err != nil {
			fmt.Printf("Warning: could not open output file: %v\n", err)
		}
	}

	return nil
}
