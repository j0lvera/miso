package agents

import (
	"os"
	"strings"
	"testing"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/git"
)

func TestNewCodeReviewer(t *testing.T) {
	// Save original env var
	originalKey := os.Getenv("OPENROUTER_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENROUTER_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	t.Run(
		"missing API key", func(t *testing.T) {
			os.Unsetenv("OPENROUTER_API_KEY")

			_, err := NewCodeReviewer()
			if err == nil {
				t.Error("Expected error when OPENROUTER_API_KEY is not set")
			}
			if !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
				t.Errorf(
					"Error should mention OPENROUTER_API_KEY, got: %v", err,
				)
			}
		},
	)

	t.Run(
		"with API key", func(t *testing.T) {
			os.Setenv("OPENROUTER_API_KEY", "test-key")

			reviewer, err := NewCodeReviewer()
			if err != nil {
				t.Errorf("Unexpected error with API key set: %v", err)
			}
			if reviewer == nil {
				t.Error("Expected non-nil reviewer")
			}
		},
	)
}

func TestCodeReviewer_Review(t *testing.T) {
	// Skip API tests to avoid costs
	t.Skip("Skipping API test to avoid costs")

	reviewer, err := NewCodeReviewer()
	if err != nil {
		t.Fatalf("Failed to create reviewer: %v", err)
	}
	cfg := config.DefaultConfig()

	tests := []struct {
		name     string
		code     string
		filename string
		wantErr  bool
	}{
		{
			name: "simple Go code",
			code: `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
			filename: "main.go",
			wantErr:  false,
		},
		{
			name:     "empty code",
			code:     "",
			filename: "empty.go",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result, err := reviewer.Review(cfg, tt.code, tt.filename)

				if (err != nil) != tt.wantErr {
					t.Errorf("Review() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if err == nil {
					if result == nil {
						t.Error("Expected non-nil result")
					}
					if len(result.Suggestions) == 0 {
						t.Error("Expected non-empty suggestions")
					}
				}
			},
		)
	}
}

func TestCodeReviewer_ReviewDiff(t *testing.T) {
	// Skip API tests to avoid costs
	t.Skip("Skipping API test to avoid costs")

	reviewer, err := NewCodeReviewer()
	if err != nil {
		t.Fatalf("Failed to create reviewer: %v", err)
	}
	cfg := config.DefaultConfig()

	tests := []struct {
		name     string
		diffData *git.DiffData
		filename string
		wantErr  bool
	}{
		{
			name: "simple diff",
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
		},
		{
			name: "empty diff",
			diffData: &git.DiffData{
				FilePath: "empty.go",
				Hunks:    []git.DiffHunk{},
			},
			filename: "empty.go",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result, err := reviewer.ReviewDiff(
					cfg, tt.diffData, tt.filename,
				)

				if (err != nil) != tt.wantErr {
					t.Errorf(
						"ReviewDiff() error = %v, wantErr %v", err, tt.wantErr,
					)
					return
				}

				if err == nil {
					if result == nil {
						t.Error("Expected non-nil result")
					}
					if len(result.Suggestions) == 0 {
						t.Error("Expected non-empty suggestions")
					}
				}
			},
		)
	}
}

func TestCodeReviewer_callLLM(t *testing.T) {
	// Skip API tests to avoid costs
	t.Skip("Skipping API test to avoid costs")

	reviewer, err := NewCodeReviewer()
	if err != nil {
		t.Fatalf("Failed to create reviewer: %v", err)
	}

	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "simple prompt",
			prompt:  "Review this code: package main",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true, // Empty prompts should cause an error
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result, err := reviewer.callLLM(tt.prompt)

				if (err != nil) != tt.wantErr {
					t.Errorf(
						"callLLM() error = %v, wantErr %v", err, tt.wantErr,
					)
					return
				}

				if err == nil {
					if result == nil {
						t.Error("Expected non-nil result")
					}
				}
			},
		)
	}
}

// Mock tests for unit testing without API calls
func TestReviewResult_Structure(t *testing.T) {
	result := &ReviewResult{
		Suggestions: []Suggestion{
			{
				Title: "Test review content",
			},
		},
		TokensUsed:   100,
		InputTokens:  60,
		OutputTokens: 40,
		Cost:         0.001,
	}

	if len(result.Suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(result.Suggestions))
	}

	if result.Suggestions[0].Title != "Test review content" {
		t.Errorf(
			"Expected content 'Test review content', got %s", result.Suggestions[0].Title,
		)
	}

	if result.TokensUsed != 100 {
		t.Errorf("Expected 100 tokens used, got %d", result.TokensUsed)
	}

	if result.InputTokens != 60 {
		t.Errorf("Expected 60 input tokens, got %d", result.InputTokens)
	}

	if result.OutputTokens != 40 {
		t.Errorf("Expected 40 output tokens, got %d", result.OutputTokens)
	}

	if result.Cost != 0.001 {
		t.Errorf("Expected cost 0.001, got %f", result.Cost)
	}
}
