package matcher

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/j0lvera/miso/internal/config"
)

// Matcher handles pattern matching for files and content based on configuration rules.
// It caches compiled regular expressions for performance and supports multiple matching strategies.
type Matcher struct {
	config          *config.Config
	compiledRegexes map[string]*regexp.Regexp // Cache compiled regexes
}

// NewMatcher creates a new matcher with the given configuration.
// The matcher will use the provided config to determine matching rules and strategies.
func NewMatcher(cfg *config.Config) *Matcher {
	return &Matcher{
		config:          cfg,
		compiledRegexes: make(map[string]*regexp.Regexp),
	}
}

// MatchFile determines which patterns match a given filename.
// Only evaluates filename-based patterns, not content patterns.
func (m *Matcher) MatchFile(filename string) ([]config.Pattern, error) {
	var matchedPatterns []config.Pattern

	for _, pattern := range m.config.Patterns {
		// Only check filename patterns here (content patterns handled separately)
		if pattern.Filename != "" {
			regex, err := m.getRegex(pattern.Name+"_filename", pattern.Filename)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid filename regex for pattern %s: %w", pattern.Name,
					err,
				)
			}

			if regex.MatchString(filename) {
				matchedPatterns = append(matchedPatterns, pattern)
				if pattern.Stop {
					break
				}
			}
		}
	}

	return matchedPatterns, nil
}

// MatchFileContent determines which patterns match based on both filename and file content.
// Evaluates all patterns and applies the appropriate content scanning strategy for each.
func (m *Matcher) MatchFileContent(
	filename string, content []byte,
) ([]config.Pattern, error) {
	var matchedPatterns []config.Pattern

	// First, get filename matches
	filenameMatches, err := m.MatchFile(filename)
	if err != nil {
		return nil, err
	}

	// Convert filename matches to a map for quick lookup
	filenameMatchMap := make(map[string]bool)
	for _, p := range filenameMatches {
		filenameMatchMap[p.Name] = true
	}

	// Now check all patterns
	for _, pattern := range m.config.Patterns {
		matched := false

		// If pattern has both filename and content, both must match
		if pattern.Filename != "" && pattern.Content != "" {
			if filenameMatchMap[pattern.Name] {
				// Filename matched, now check content
				contentToScan := m.getContentToScan(content, pattern)
				regex, err := m.getRegex(
					pattern.Name+"_content", pattern.Content,
				)
				if err != nil {
					return nil, fmt.Errorf(
						"invalid content regex for pattern %s: %w",
						pattern.Name, err,
					)
				}
				if regex.Match(contentToScan) {
					matched = true
				}
			}
		} else if pattern.Filename != "" && pattern.Content == "" {
			// Only filename pattern, already matched
			if filenameMatchMap[pattern.Name] {
				matched = true
			}
		} else if pattern.Filename == "" && pattern.Content != "" {
			// Only content pattern
			contentToScan := m.getContentToScan(content, pattern)
			regex, err := m.getRegex(pattern.Name+"_content", pattern.Content)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid content regex for pattern %s: %w", pattern.Name,
					err,
				)
			}
			if regex.Match(contentToScan) {
				matched = true
			}
		}

		if matched {
			matchedPatterns = append(matchedPatterns, pattern)
			if pattern.Stop {
				break
			}
		}
	}

	return matchedPatterns, nil
}

// getContentToScan returns the portion of content to scan based on strategy
func (m *Matcher) getContentToScan(
	content []byte, pattern config.Pattern,
) []byte {
	strategy := pattern.ContentStrategy
	if strategy == "" {
		strategy = m.config.ContentDefaults.Strategy
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	switch strategy {
	case "full_file":
		return content

	case "smart":
		// Get line counts for smart strategy
		var firstLines, lastLines, randomLines int
		if len(pattern.ContentLines) == 3 {
			firstLines = pattern.ContentLines[0]
			lastLines = pattern.ContentLines[1]
			randomLines = pattern.ContentLines[2]
		} else {
			// Default smart values
			firstLines = 100
			lastLines = 100
			randomLines = 100
		}

		var selectedLines []string

		// Add first lines
		for i := 0; i < firstLines && i < totalLines; i++ {
			selectedLines = append(selectedLines, lines[i])
		}

		// Add last lines
		startLast := totalLines - lastLines
		if startLast < firstLines {
			startLast = firstLines
		}
		for i := startLast; i < totalLines; i++ {
			selectedLines = append(selectedLines, lines[i])
		}

		// Add random lines from the middle
		if totalLines > firstLines+lastLines {
			middleStart := firstLines
			middleEnd := totalLines - lastLines
			for i := 0; i < randomLines && middleStart < middleEnd; i++ {
				randomIdx := middleStart + rand.Intn(middleEnd-middleStart)
				selectedLines = append(selectedLines, lines[randomIdx])
			}
		}

		return []byte(strings.Join(selectedLines, "\n"))

	default: // "first_lines"
		linesToScan := m.config.ContentDefaults.Lines
		if pattern.ContentStrategy == "first_lines" && len(pattern.ContentLines) > 0 {
			linesToScan = pattern.ContentLines[0]
		}

		if linesToScan > totalLines {
			return content
		}

		selectedLines := lines[:linesToScan]
		return []byte(strings.Join(selectedLines, "\n"))
	}
}

// GetMatchedGuides returns the appropriate guide files for the given matched patterns.
// Uses diff_context guides if isDiff is true, otherwise uses regular context guides.
func (m *Matcher) GetMatchedGuides(
	patterns []config.Pattern, isDiff bool,
) []string {
	guideMap := make(map[string]bool)
	var guides []string

	for _, pattern := range patterns {
		var patternGuides []string
		if isDiff && len(pattern.DiffContext) > 0 {
			patternGuides = pattern.DiffContext
		} else {
			patternGuides = pattern.Context
		}

		for _, guide := range patternGuides {
			if !guideMap[guide] {
				guideMap[guide] = true
				guides = append(guides, guide)
			}
		}
	}

	return guides
}

// getRegex returns a cached compiled regex or compiles and caches a new one
func (m *Matcher) getRegex(key, pattern string) (*regexp.Regexp, error) {
	if regex, exists := m.compiledRegexes[key]; exists {
		return regex, nil
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	m.compiledRegexes[key] = regex
	return regex, nil
}

// ScanFile reads a file from disk and returns which patterns match.
// Convenience method that combines file reading with pattern matching.
func (m *Matcher) ScanFile(filename string) ([]config.Pattern, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return m.MatchFileContent(filename, content)
}

// ScanFileLines reads a file line by line up to maxLines for memory efficiency.
// Useful for large files where full content scanning would be expensive.
func (m *Matcher) ScanFileLines(
	filename string, maxLines int,
) ([]config.Pattern, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && lineCount < maxLines {
		lines = append(lines, scanner.Text())
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filename, err)
	}

	content := []byte(strings.Join(lines, "\n"))
	return m.MatchFileContent(filename, content)
}
