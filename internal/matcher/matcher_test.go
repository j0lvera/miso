package matcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/j0lvera/miso/internal/config"
)

func TestMatchFile(t *testing.T) {
	cfg := &config.Config{
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
			{
				Name:     "handlers",
				Filename: `handlers/`,
				Context:  []string{"handlers.md"},
			},
		},
	}

	matcher := NewMatcher(cfg)

	tests := []struct {
		filename        string
		expectedMatches []string
	}{
		{
			filename:        "main.go",
			expectedMatches: []string{"go-files"},
		},
		{
			filename:        "main_test.go",
			expectedMatches: []string{"test-files"}, // Stop flag prevents go-files match
		},
		{
			filename:        "handlers/user.go",
			expectedMatches: []string{"go-files", "handlers"},
		},
		{
			filename:        "README.md",
			expectedMatches: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.filename, func(t *testing.T) {
				matches, err := matcher.MatchFile(tt.filename)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if len(matches) != len(tt.expectedMatches) {
					t.Errorf(
						"expected %d matches, got %d", len(tt.expectedMatches),
						len(matches),
					)
				}

				for i, match := range matches {
					if i < len(tt.expectedMatches) && match.Name != tt.expectedMatches[i] {
						t.Errorf(
							"expected match %s, got %s", tt.expectedMatches[i],
							match.Name,
						)
					}
				}
			},
		)
	}
}

func TestMatchFileContent(t *testing.T) {
	cfg := &config.Config{
		ContentDefaults: config.ContentDefaults{
			Strategy: "first_lines",
			Lines:    10,
		},
		Patterns: []config.Pattern{
			{
				Name:    "imports-react",
				Content: `import.*react|from ['"]react`,
				Context: []string{"react.md"},
			},
			{
				Name:     "go-sql",
				Filename: `\.go$`,
				Content:  `database/sql|\.Query\(|\.Exec\(`,
				Context:  []string{"database.md"},
			},
		},
	}

	matcher := NewMatcher(cfg)

	tests := []struct {
		name            string
		filename        string
		content         string
		expectedMatches []string
	}{
		{
			name:            "react import",
			filename:        "component.tsx",
			content:         "import React from 'react'\nimport { useState } from 'react'",
			expectedMatches: []string{"imports-react"},
		},
		{
			name:            "go with sql",
			filename:        "db.go",
			content:         "package db\n\nimport \"database/sql\"\n\nfunc Query() {}",
			expectedMatches: []string{"go-sql"},
		},
		{
			name:            "go without sql",
			filename:        "main.go",
			content:         "package main\n\nimport \"fmt\"\n\nfunc main() {}",
			expectedMatches: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				matches, err := matcher.MatchFileContent(
					tt.filename, []byte(tt.content),
				)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if len(matches) != len(tt.expectedMatches) {
					t.Errorf(
						"expected %d matches, got %d", len(tt.expectedMatches),
						len(matches),
					)
				}

				for i, match := range matches {
					if i < len(tt.expectedMatches) && match.Name != tt.expectedMatches[i] {
						t.Errorf(
							"expected match %s, got %s", tt.expectedMatches[i],
							match.Name,
						)
					}
				}
			},
		)
	}
}

func TestGetMatchedGuides(t *testing.T) {
	matcher := NewMatcher(&config.Config{})

	patterns := []config.Pattern{
		{
			Name:        "pattern1",
			Context:     []string{"guide1.md", "guide2.md"},
			DiffContext: []string{"diff1.md"},
		},
		{
			Name:        "pattern2",
			Context:     []string{"guide2.md", "guide3.md"},
			DiffContext: []string{"diff2.md"},
		},
	}

	// Test regular context
	guides := matcher.GetMatchedGuides(patterns, false)
	expectedGuides := map[string]bool{
		"guide1.md": true,
		"guide2.md": true,
		"guide3.md": true,
	}

	if len(guides) != len(expectedGuides) {
		t.Errorf("expected %d guides, got %d", len(expectedGuides), len(guides))
	}

	for _, guide := range guides {
		if !expectedGuides[guide] {
			t.Errorf("unexpected guide: %s", guide)
		}
	}

	// Test diff context
	diffGuides := matcher.GetMatchedGuides(patterns, true)
	if len(diffGuides) != 2 {
		t.Errorf("expected 2 diff guides, got %d", len(diffGuides))
	}
}

func TestContentScanning(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main
// Line 2
// Line 3
// Line 4
// Line 5
import "database/sql"
// Line 7
// Line 8
// Line 9
// Line 10
// Line 11
// Line 12
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg := &config.Config{
		ContentDefaults: config.ContentDefaults{
			Strategy: "first_lines",
			Lines:    5,
		},
		Patterns: []config.Pattern{
			{
				Name:            "sql-early",
				Content:         `database/sql`,
				ContentStrategy: "first_lines",
				Context:         []string{"sql.md"},
			},
		},
	}

	matcher := NewMatcher(cfg)

	// Should not match because "database/sql" is on line 6
	matches, err := matcher.ScanFile(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf(
			"expected no matches with first_lines strategy, got %d",
			len(matches),
		)
	}

	// Now test with full_file strategy
	cfg.Patterns[0].ContentStrategy = "full_file"
	matches, err = matcher.ScanFile(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf(
			"expected 1 match with full_file strategy, got %d", len(matches),
		)
	}
}
