package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/glamour"
	"github.com/j0lvera/miso/internal/agents"
	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/diff"
	"github.com/j0lvera/miso/internal/git"
	misoGithub "github.com/j0lvera/miso/internal/github"
	"github.com/j0lvera/miso/internal/resolver"
)

var version = "0.4.0"

const (
	spinnerCharSet     = 14
	spinnerRefreshRate = 100 * time.Millisecond
)

type CLI struct {
	Config string `short:"c" help:"Path to config file" type:"existingfile"`

	Review         ReviewCmd         `cmd:"" help:"Review a code file"`
	Diff           DiffCmd           `cmd:"" help:"Review changes in a git diff"`
	ValidateConfig ValidateConfigCmd `cmd:"" help:"Validate configuration file"`
	TestPattern    TestPatternCmd    `cmd:"" help:"Test which patterns match a file"`
	GitHub         GitHubCmd         `cmd:"" name:"github" help:"GitHub integration commands"`
	Version        VersionCmd        `cmd:"" help:"Show version"`
}

type ReviewCmd struct {
	File        string `arg:"" required:"" help:"Path to the file to review" type:"existingfile"`
	Verbose     bool   `short:"v" help:"Enable verbose output"`
	Message     string `short:"m" help:"Message to display while processing" default:"Thinking..."`
	DryRun      bool   `short:"d" help:"Show what would be reviewed without calling LLM"`
	OutputStyle string `short:"s" name:"output-style" help:"Output style: plain (default) or rich (formatted with colors and markdown)" enum:"plain,rich" default:"plain"`
	One         bool   `short:"1" name:"one" help:"Show only the first suggestion."`
}

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("miso version %s\n", version)
	return nil
}

func (vc *ValidateConfigCmd) Run(cli *CLI) error {
	configPath := vc.Config
	if configPath == "" && cli.Config != "" {
		configPath = cli.Config
	}

	parser := config.NewParser()
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = parser.LoadFile(configPath)
		fmt.Printf("Validating config file: %s\n", configPath)
	} else {
		cfg, err = parser.Load()
		if err != nil {
			fmt.Println("No config file found, using defaults")
			return nil
		}

		// Find which config file was loaded
		foundPath, findErr := parser.FindConfigFile()
		if findErr == nil {
			fmt.Printf("Validating config file: %s\n", foundPath)
		} else {
			fmt.Println("Validating default configuration")
		}
	}

	if err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}

	// Validate patterns
	issues := validatePatterns(cfg.Patterns)

	if len(issues) == 0 {
		fmt.Printf("✅ Configuration is valid!\n")
		fmt.Printf("   - Content strategy: %s\n", cfg.ContentDefaults.Strategy)
		fmt.Printf("   - Default lines: %d\n", cfg.ContentDefaults.Lines)
		fmt.Printf("   - Patterns defined: %d\n", len(cfg.Patterns))
	} else {
		fmt.Printf("⚠️  Configuration has issues:\n")
		for _, issue := range issues {
			fmt.Printf("   - %s\n", issue)
		}
		return fmt.Errorf("configuration validation failed")
	}

	return nil
}

func (tp *TestPatternCmd) Run(cli *CLI) error {
	cfg, err := loadConfig(cli.Config, tp.Verbose)
	if err != nil {
		return err
	}

	// Create resolver and test the file
	res := resolver.NewResolver(cfg)

	fmt.Printf("Testing file: %s\n", tp.File)
	fmt.Printf("Configuration: %d patterns defined\n\n", len(cfg.Patterns))

	// Test if file should be reviewed
	shouldReview := res.ShouldReview(tp.File)
	fmt.Printf("Should review: %t\n", shouldReview)

	if !shouldReview {
		fmt.Println("No patterns matched this file.")
		return nil
	}

	// Get matched guides for regular review
	guides, err := res.GetGuides(tp.File)
	if err != nil {
		return fmt.Errorf("failed to get guides: %w", err)
	}

	fmt.Printf("\nRegular review guides: %v\n", guides)

	// Get matched guides for diff review
	diffGuides, err := res.GetDiffGuides(tp.File)
	if err != nil {
		return fmt.Errorf("failed to get diff guides: %w", err)
	}

	fmt.Printf("Diff review guides: %v\n", diffGuides)

	if tp.Verbose {
		fmt.Printf("\nDetailed pattern matching:\n")
		// We need to expose pattern matching details - let's add this functionality
		showDetailedMatching(cfg, tp.File)
	}

	return nil
}

func buildSuggestionBody(suggestion agents.Suggestion) string {
	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(strings.ReplaceAll(suggestion.Body, "\\n", "\n"))

	if suggestion.Original != "" || suggestion.Suggestion != "" {
		bodyBuilder.WriteString("\n\n")
		bodyBuilder.WriteString("```original\n")
		bodyBuilder.WriteString(strings.ReplaceAll(suggestion.Original, "\\n", "\n"))
		bodyBuilder.WriteString("\n```\n")
		bodyBuilder.WriteString("```suggestion\n")
		bodyBuilder.WriteString(strings.ReplaceAll(suggestion.Suggestion, "\\n", "\n"))
		bodyBuilder.WriteString("\n```")
	}
	return bodyBuilder.String()
}

func formatSuggestionsToMarkdown(suggestions []agents.Suggestion, filename string) string {
	if len(suggestions) == 0 {
		return "✅ No issues found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# 🍲 miso Code review for %s\n\n", filename))

	formatter := diff.NewFormatter()
	for _, suggestion := range suggestions {
		fullBody := buildSuggestionBody(suggestion)
		// Format the body to render diffs correctly
		formattedBody := formatter.Format(fullBody)
		builder.WriteString(fmt.Sprintf("## %s\n%s\n\n", suggestion.Title, formattedBody))
	}

	return builder.String()
}

func (r *ReviewCmd) Run(cli *CLI) error {
	// Load configuration
	cfg, err := loadConfig(cli.Config, r.Verbose)
	if err != nil {
		return err
	}

	// Check if file should be reviewed
	res := resolver.NewResolver(cfg)
	if !res.ShouldReview(r.File) {
		fmt.Printf("File %s does not match any review patterns.\n", r.File)
		return nil
	}

	// Get guides for this file
	guides, err := res.GetGuides(r.File)
	if err != nil {
		return fmt.Errorf("failed to get guides: %w", err)
	}

	if r.Verbose {
		fmt.Printf("Reviewing file: %s\n", r.File)
		fmt.Printf("Using guides: %v\n", guides)
	}

	// Dry run mode
	if r.DryRun {
		fmt.Printf("=== DRY RUN MODE ===\n")
		fmt.Printf("File: %s\n", r.File)
		fmt.Printf("Would use guides: %v\n", guides)
		fmt.Printf("Review would be performed with these settings.\n")
		return nil
	}

	// Read file contents
	content, err := os.ReadFile(r.File)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", r.File, err)
	}

	// Initialize reviewer
	reviewer, err := agents.NewCodeReviewer()
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}

	// Get just the filename for the review
	filename := filepath.Base(r.File)

	// Create and start spinner
	s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerRefreshRate)
	s.Suffix = " " + r.Message
	s.Start()

	// Perform review
	result, err := reviewer.Review(cfg, string(content), filename)

	// Stop spinner
	s.Stop()

	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	if r.One && len(result.Suggestions) > 0 {
		result.Suggestions = result.Suggestions[:1]
	}

	markdownReport := formatSuggestionsToMarkdown(result.Suggestions, filename)

	// Apply glamour rendering if requested
	if r.OutputStyle == "rich" && len(result.Suggestions) > 0 {
		rendered, err := renderRichOutput(markdownReport)
		if err != nil {
			log.Printf("Failed to initialize rich renderer: %v", err)
			fmt.Println(markdownReport) // Fallback to plain
		} else {
			fmt.Print(rendered)
		}
	} else {
		fmt.Println(markdownReport)
	}

	// Display token usage if available
	if result.TokensUsed > 0 {
		fmt.Printf("\n---\n")
		fmt.Printf(
			"Tokens used: %d (input: %d, output: %d)\n",
			result.TokensUsed, result.InputTokens, result.OutputTokens,
		)
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
	Args        []string `arg:"" optional:"" name:"range-or-file" help:"[range] [file]. Git range to review (defaults to 'main..HEAD'). Optionally, a file path to review within the range."`
	Verbose     bool     `short:"v" help:"Enable verbose output"`
	Message     string   `short:"m" help:"Message to display while processing" default:"Analyzing changes..."`
	DryRun      bool     `short:"d" help:"Show what would be reviewed without calling LLM"`
	One         bool     `short:"1" name:"one" help:"Show only the first suggestion per file."`
	OutputStyle string   `short:"s" name:"output-style" help:"Output style: plain (default) or rich (formatted with colors and markdown)" enum:"plain,rich" default:"plain"`
}

type ValidateConfigCmd struct {
	Config string `arg:"" optional:"" help:"Path to config file to validate" type:"existingfile"`
}

type TestPatternCmd struct {
	File    string `arg:"" help:"File to test against patterns" type:"existingfile"`
	Verbose bool   `short:"v" help:"Show detailed matching info"`
}

type GitHubCmd struct {
	ReviewPR GitHubReviewPRCmd `cmd:"" help:"Review a PR and post a comment."`
}

// GitHubReviewPRCmd reviews a pull request.
// The fields PR, Base, and Head are intentionally not marked as 'required'
// because they are designed to be auto-detected from the GitHub Actions environment.
// The validation logic is handled within the Run method after attempting auto-detection.
type GitHubReviewPRCmd struct {
	PR      int    `short:"p" help:"Pull request number (auto-detected in GitHub Actions)."`
	Base    string `short:"b" help:"Base commit SHA (auto-detected in GitHub Actions)."`
	Head    string `short:"H" help:"Head commit SHA (auto-detected in GitHub Actions)."`
	Verbose bool   `short:"v" help:"Enable verbose output."`
	Message string `short:"m" help:"Message to display while processing." default:"Analyzing PR..."`
}

func isValidSHA(sha string) bool {
	// A git SHA is typically 40 hex characters, but can be shorter (e.g. 7).
	// We'll check for 4-40 hex characters.
	match, _ := regexp.MatchString(`^[a-f0-9]{4,40}$`, sha)
	return match
}

func (gr *GitHubReviewPRCmd) validate() error {
	if gr.PR < 0 {
		return fmt.Errorf("invalid PR number: %d", gr.PR)
	}
	if gr.Base != "" && !isValidSHA(gr.Base) {
		return fmt.Errorf("invalid base SHA: %s", gr.Base)
	}
	if gr.Head != "" && !isValidSHA(gr.Head) {
		return fmt.Errorf("invalid head SHA: %s", gr.Head)
	}
	return nil
}

func (gr *GitHubReviewPRCmd) Run(cli *CLI) error {
	if err := gr.validate(); err != nil {
		return err
	}
	// Load configuration
	cfg, err := loadConfig(cli.Config, gr.Verbose)
	if err != nil {
		return err
	}

	ghClient, err := misoGithub.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client (check GITHUB_TOKEN and GITHUB_REPOSITORY env vars): %w", err)
	}

	// Auto-detect PR info if not provided
	base, head := gr.Base, gr.Head
	prNumber := gr.PR

	if prNumber == 0 || base == "" || head == "" {
		if event, err := ghClient.GetPRInfo(); err == nil {
			if prNumber == 0 {
				prNumber = event.PullRequest.Number
			}
			if base == "" {
				base = event.PullRequest.Base.SHA
			}
			if head == "" {
				head = event.PullRequest.Head.SHA
			}
		}
	}

	if base == "" || head == "" {
		return fmt.Errorf("could not determine base and head commits. Please specify --base and --head, or run in a GitHub Action context")
	}
	if prNumber == 0 {
		return fmt.Errorf("could not determine pull request number. Please specify --pr, or run in a GitHub Action context")
	}

	if gr.Verbose {
		fmt.Printf("Reviewing PR #%d: %s..%s\n", prNumber, base, head)
	}

	// Initialize git client
	gitClient, err := git.NewGitClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
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

	if gr.Verbose {
		fmt.Printf("Found %d changed files\n", len(files))
	}

	// Filter files that should be reviewed
	res := resolver.NewResolver(cfg)
	var reviewableFiles []string
	for _, file := range files {
		if res.ShouldReview(file) {
			reviewableFiles = append(reviewableFiles, file)
		} else if gr.Verbose {
			fmt.Printf("Skipping %s (no matching patterns)\n", file)
		}
	}

	if len(reviewableFiles) == 0 {
		fmt.Println("No files match review patterns.")
		return nil
	}

	// Initialize reviewer
	reviewer, err := agents.NewCodeReviewer()
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}

	// Capture review output
	var reviewOutput bytes.Buffer

	// Review each changed file
	totalTokens := 0
	formatter := diff.NewFormatter()
	for _, file := range reviewableFiles {
		// Get guides for this file
		guides, err := res.GetDiffGuides(file)
		if err != nil {
			fmt.Printf("Error getting guides for file: %v\n", err)
			continue
		}

		if gr.Verbose {
			fmt.Printf("Using diff guides: %v\n", guides)
		}

		// Get the structured diff data
		diffData, err := gitClient.GetFileDiffData(base, head, file)
		if err != nil {
			fmt.Printf("Error getting diff for file: %v\n", err)
			continue
		}

		// Create spinner
		s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerRefreshRate)
		s.Suffix = " " + gr.Message
		s.Start()

		// Perform diff review (reviewing only the changes)
		result, err := reviewer.ReviewDiff(cfg, diffData, file)

		// Stop spinner
		s.Stop()

		if err != nil {
			fmt.Printf("Error reviewing file: %v\n", err)
			continue
		}

		if len(result.Suggestions) > 0 {
			reviewOutput.WriteString(fmt.Sprintf("<details>\n"))
			reviewOutput.WriteString(
				fmt.Sprintf(
					"<summary>📝 Review for <strong>%s</strong> (%d issues)</summary>\n\n", file, len(result.Suggestions),
				),
			)
			for _, suggestion := range result.Suggestions {
				fullBody := buildSuggestionBody(suggestion)
				formattedBody := formatter.Format(fullBody)
				reviewOutput.WriteString(fmt.Sprintf("### %s\n%s\n\n", suggestion.Title, formattedBody))
			}
			reviewOutput.WriteString("</details>\n")
		}

		if result.TokensUsed > 0 {
			totalTokens += result.TokensUsed
		}
	}

	// Post to GitHub
	var commentBody string
	if reviewOutput.Len() > 0 {
		commentBody = fmt.Sprintf("# 🍲 miso Code review\n\n%s", reviewOutput.String())
	} else {
		commentBody = "# 🍲 miso Code review\n\n✅ No issues found."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ghClient.PostOrUpdateComment(ctx, prNumber, commentBody); err != nil {
		return fmt.Errorf("failed to post comment to GitHub (PR #%d): %w", prNumber, err)
	}
	fmt.Printf("✅ Successfully posted review to PR #%d\n", prNumber)

	// Clean up old comments
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	if err := ghClient.CleanupOldComments(cleanupCtx, prNumber); err != nil {
		// This is not a fatal error, so just log it.
		if gr.Verbose {
			log.Printf("Failed to clean up old comments: %v", err)
		}
	}

	// Summary for verbose mode
	if gr.Verbose {
		log.Printf("Review completed: Files=%d, Tokens=%d, PR=#%d\n",
			len(reviewableFiles), totalTokens, prNumber)
	}

	return nil
}

func (d *DiffCmd) Run(cli *CLI) error {
	// Load configuration
	cfg, err := loadConfig(cli.Config, d.Verbose)
	if err != nil {
		return err
	}

	// Initialize git client
	gitClient, err := git.NewGitClient()
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	var rangeStr string
	var targetFile string

	switch len(d.Args) {
	case 0:
		rangeStr = "main..HEAD"
	case 1:
		// Could be a range or a file.
		if _, err := os.Stat(d.Args[0]); err == nil {
			targetFile = d.Args[0]
			rangeStr = "main..HEAD"
		} else {
			rangeStr = d.Args[0]
		}
	case 2:
		rangeStr = d.Args[0]
		targetFile = d.Args[1]
	default:
		return fmt.Errorf("too many arguments for diff command, expected [range] [file]")
	}

	// Parse git range
	base, head := git.ParseGitRange(rangeStr)

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

	// Filter files that should be reviewed
	res := resolver.NewResolver(cfg)
	var reviewableFiles []string

	if targetFile != "" {
		fileIsChanged := false
		for _, f := range files {
			if f == targetFile {
				fileIsChanged = true
				break
			}
		}

		if fileIsChanged {
			if res.ShouldReview(targetFile) {
				reviewableFiles = append(reviewableFiles, targetFile)
			} else if d.Verbose {
				fmt.Printf("Skipping %s (no matching patterns)\n", targetFile)
			}
		} else {
			fmt.Printf("File '%s' was not changed in the specified range.\n", targetFile)
			return nil
		}
	} else {
		for _, file := range files {
			if res.ShouldReview(file) {
				reviewableFiles = append(reviewableFiles, file)
			} else if d.Verbose {
				fmt.Printf("Skipping %s (no matching patterns)\n", file)
			}
		}
	}

	if len(reviewableFiles) == 0 {
		fmt.Println("No files match review patterns.")
		return nil
	}

	// Dry run mode
	if d.DryRun {
		fmt.Printf("=== DRY RUN MODE ===\n")
		fmt.Printf("Range: %s..%s\n", base, head)
		fmt.Printf("Files that would be reviewed:\n")
		for _, file := range reviewableFiles {
			guides, _ := res.GetDiffGuides(file)
			fmt.Printf("  - %s (guides: %v)\n", file, guides)
		}
		return nil
	}

	// Initialize reviewer
	reviewer, err := agents.NewCodeReviewer()
	if err != nil {
		return fmt.Errorf("failed to create reviewer: %w", err)
	}

	// Review each changed file
	totalTokens := 0
	for _, file := range reviewableFiles {
		// Get guides for this file
		guides, err := res.GetDiffGuides(file)
		if err != nil {
			fmt.Printf("Error getting guides for file: %v\n", err)
			continue
		}

		if d.Verbose {
			fmt.Printf("Using diff guides: %v\n", guides)
		}

		// Get the structured diff data
		diffData, err := gitClient.GetFileDiffData(base, head, file)
		if err != nil {
			fmt.Printf("Error getting diff for file: %v\n", err)
			continue
		}

		// Create spinner
		s := spinner.New(spinner.CharSets[spinnerCharSet], spinnerRefreshRate)
		s.Suffix = " " + d.Message
		s.Start()

		// Perform diff review (reviewing only the changes)
		result, err := reviewer.ReviewDiff(cfg, diffData, file)

		// Stop spinner
		s.Stop()

		if err != nil {
			fmt.Printf("Error reviewing file: %v\n", err)
			continue
		}

		if d.One && len(result.Suggestions) > 0 {
			result.Suggestions = result.Suggestions[:1]
		}

		markdownReport := formatSuggestionsToMarkdown(result.Suggestions, file)

		// Apply glamour rendering if requested
		if d.OutputStyle == "rich" && len(result.Suggestions) > 0 {
			rendered, err := renderRichOutput(markdownReport)
			if err != nil {
				log.Printf("Failed to initialize rich renderer: %v", err)
				fmt.Println(markdownReport) // Fallback to plain
			} else {
				fmt.Print(rendered)
			}
		} else {
			fmt.Println(markdownReport)
		}

		if result.TokensUsed > 0 {
			totalTokens += result.TokensUsed
		}
	}

	// Summary for verbose mode
	if d.Verbose {
		fmt.Printf("\n=== Summary ===\n")
		fmt.Printf("Files reviewed: %d\n", len(reviewableFiles))
		if totalTokens > 0 {
			fmt.Printf("Total tokens used: %d\n", totalTokens)
		}
	}

	return nil
}

func validatePatterns(patterns []config.Pattern) []string {
	var issues []string

	for i, pattern := range patterns {
		// Check if pattern has at least one matching criteria
		if pattern.Filename == "" && pattern.Content == "" {
			issues = append(
				issues, fmt.Sprintf(
					"Pattern %d (%s): no filename or content pattern defined",
					i+1, pattern.Name,
				),
			)
		}

		// Validate regex patterns
		if pattern.Filename != "" {
			if _, err := regexp.Compile(pattern.Filename); err != nil {
				issues = append(
					issues, fmt.Sprintf(
						"Pattern %d (%s): invalid filename regex: %v", i+1,
						pattern.Name, err,
					),
				)
			}
		}

		if pattern.Content != "" {
			if _, err := regexp.Compile(pattern.Content); err != nil {
				issues = append(
					issues, fmt.Sprintf(
						"Pattern %d (%s): invalid content regex: %v", i+1,
						pattern.Name, err,
					),
				)
			}
		}

		// Check if guide files exist
		allGuides := append(pattern.Context, pattern.DiffContext...)
		for _, guide := range allGuides {
			if !checkGuideExists(guide) {
				issues = append(
					issues, fmt.Sprintf(
						"Pattern %d (%s): guide file not found: %s", i+1,
						pattern.Name, guide,
					),
				)
			}
		}

		// Validate content strategy
		if pattern.ContentStrategy != "" {
			validStrategies := map[string]bool{
				"first_lines": true,
				"full_file":   true,
				"smart":       true,
			}
			if !validStrategies[pattern.ContentStrategy] {
				issues = append(
					issues, fmt.Sprintf(
						"Pattern %d (%s): invalid content strategy: %s", i+1,
						pattern.Name, pattern.ContentStrategy,
					),
				)
			}
		}
	}

	return issues
}

func checkGuideExists(guidePath string) bool {
	// Try multiple paths for guides
	paths := []string{
		filepath.Join("guides", guidePath),
		filepath.Join("guides", "react", guidePath), // Legacy path support
		guidePath, // Direct path
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func showDetailedMatching(cfg *config.Config, filename string) {
	// Test filename patterns
	fmt.Printf("\nFilename pattern matching:\n")
	for _, pattern := range cfg.Patterns {
		if pattern.Filename != "" {
			regex, err := regexp.Compile(pattern.Filename)
			if err != nil {
				fmt.Printf("  ❌ %s: invalid regex (%v)\n", pattern.Name, err)
				continue
			}

			if regex.MatchString(filename) {
				fmt.Printf(
					"  ✅ %s: matches filename pattern '%s'\n", pattern.Name,
					pattern.Filename,
				)
			} else {
				fmt.Printf(
					"  ❌ %s: no match for pattern '%s'\n", pattern.Name,
					pattern.Filename,
				)
			}
		}
	}

	// Test content patterns (if file exists and is readable)
	if content, err := os.ReadFile(filename); err == nil {
		fmt.Printf("\nContent pattern matching:\n")
		for _, pattern := range cfg.Patterns {
			if pattern.Content != "" {
				regex, err := regexp.Compile(pattern.Content)
				if err != nil {
					fmt.Printf(
						"  ❌ %s: invalid content regex (%v)\n", pattern.Name,
						err,
					)
					continue
				}

				// Get content to scan based on strategy
				contentToScan := getContentToScan(
					content, pattern, cfg.ContentDefaults,
				)

				if regex.Match(contentToScan) {
					fmt.Printf(
						"  ✅ %s: matches content pattern '%s'\n", pattern.Name,
						pattern.Content,
					)
				} else {
					fmt.Printf(
						"  ❌ %s: no match for content pattern '%s'\n",
						pattern.Name, pattern.Content,
					)
				}
			}
		}
	} else {
		fmt.Printf(
			"\nContent pattern matching: skipped (cannot read file: %v)\n", err,
		)
	}
}

func getContentToScan(
	content []byte, pattern config.Pattern, defaults config.ContentDefaults,
) []byte {
	strategy := pattern.ContentStrategy
	if strategy == "" {
		strategy = defaults.Strategy
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	switch strategy {
	case "full_file":
		return content
	case "smart":
		// Implementation similar to matcher package
		var firstLines, lastLines int
		if len(pattern.ContentLines) >= 3 {
			firstLines = pattern.ContentLines[0]
			lastLines = pattern.ContentLines[1]
		} else {
			firstLines = defaults.Lines
			lastLines = defaults.Lines
		}

		var selectedLines []string

		// Add first lines
		for i := 0; i < firstLines && i < totalLines; i++ {
			selectedLines = append(selectedLines, lines[i])
		}

		// Add last lines
		start := totalLines - lastLines
		if start < firstLines {
			start = firstLines
		}
		for i := start; i < totalLines; i++ {
			selectedLines = append(selectedLines, lines[i])
		}

		return []byte(strings.Join(selectedLines, "\n"))
	default: // first_lines
		linesToScan := defaults.Lines
		if len(pattern.ContentLines) > 0 {
			linesToScan = pattern.ContentLines[0]
		}

		if linesToScan >= totalLines {
			return content
		}

		return []byte(strings.Join(lines[:linesToScan], "\n"))
	}
}

func renderRichOutput(content string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create renderer: %w", err)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return "", fmt.Errorf("failed to render content: %w", err)
	}

	return rendered, nil
}

func loadConfig(configPath string, verbose bool) (*config.Config, error) {
	parser := config.NewParser()
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = parser.LoadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load config file %s: %w", configPath, err,
			)
		}
		if verbose {
			fmt.Printf("Using config file: %s\n", configPath)
		}
	} else {
		cfg, err = parser.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
		if verbose && len(cfg.Patterns) == 0 {
			fmt.Println("Using default configuration (no config file found or config is empty)")
		}
	}
	return cfg, nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(
		&cli,
		kong.Name("miso"),
		kong.Description("AI-powered code review tool"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
