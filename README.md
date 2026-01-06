# ai-review-pr

AI-powered code review and error analysis CLI tools using Claude.

## Installation

```bash
go install github.com/sthembisoo/ai-review-pr@latest
```

Or build from source:

```bash
git clone https://github.com/sthembisoo/ai-review-pr.git
cd ai-review-pr
go install .
```

Make sure `~/go/bin` is in your `PATH`.

## Prerequisites

- [Claude CLI](https://github.com/anthropics/claude-code) installed and configured
- Go 1.21+

## Commands

### ai-review

Perform AI code review on a git branch using Claude.

```bash
# Review current branch against main
ai-review-pr ai-review --repo /path/to/repo

# Review a specific branch
ai-review-pr ai-review --repo /path/to/repo --branch feature-branch

# Specify target branch for diff (default: main)
ai-review-pr ai-review --repo /path/to/repo --branch feature-branch --target dev
```

**Flags:**
- `--repo, -r` - Path to the git repository (default: current directory)
- `--branch, -b` - Branch to review (default: current branch)
- `--target, -t` - Target branch to diff against (default: main)

**Output:** Creates a `Review.md` file in the repository with the AI review.

### errors

Analyze Raygun crash reports using Claude AI.

```bash
# Interactive mode
ai-review-pr errors --repo /path/to/repo --token YOUR_RAYGUN_TOKEN

# Or use environment variable for token
export RAYGUN_TOKEN=your_token
ai-review-pr errors --repo /path/to/repo

# Specify Raygun project directly
ai-review-pr errors --repo /path/to/repo --raygun-project "MyApp-prod"

# Checkout a specific branch before analysis
ai-review-pr errors --repo /path/to/repo --branch main
```

**Flags:**
- `--repo, -r` - Path to the git repository (default: current directory)
- `--token, -t` - Raygun API access token (or set `RAYGUN_TOKEN` env var)
- `--raygun-project, -p` - Raygun project name (interactive selection if not specified)
- `--branch, -b` - Branch to checkout before analysis (optional)

**Output:** Creates a `RaygunError.md` file in the repository with the crash analysis.

## License

MIT
