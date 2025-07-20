package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/j0lvera/go-review/internal/agents"
	"github.com/j0lvera/go-review/internal/git"
	"github.com/j0lvera/go-review/internal/router"
)

var version = "dev"

type CLI struct {
	Review  ReviewCmd  `cmd:"" help:"Review a code file"`
	Diff    DiffCmd    `cmd:"" help:"Review changes in a git diff"`
	Version VersionCmd `cmd:"" help:"Show version"`
}

type ReviewCmd struct {
	File    string `arg:"" required:"" help:"Path to the file to review" type:"existingfile"`
	Verbose bool   `short:"v" help:"Enable verbose output"`
	Message string `short:"m" help:"Message to display while processing" default:"Thinking..."`
}

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("go-review version %s\n", version)
	return nil
}

func (r *ReviewCmd) Run() error {
	// Read file contents
	content, err := os.ReadFile(r.File)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", r.File, err)
	}

	if r.Verbose {
		fmt.Printf("Reviewing file: %s\n", r.File)
	}

	// Initialize reviewer
	reviewer, err := agents.NewCodeReviewer()
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}

	// Get just the filename for the review
	filename := filepath.Base(r.File)

	// Create and start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + r.Message
	s.Start()

	// Perform review
	result, err := reviewer.Review(string(content), filename)
	
	// Stop spinner
	s.Stop()
	
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	fmt.Println(result.Content)
	
	// Display token usage if available
	if result.TokensUsed > 0 {
		fmt.Printf("\n---\n")
		fmt.Printf("Tokens used: %d (input: %d, output: %d)\n", 
			result.TokensUsed, result.InputTokens, result.OutputTokens)
	}
	
	// Debug info at the very end
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf("\n[DEBUG] Token extraction details:\n")
		fmt.Printf("  Total tokens: %d\n", result.TokensUsed)
		fmt.Printf("  Input tokens: %d\n", result.InputTokens)
		fmt.Printf("  Output tokens: %d\n", result.OutputTokens)
	}
	
	return nil
}

type DiffCmd struct {
	Range   string `arg:"" optional:"" help:"Git range (e.g., main..HEAD, HEAD~1)" default:"HEAD~1"`
	Verbose bool   `short:"v" help:"Enable verbose output"`
	Message string `short:"m" help:"Message to display while processing" default:"Analyzing changes..."`
}

func (d *DiffCmd) Run() error {
	// Initialize git client
	gitClient, err := git.NewGitClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	// Parse git range
	base, head := git.ParseGitRange(d.Range)
	
	if d.Verbose {
		fmt.Printf("Reviewing changes between %s and %s\n", base, head)
	}

	// Get changed files
	files, err := gitClient.GetChangedFiles(base, head)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files changed in the specified range.")
		return nil
	}

	if d.Verbose {
		fmt.Printf("Found %d changed files\n", len(files))
	}

	// Initialize reviewer
	reviewer, err := agents.NewCodeReviewer()
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}

	// Review each changed file
	totalTokens := 0
	for _, file := range files {
		// Skip files we don't review
		if !shouldReviewFile(file) {
			if d.Verbose {
				fmt.Printf("Skipping %s\n", file)
			}
			continue
		}

		fmt.Printf("\n=== Reviewing: %s ===\n", file)

		// Read file contents
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		// Get the diff for context (for future use)
		_, err = gitClient.GetFileDiff(base, head, file)
		if err != nil && d.Verbose {
			fmt.Printf("Warning: couldn't get diff for %s: %v\n", file, err)
		}

		// Create spinner
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " " + d.Message
		s.Start()

		// Perform review
		result, err := reviewer.Review(string(content), file)
		
		// Stop spinner
		s.Stop()
		
		if err != nil {
			fmt.Printf("Error reviewing file: %v\n", err)
			continue
		}

		fmt.Println(result.Content)
		
		if result.TokensUsed > 0 {
			totalTokens += result.TokensUsed
		}
	}

	// Summary
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Files reviewed: %d\n", len(files))
	if totalTokens > 0 {
		fmt.Printf("Total tokens used: %d\n", totalTokens)
	}

	return nil
}

func shouldReviewFile(filename string) bool {
	// Get the router to check if we have a guide for this file
	r := router.NewRouter()
	guide := r.GetGuide(filename)
	
	// Skip if no guide available
	if guide == "" {
		return false
	}

	// Skip common non-code files
	skipExtensions := []string{
		".md", ".json", ".lock", ".yaml", ".yml",
		".png", ".jpg", ".jpeg", ".gif", ".svg",
		".ico", ".pdf", ".zip", ".tar", ".gz",
	}
	
	ext := filepath.Ext(filename)
	for _, skip := range skipExtensions {
		if ext == skip {
			return false
		}
	}

	// Skip common directories
	skipDirs := []string{
		"node_modules", ".git", "dist", "build",
		"coverage", ".next", "out",
	}
	
	for _, skip := range skipDirs {
		if strings.Contains(filename, skip+"/") {
			return false
		}
	}

	return true
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("go-review"),
		kong.Description("AI-powered code review tool"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
