package agents

import (
	"context"
	"fmt"

	"github.com/j0lvera/go-review/internal/prompts"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CodeReviewer is a struct that represents a code reviewer agent.
type CodeReviewer struct {
	llm llms.Model
}

// NewCodeReviewer creates a new CodeReviewer instance.
func NewCodeReviewer() (*CodeReviewer, error) {
	llm, err := openai.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenAI client: %w", err)
	}

	return &CodeReviewer{
		llm: llm,
	}, nil
}

// Review performs a code review on the provided code.
func (cr *CodeReviewer) Review(code string, filename string) (string, error) {
	// Get the formatted prompt
	prompt, err := prompts.CodeReview(code, filename)
	if err != nil {
		return "", fmt.Errorf("failed to format prompt: %w", err)
	}

	// Call the LLM
	ctx := context.Background()
	response, err := cr.llm.Call(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return response, nil
}
