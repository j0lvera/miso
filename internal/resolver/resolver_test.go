package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/j0lvera/miso/internal/config"
)

func TestResolver(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		ContentDefaults: config.ContentDefaults{
			Strategy: "first_lines",
			Lines:    50,
		},
		Patterns: []config.Pattern{
			{
				Name:     "test-files",
				Filename: `_test\.go$`,
				Context:  []string{"testing.md"},
				Stop:     true,
			},
			{
				Name:     "go-files",
				Filename: `\.go$`,
				Context:  []string{"go.md"},
			},
		},
	}

	resolver := NewResolver(cfg)

	tests := []struct {
		name          string
		filename      string
		shouldReview  bool
		expectedCount int
	}{
		{
			name:          "go file",
			filename:      "main.go",
			shouldReview:  true,
			expectedCount: 1,
		},
		{
			name:          "test file",
			filename:      "main_test.go",
			shouldReview:  true,
			expectedCount: 1, // Stop flag prevents multiple matches
		},
		{
			name:          "non-matching file",
			filename:      "README.md",
			shouldReview:  false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Test ShouldReview
				shouldReview := resolver.ShouldReview(tt.filename)
				if shouldReview != tt.shouldReview {
					t.Errorf(
						"ShouldReview() = %v, want %v", shouldReview,
						tt.shouldReview,
					)
				}

				// Test GetGuides
				guides, err := resolver.GetGuides(tt.filename)
				if err != nil {
					t.Fatalf("GetGuides() error = %v", err)
				}

				if len(guides) != tt.expectedCount {
					t.Errorf(
						"GetGuides() returned %d guides, want %d", len(guides),
						tt.expectedCount,
					)
				}
			},
		)
	}
}

func TestResolverWithContent(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "database/sql"

func main() {
	// Test
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &config.Config{
		ContentDefaults: config.ContentDefaults{
			Strategy: "first_lines",
			Lines:    10,
		},
		Patterns: []config.Pattern{
			{
				Name:     "go-sql",
				Filename: `\.go$`,
				Content:  `database/sql`,
				Context:  []string{"database.md"},
			},
		},
	}

	resolver := NewResolver(cfg)

	guides, err := resolver.GetGuides(testFile)
	if err != nil {
		t.Fatalf("GetGuides() error = %v", err)
	}

	if len(guides) != 1 {
		t.Errorf("Expected 1 guide, got %d", len(guides))
	}

	if len(guides) > 0 && guides[0] != "database.md" {
		t.Errorf("Expected database.md, got %s", guides[0])
	}
}
