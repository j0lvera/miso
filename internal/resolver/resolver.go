package resolver

import (
	"os"
	"path/filepath"

	"github.com/j0lvera/go-review/internal/config"
	"github.com/j0lvera/go-review/internal/matcher"
)

// Resolver handles guide resolution based on configuration patterns.
// It determines which review guides should be applied to specific files.
type Resolver struct {
	config  *config.Config
	matcher *matcher.Matcher
}

// NewResolver creates a new resolver with the given configuration.
// The resolver uses the config to match files and determine appropriate guides.
func NewResolver(cfg *config.Config) *Resolver {
	return &Resolver{
		config:  cfg,
		matcher: matcher.NewMatcher(cfg),
	}
}

// GetGuides returns the appropriate review guide files for a given filename.
// Performs pattern matching and returns the context guides for matched patterns.
func (r *Resolver) GetGuides(filename string) ([]string, error) {
	// First try to match by filename only
	filenameMatches, err := r.matcher.MatchFile(filename)
	if err != nil {
		return nil, err
	}

	// If we have filename matches and don't need content scanning, return early
	if len(filenameMatches) > 0 && !r.needsContentScan(filenameMatches) {
		return r.matcher.GetMatchedGuides(filenameMatches, false), nil
	}

	// If we need content scanning or have patterns with only content matching
	if r.hasContentPatterns() {
		// Read file content for content-based matching
		content, err := os.ReadFile(filename)
		if err != nil {
			// If we can't read the file but have filename matches, use those
			if len(filenameMatches) > 0 {
				return r.matcher.GetMatchedGuides(filenameMatches, false), nil
			}
			return nil, err
		}

		contentMatches, err := r.matcher.MatchFileContent(filename, content)
		if err != nil {
			return nil, err
		}

		return r.matcher.GetMatchedGuides(contentMatches, false), nil
	}

	// Return guides from filename matches
	return r.matcher.GetMatchedGuides(filenameMatches, false), nil
}

// GetDiffGuides returns the diff-specific guide files for a given filename.
// Uses diff_context guides which focus on change impact rather than full file review.
func (r *Resolver) GetDiffGuides(filename string) ([]string, error) {
	// For diffs, we typically don't need content scanning since we're reviewing changes
	filenameMatches, err := r.matcher.MatchFile(filename)
	if err != nil {
		return nil, err
	}

	return r.matcher.GetMatchedGuides(filenameMatches, true), nil
}

// ShouldReview returns true if the file matches any patterns and should be reviewed.
// Used to filter files before performing expensive review operations.
func (r *Resolver) ShouldReview(filename string) bool {
	guides, err := r.GetGuides(filename)
	if err != nil {
		return false
	}
	return len(guides) > 0
}

// needsContentScan checks if any matched patterns require content scanning
func (r *Resolver) needsContentScan(patterns []config.Pattern) bool {
	for _, p := range patterns {
		if p.Content != "" {
			return true
		}
	}
	return false
}

// hasContentPatterns checks if config has any content-only patterns
func (r *Resolver) hasContentPatterns() bool {
	for _, p := range r.config.Patterns {
		if p.Filename == "" && p.Content != "" {
			return true
		}
	}
	return false
}

// LoadGuideContent loads the content of guide files from disk.
// Tries multiple paths for each guide and returns a map of guide name to content.
func (r *Resolver) LoadGuideContent(guides []string) (map[string]string, error) {
	guideContent := make(map[string]string)
	
	for _, guide := range guides {
		// Try multiple paths for guides
		paths := []string{
			filepath.Join("guides", guide),
			filepath.Join("guides", "react", guide), // Legacy path support
			guide, // Direct path
		}

		var content []byte
		var err error
		found := false

		for _, path := range paths {
			content, err = os.ReadFile(path)
			if err == nil {
				found = true
				break
			}
		}

		if !found {
			// Skip guides that can't be found
			continue
		}

		guideContent[guide] = string(content)
	}

	return guideContent, nil
}
