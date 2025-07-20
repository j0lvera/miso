# go-review

An AI-powered code review tool that provides intelligent feedback on your code using Claude via OpenRouter. The tool is designed to be model-agnostic and can work with any OpenAI-compatible API through OpenRouter.

## What it does

`go-review` analyzes your code and provides constructive feedback based on file-specific guidelines. It supports:

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
go-review review path/to/file.tsx
```

Options:
- `-v, --verbose`: Enable verbose output
- `-m, --message`: Custom message to display while processing (default: "Thinking...")

#### Review git changes
```bash
# Review changes in the last commit
go-review diff

# Review changes between commits
go-review diff HEAD~3

# Review changes between branches
go-review diff main..feature-branch
```

Options:
- `-v, --verbose`: Enable verbose output
- `-m, --message`: Custom message to display while processing (default: "Analyzing changes...")

#### Show version
```bash
go-review version
```

## How it works

The tool uses file naming patterns to determine which review guidelines to apply. For example:
- Files ending in `.page.tsx` use page-specific review guidelines
- Files ending in `.utils.ts` use utility function guidelines
- Files ending in `.hooks.ts` use React hooks guidelines

Only files with matching review guides will be analyzed.
