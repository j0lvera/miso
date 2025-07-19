package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/j0lvera/go-review/internal/agents"
)

var version = "dev"

type CLI struct {
	Review  ReviewCmd  `cmd:"" help:"Review a code file"`
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
	
	// Display token usage and cost if available
	if result.TokensUsed > 0 {
		fmt.Printf("\n---\n")
		fmt.Printf("Tokens used: %d", result.TokensUsed)
		if result.InputTokens > 0 || result.OutputTokens > 0 {
			fmt.Printf(" (input: %d, output: %d)", result.InputTokens, result.OutputTokens)
		}
		fmt.Println()
		
		if result.Cost > 0 {
			fmt.Printf("Cost: $%.4f\n", result.Cost)
		}
	}
	
	return nil
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
