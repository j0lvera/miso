package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/briandowns/spinner"
	"github.com/j0lvera/miso/internal/agents"
	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/diff"
	"github.com/j0lvera/miso/internal/git"
	"github.com/j0lvera/miso/internal/resolver"
)

var version = "dev"

type CLI struct {
	Config string `short:"c" help:"Path to config file" type:"existingfile"`

	Review         ReviewCmd         `cmd:"" help:"Review a code file"`
	Diff           DiffCmd           `cmd:"" help:"Review changes in a git diff"`
	ValidateConfig ValidateConfigCmd `cmd:"" help:"Validate configuration file"`
	TestPattern    TestPatternCmd    `cmd:"" help:"Test which patterns match a file"`
	Version        VersionCmd        `cmd:"" help:"Show version"`
}

type ReviewCmd struct {
	File    string `arg:"" required:"" help:"Path to the file to review" type:"existingfile"`
	Verbose bool   `short:"v" help:"Enable verbose output"`
	Message string `short:"m" help:"Message to display while processing" default:"Thinking..."`
	DryRun  bool   `short:"d" help:"Show what would be reviewed without calling LLM"`
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
	parser := config.NewParser()
	var cfg *config.Config
	var err error

	if cli.Config != "" {
		cfg, err = parser.LoadFile(cli.Config)
		if err != nil {
			return fmt.Errorf(
				"failed to load config file %s: %w", cli.Config, err,
			)
		}
	} else {
		cfg, err = parser.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if len(cfg.Patterns) == 0 {
			fmt.Println("Using default configuration (no config file found or config is empty)")
		}
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

func (r *ReviewCmd) Run(cli *CLI) error {
	// Load configuration
	parser := config.NewParser()
	var cfg *config.Config
	var err error

	if cli.Config != "" {
		cfg, err = parser.LoadFile(cli.Config)
		if err != nil {
			return fmt.Errorf(
				"failed to load config file %s: %w", cli.Config, err,
			)
		}
		if r.Verbose {
			fmt.Printf("Using config file: %s\n", cli.Config)
		}
	} else {
		cfg, err = parser.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if r.Verbose && len(cfg.Patterns) == 0 {
			fmt.Println("Using default configuration (no config file found or config is empty)")
		}
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
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + r.Message
	s.Start()

	// Perform review
	result, err := reviewer.Review(cfg, string(content), filename)

	// Stop spinner
	s.Stop()

	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Format the output to include diffs
	formatter := diff.NewFormatter()
	formattedContent := formatter.Format(result.Content)

	fmt.Println(formattedContent)

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
	Range   string `arg:"" optional:"" help:"Git range (e.g., main..HEAD, HEAD~1)" default:"HEAD~1"`
	Verbose bool   `short:"v" help:"Enable verbose output"`
	Message string `short:"m" help:"Message to display while processing" default:"Analyzing changes..."`
	DryRun  bool   `short:"d" help:"Show what would be reviewed without calling LLM"`
}

type ValidateConfigCmd struct {
	Config string `arg:"" help:"Path to config file to validate" type:"existingfile"`
}

type TestPatternCmd struct {
	File    string `arg:"" help:"File to test against patterns" type:"existingfile"`
	Verbose bool   `short:"v" help:"Show detailed matching info"`
}

func (d *DiffCmd) Run(cli *CLI) error {
	// Load configuration
	parser := config.NewParser()
	var cfg *config.Config
	var err error

	if cli.Config != "" {
		cfg, err = parser.LoadFile(cli.Config)
		if err != nil {
			return fmt.Errorf(
				"failed to load config file %s: %w", cli.Config, err,
			)
		}
		if d.Verbose {
			fmt.Printf("Using config file: %s\n", cli.Config)
		}
	} else {
		cfg, err = parser.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if d.Verbose && len(cfg.Patterns) == 0 {
			fmt.Println("Using default configuration (no config file found or config is empty)")
		}
	}

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

	// Filter files that should be reviewed
	res := resolver.NewResolver(cfg)
	var reviewableFiles []string
	for _, file := range files {
		if res.ShouldReview(file) {
			reviewableFiles = append(reviewableFiles, file)
		} else if d.Verbose {
			fmt.Printf("Skipping %s (no matching patterns)\n", file)
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

	// Initialize diff formatter
	formatter := diff.NewFormatter()

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
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
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

		// Format the output to include diffs
		formattedContent := formatter.Format(result.Content)

		if formattedContent != "" {
			fmt.Printf("<details>\n")
			fmt.Printf("<summary>📝 Review for <strong>%s</strong></summary>\n\n", file)
			fmt.Println(formattedContent)
			fmt.Printf("\n</details>\n")
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
