package analyze

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/sthembisoo/ai-review-pr/cmd/raygun/types"
)

//go:embed prompt.tmpl
var promptTemplate string

const (
	raygunAPIBase    = "https://api.raygun.com/v3"
	claudeModelHaiku = "Haiku"
)

var (
	raygunProject string
	raygunToken   string
	repoPath      string
	branch        string
)

func NewCmdRaygunErrors() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Analyze raygun errors using AI",
		Long: `Analyze raygun errors using AI.

This command will:
1. Let you select a Raygun project
2. Fetch and display recent active crash reports
3. Let you select a specific crash to analyze
4. Run Claude AI analysis on the crash in your local repository

Examples:
  # Interactive mode - Analyze a crash report from Raygun
  raygun-errors --repo /path/to/repo --token YOUR_RAYGUN_TOKEN

  # Specify raygun project
  raygun-errors --repo /path/to/repo --token YOUR_RAYGUN_TOKEN --raygun-project "MyApp-prod"

  # Specify branch to checkout
  raygun-errors --repo /path/to/repo --token YOUR_RAYGUN_TOKEN --branch main`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return start()
		},
	}

	cmd.Flags().StringVarP(&raygunProject, "raygun-project", "p", "", "Raygun project name")
	cmd.Flags().StringVarP(&raygunToken, "token", "t", "", "Raygun API access token (or set RAYGUN_TOKEN env var)")
	cmd.Flags().StringVarP(&repoPath, "repo", "r", ".", "Path to the git repository")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to checkout (optional)")

	return cmd
}

func start() error {
	// Get Raygun token
	token := raygunToken
	if token == "" {
		token = os.Getenv("RAYGUN_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("raygun token required: use --token flag or set RAYGUN_TOKEN environment variable")
	}

	// Resolve repo path
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repo path: %w", err)
	}

	// Verify it's a git repository
	if _, err := os.Stat(filepath.Join(absRepoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", absRepoPath)
	}

	// Choose Raygun application (Project)
	raygunApp, err := chooseRaygunApplication(token)
	if err != nil {
		return fmt.Errorf("error selecting project: %w", err)
	}

	fmt.Printf("Selected raygun project: %s\n\n", raygunApp.Name)

	// List crash groups
	errorGroups, err := listErrorGroups(token, *raygunApp)
	if err != nil {
		return fmt.Errorf("error fetching crash reports: %w", err)
	}

	if len(errorGroups) == 0 {
		fmt.Println("No crash reports found for this project.")
		return nil
	}

	// Show only error groups that have a status of active
	activeErrorGroups := lo.Filter(errorGroups, func(errorGroup types.ErrorGroup, _ int) bool {
		return errorGroup.Status == "active"
	})

	if len(activeErrorGroups) == 0 {
		fmt.Println("No active crash reports found for this project.")
		return nil
	}

	// Choose a crash report
	selectedErrorGroup, err := chooseErrorGroup(activeErrorGroups)
	if err != nil {
		return fmt.Errorf("error selecting crash: %w", err)
	}

	fmt.Printf("\nAnalyzing crash: %s\n", selectedErrorGroup.Message)

	// Get detailed crash data
	errorDetail, err := getErrorReportDetail(token, raygunApp.Identifier, selectedErrorGroup.Identifier)
	if err != nil {
		return fmt.Errorf("error fetching crash details: %w", err)
	}

	// Checkout branch if specified
	if branch != "" {
		if err := checkoutBranch(absRepoPath, branch); err != nil {
			return fmt.Errorf("error checking out branch: %w", err)
		}
	}

	// Launch Claude Code with the crash analysis prompt
	raygunErrorFilePath := filepath.Join(absRepoPath, "RaygunError.md")
	err = launchClaudeErrorAnalysis(absRepoPath, raygunErrorFilePath, *errorDetail)
	if err != nil {
		return fmt.Errorf("error launching Claude analysis: %w", err)
	}

	fmt.Println("\nClaude error analysis complete")
	return nil
}

// chooseRaygunApplication fetches and prompts user to select a Raygun application
func chooseRaygunApplication(token string) (*types.RaygunApplication, error) {
	applications, err := fetchApplications(token, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch applications: %w", err)
	}

	if len(applications) == 0 {
		return nil, fmt.Errorf("no applications found")
	}

	if raygunProject != "" {
		application, exists := lo.Find(applications, func(app types.RaygunApplication) bool {
			return app.Name == raygunProject
		})
		if exists {
			return &application, nil
		}
		return nil, fmt.Errorf("raygun project '%s' not found", raygunProject)
	}

	// Build display options
	fmt.Println("Available Raygun projects:")
	for i, app := range applications {
		fmt.Printf("  %d. %s\n", i+1, app.Name)
	}

	var selection int
	fmt.Print("\nSelect project number: ")
	if _, err := fmt.Scanf("%d", &selection); err != nil {
		return nil, fmt.Errorf("invalid selection: %w", err)
	}

	if selection < 1 || selection > len(applications) {
		return nil, fmt.Errorf("selection out of range")
	}

	return &applications[selection-1], nil
}

// fetchApplications retrieves all applications from Raygun API
func fetchApplications(token string, count int) ([]types.RaygunApplication, error) {
	endpointURL := fmt.Sprintf("%s/applications?count=%d", raygunAPIBase, count)

	client := resty.New()
	response, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		Get(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch applications: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("raygun API returned status %d: %s", response.StatusCode(), string(response.Body()))
	}

	var applications []types.RaygunApplication
	err = json.Unmarshal(response.Body(), &applications)
	if err != nil {
		return nil, fmt.Errorf("failed to decode applications: %w", err)
	}

	return applications, nil
}

// listErrorGroups fetches error groups from Raygun
func listErrorGroups(token string, app types.RaygunApplication) ([]types.ErrorGroup, error) {
	endpointURL := fmt.Sprintf("%s/applications/%s/error-groups", raygunAPIBase, app.Identifier)

	client := resty.New()
	response, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		Get(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch error groups: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("raygun API returned status %d: %s", response.StatusCode(), string(response.Body()))
	}

	var errorGroups []types.ErrorGroup
	err = json.Unmarshal(response.Body(), &errorGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to decode error groups: %w", err)
	}

	return errorGroups, nil
}

// chooseErrorGroup prompts user to select an error group
func chooseErrorGroup(errorGroups []types.ErrorGroup) (*types.ErrorGroup, error) {
	fmt.Println("\nActive error groups:")
	for i, eg := range errorGroups {
		// Truncate message if too long
		msg := eg.Message
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}
		fmt.Printf("  %d. [%d occurrences] %s\n", i+1, eg.Count, msg)
	}

	var selection int
	fmt.Print("\nSelect error group number: ")
	if _, err := fmt.Scanf("%d", &selection); err != nil {
		return nil, fmt.Errorf("invalid selection: %w", err)
	}

	if selection < 1 || selection > len(errorGroups) {
		return nil, fmt.Errorf("selection out of range")
	}

	return &errorGroups[selection-1], nil
}

// getErrorReportDetail fetches detailed error information
func getErrorReportDetail(token, appIdentifier, errorGroupIdentifier string) (*types.CrashReportDetail, error) {
	endpointURL := fmt.Sprintf("%s/applications/%s/error-groups/%s/errors?count=1", raygunAPIBase, appIdentifier, errorGroupIdentifier)

	client := resty.New()
	response, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Content-Type", "application/json").
		Get(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch error detail: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("raygun API returned status %d: %s", response.StatusCode(), string(response.Body()))
	}

	var details []types.CrashReportDetail
	err = json.Unmarshal(response.Body(), &details)
	if err != nil {
		return nil, fmt.Errorf("failed to decode error detail: %w", err)
	}

	if len(details) == 0 {
		return nil, fmt.Errorf("no error details found")
	}

	return &details[0], nil
}

func checkoutBranch(repoDir, branchName string) error {
	// Fetch latest
	fetchCmd := exec.Command("git", "fetch", "--all")
	fetchCmd.Dir = repoDir
	fetchCmd.Run() // Ignore fetch errors

	// Checkout branch
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w\n%s", branchName, err, string(output))
	}

	// Pull latest
	pullCmd := exec.Command("git", "pull")
	pullCmd.Dir = repoDir
	pullCmd.Run() // Ignore pull errors

	fmt.Printf("Checked out branch: %s\n", branchName)
	return nil
}

// promptData holds the data for the crash analysis prompt template
type promptData struct {
	ErrorMessage     string
	CrashDetails     []byte
	AnalysisFilePath string
}

func launchClaudeErrorAnalysis(repoDir, errorFilePath string, crashDetail types.CrashReportDetail) error {
	fmt.Println("Launching Claude Code for crash analysis...")

	// Convert struct to json
	crashDetailJson, err := json.MarshalIndent(crashDetail, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal crash detail: %w", err)
	}

	data := promptData{
		ErrorMessage:     crashDetail.Error.Message,
		CrashDetails:     crashDetailJson,
		AnalysisFilePath: errorFilePath,
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
	cmd := exec.Command("claude", "--model="+claudeModelHaiku, "--dangerously-skip-permissions", "-p", promptBuf.String())
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	// Open the output file after completion
	if _, err := os.Stat(errorFilePath); err == nil {
		fmt.Printf("Opening output file: %s\n", errorFilePath)
		openCmd := exec.Command("open", errorFilePath)
		if err := openCmd.Run(); err != nil {
			fmt.Printf("Warning: could not open output file: %v\n", err)
		}
	}

	return nil
}
