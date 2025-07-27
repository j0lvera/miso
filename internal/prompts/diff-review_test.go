package prompts

import (
	"strings"
	"testing"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/git"
)

func TestDiffReview(t *testing.T) {
	cfg := config.DefaultConfig()
	tests := []struct {
		name     string
		diffData *git.DiffData
		filename string
		wantErr  bool
		contains []string // Strings that should be in the output
	}{
		{
			name: "simple diff with additions",
			diffData: &git.DiffData{
				FilePath: "test.go",
				Hunks: []git.DiffHunk{
					{
						OldStart: 1,
						OldCount: 2,
						NewStart: 1,
						NewCount: 3,
						Header:   "@@ -1,2 +1,3 @@",
						Lines: []git.DiffLine{
							{
								Type:    git.DiffLineContext,
								Content: "package main",
							},
							{
								Type:    git.DiffLineAdded,
								Content: "import \"fmt\"",
							},
							{
								Type:    git.DiffLineContext,
								Content: "func main() {}",
							},
						},
					},
				},
			},
			filename: "test.go",
			wantErr:  false,
			contains: []string{
				"Changes Summary:",
				"Added lines: 1",
				"Removed lines: 0",
				"Total hunks: 1",
				"File: test.go",
				"@@ -1,2 +1,3 @@",
				"+import \"fmt\"",
				"**CHANGE ANALYSIS FOCUS:**",
				"**REVIEW GUIDELINES:**",
				"**Output Format:**",
			},
		},
		{
			name: "diff with removals",
			diffData: &git.DiffData{
				FilePath: "handler.go",
				Hunks: []git.DiffHunk{
					{
						OldStart: 5,
						OldCount: 3,
						NewStart: 5,
						NewCount: 2,
						Header:   "@@ -5,3 +5,2 @@",
						Lines: []git.DiffLine{
							{
								Type:    git.DiffLineContext,
								Content: "func handler() {",
							},
							{
								Type:    git.DiffLineRemoved,
								Content: "// TODO: implement",
							},
							{Type: git.DiffLineContext, Content: "}"},
						},
					},
				},
			},
			filename: "handler.go",
			wantErr:  false,
			contains: []string{
				"Changes Summary:",
				"Added lines: 0",
				"Removed lines: 1",
				"Total hunks: 1",
				"File: handler.go",
				"-// TODO: implement",
			},
		},
		{
			name: "empty diff",
			diffData: &git.DiffData{
				FilePath: "empty.go",
				Hunks:    []git.DiffHunk{},
			},
			filename: "empty.go",
			wantErr:  false,
			contains: []string{
				"Changes Summary:",
				"Added lines: 0",
				"Removed lines: 0",
				"Total hunks: 0",
				"File: empty.go",
			},
		},
		{
			name: "multiple hunks",
			diffData: &git.DiffData{
				FilePath: "complex.go",
				Hunks: []git.DiffHunk{
					{
						Header: "@@ -1,2 +1,3 @@",
						Lines: []git.DiffLine{
							{
								Type:    git.DiffLineAdded,
								Content: "// New comment",
							},
							{
								Type:    git.DiffLineContext,
								Content: "package main",
							},
						},
					},
					{
						Header: "@@ -10,3 +11,2 @@",
						Lines: []git.DiffLine{
							{
								Type:    git.DiffLineContext,
								Content: "func test() {",
							},
							{Type: git.DiffLineRemoved, Content: "old code"},
							{Type: git.DiffLineContext, Content: "}"},
						},
					},
				},
			},
			filename: "complex.go",
			wantErr:  false,
			contains: []string{
				"Changes Summary:",
				"Added lines: 1",
				"Removed lines: 1",
				"Total hunks: 2",
				"@@ -1,2 +1,3 @@",
				"@@ -10,3 +11,2 @@",
				"+// New comment",
				"-old code",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := DiffReview(cfg, tt.diffData, tt.filename)

				if (err != nil) != tt.wantErr {
					t.Errorf(
						"DiffReview() error = %v, wantErr %v", err, tt.wantErr,
					)
					return
				}

				if err != nil {
					return // Skip content checks if we expected an error
				}

				// Check that all expected strings are present
				for _, expected := range tt.contains {
					if !strings.Contains(got, expected) {
						t.Errorf(
							"DiffReview() output missing expected string: %q",
							expected,
						)
						t.Logf("Full output:\n%s", got)
					}
				}

				// Basic structure checks
				if !strings.Contains(got, "You are an expert code reviewer") {
					t.Error("DiffReview() should contain reviewer instruction")
				}

				if !strings.Contains(got, "CHANGE ANALYSIS FOCUS") {
					t.Error("DiffReview() should contain analysis focus section")
				}
			},
		)
	}
}

func TestDiffReview_GuideIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	// Test that guides are properly integrated into the prompt
	diffData := &git.DiffData{
		FilePath: "test.page.tsx", // Use a file that matches default patterns
		Hunks: []git.DiffHunk{
			{
				Header: "@@ -1,1 +1,2 @@",
				Lines: []git.DiffLine{
					{
						Type:    git.DiffLineContext,
						Content: "import React from 'react'",
					},
					{Type: git.DiffLineAdded, Content: "// Added comment"},
				},
			},
		},
	}

	result, err := DiffReview(cfg, diffData, "test.page.tsx")
	if err != nil {
		t.Fatalf("DiffReview() failed: %v", err)
	}

	// Should contain the architecture guides section when guides are found
	// Note: This test may not always have guides depending on file patterns
	// The important thing is that it doesn't error and generates a valid prompt
	if !strings.Contains(result, "You are an expert code reviewer") {
		t.Error("DiffReview() should generate valid prompt")
	}
	if strings.Contains(result, "**Architecture Guides:**") {
		t.Error("Default config should not find any guides")
	}
}

func TestDiffReview_FallbackToRegularGuides(t *testing.T) {
	cfg := config.DefaultConfig()
	// This test verifies that the function falls back to regular guides
	// when no diff-specific guides are available
	diffData := &git.DiffData{
		FilePath: "unknown.xyz", // File that won't match any patterns
		Hunks: []git.DiffHunk{
			{
				Header: "@@ -1,1 +1,2 @@",
				Lines: []git.DiffLine{
					{Type: git.DiffLineAdded, Content: "new content"},
				},
			},
		},
	}

	result, err := DiffReview(cfg, diffData, "unknown.xyz")
	if err != nil {
		t.Fatalf("DiffReview() failed: %v", err)
	}

	// Should still generate a valid prompt even without specific guides
	if !strings.Contains(result, "You are an expert code reviewer") {
		t.Error("DiffReview() should generate valid prompt even without matching guides")
	}
}

func TestDiffReview_ChangesSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	// Test that changes summary is correctly calculated
	diffData := &git.DiffData{
		FilePath: "test.go",
		Hunks: []git.DiffHunk{
			{
				Lines: []git.DiffLine{
					{Type: git.DiffLineAdded, Content: "line1"},
					{Type: git.DiffLineAdded, Content: "line2"},
					{Type: git.DiffLineRemoved, Content: "old1"},
					{Type: git.DiffLineContext, Content: "unchanged"},
				},
			},
			{
				Lines: []git.DiffLine{
					{Type: git.DiffLineRemoved, Content: "old2"},
					{Type: git.DiffLineAdded, Content: "line3"},
				},
			},
		},
	}

	result, err := DiffReview(cfg, diffData, "test.go")
	if err != nil {
		t.Fatalf("DiffReview() failed: %v", err)
	}

	// Should correctly count added and removed lines
	if !strings.Contains(result, "Added lines: 3") {
		t.Error("Should correctly count 3 added lines")
	}
	if !strings.Contains(result, "Removed lines: 2") {
		t.Error("Should correctly count 2 removed lines")
	}
	if !strings.Contains(result, "Total hunks: 2") {
		t.Error("Should correctly count 2 hunks")
	}
}
