package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ai_review "github.com/sthembisoo/ai-review-pr/cmd/ai-review"
	raygun_errors "github.com/sthembisoo/ai-review-pr/cmd/raygun/errors"
)

var rootCmd = &cobra.Command{
	Use:   "ai-review-pr",
	Short: "AI-powered code review and error analysis tools",
}

func main() {
	rootCmd.AddCommand(ai_review.NewCmdAIReview())
	rootCmd.AddCommand(raygun_errors.NewCmdRaygunErrors())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
