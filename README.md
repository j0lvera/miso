# miso 

`miso` analyzes your code and provides constructive feedback based on file-specific guidelines. It supports:

- **Single file review**: Review individual code files with AI-powered insights
- **Git diff review**: Analyze changes between git commits or branches
- **File-specific guides**: Uses tailored review guidelines based on file naming patterns (e.g., `.page.tsx`, `.utils.ts`, `.hooks.ts`)

## Usage

### Prerequisites

Set your OpenRouter API key:
```bash
export OPENROUTER_API_KEY=your-api-key-here
```

### Commands

#### Review a single file
```bash
miso review path/to/file.tsx
```

Options:
- `-v, --verbose`: Enable verbose output
- `-m, --message`: Custom message to display while processing (default: "Thinking...")

#### Review git changes
```bash
# Review changes in the last commit
miso diff

# Review changes between commits
miso diff HEAD~3

# Review changes between branches
miso diff main..feature-branch
```

Options:
- `-v, --verbose`: Enable verbose output
- `-m, --message`: Custom message to display while processing (default: "Analyzing changes...")

#### Show version
```bash
miso version
```

## Configuration

`miso` uses pattern-based configuration to determine which files to review and which guidelines to apply. You can configure it using a `miso.yml` file in your repository root or embed the configuration in GitHub Actions.

### Configuration File

Create a `miso.yml` file in your repository root:

```yaml
# Global defaults for content scanning
content_defaults:
  strategy: "first_lines"  # first_lines, full_file, or smart
  lines: 50               # Number of lines to scan (for first_lines and smart)

# Pattern matching rules (evaluated in order)
patterns:
  - name: "go-test-files"
    filename: "_test\\.go$"
    context:
      - testing.md
    stop: true  # Don't evaluate further patterns for test files
  
  - name: "go-files"
    filename: "\\.go$"
    context:
      - go.md
      - error-handling.md
  
  - name: "http-handlers"
    filename: "/handlers/"
    context:
      - http-handlers.md
      - validation.md
  
  - name: "database-code"
    content: "SELECT|INSERT|UPDATE|DELETE|gorm\\.|sql\\."
    content_strategy: "full_file"  # Override global default
    context:
      - database.md
      - sql-security.md
  
  - name: "react-components"
    filename: "\\.(tsx|jsx)$"
    content: "import.*react|from ['\"]react"
    context:
      - react.md
      - components.md
  
  - name: "security-sensitive"
    content: "password|token|secret|auth|crypto"
    content_strategy: "full_file"
    context:
      - security.md
    stop: true  # Security issues need focused review
```

### Pattern Matching

#### Filename Patterns
- Use regular expressions to match file paths
- Examples: `"\\.go$"`, `"/handlers/"`, `"_test\\.go$"`

#### Content Patterns  
- Use regular expressions to match file contents
- Examples: `"import.*react"`, `"SELECT|INSERT"`, `"password|token"`

#### Content Scanning Strategies

**first_lines** (default):
```yaml
content_strategy: "first_lines"
content_lines: 50  # Scan first 50 lines
```

**full_file**:
```yaml
content_strategy: "full_file"  # Scan entire file
```

**smart**:
```yaml
content_strategy: "smart"
content_lines: [100, 100, 100]  # First 100, last 100, 100 random lines
```

### Pattern Evaluation

Patterns are evaluated **additively** with optional **stop flags**:

- `handlers/user.go` → `[go.md, error-handling.md, http-handlers.md, validation.md]`
- `handlers/user_test.go` → `[testing.md]` (stops due to stop flag)
- `models/user.go` → `[go.md, error-handling.md]`

### GitHub Actions Integration

miso can be easily integrated into your GitHub Actions workflows for automated code review on pull requests and pushes.

#### Quick Setup

1. **Add the workflow file** to your repository:

```bash
mkdir -p .github/workflows
curl -o .github/workflows/code-review.yml https://raw.githubusercontent.com/your-org/miso/main/.github/workflows/example.yml
```

2. **Set up your API key**:
   - Go to your repository Settings → Secrets and variables → Actions
   - Add a new secret named `OPENROUTER_API_KEY`
   - Set the value to your OpenRouter API key

3. **Add a configuration file** (optional):

```yaml
# miso.yml
content_defaults:
  strategy: "smart"
  lines: 100

patterns:
  - name: "source-files"
    filename: "\\.(go|ts|js|py)$"
    diff_context:
      - diff/breaking-changes.md
      - diff/security-review.md
```

#### Basic Workflow Example

```yaml
name: Code Review
on:
  pull_request:
    branches: [ main ]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install miso 
      run: go build -o miso cmd/main/main.go
    
    - name: Review changes
      env:
        OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
      run: |
        ./miso diff "${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }}"
```

For more advanced workflows and configuration options, see the [GitHub Actions examples](.github/workflows/).

### Guide Files

Place your guide files in a `guides/` directory:

```
guides/
├── go.md
├── testing.md
├── security.md
├── react/
│   ├── components.md
│   └── hooks.md
└── backend/
    ├── database.md
    └── api.md
```

Multiple guide files are combined into a single review context, allowing you to layer general and specific guidance.

## How it works

The tool uses pattern-based configuration to determine which files to review and which guidelines to apply. Files are matched against filename and content patterns, with multiple guides combined to provide comprehensive, context-aware code reviews.

Only files with matching patterns will be analyzed.
