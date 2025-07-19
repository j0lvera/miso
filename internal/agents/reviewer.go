package agents

import (
	"context"
	"fmt"
	"os"

	"github.com/j0lvera/go-review/internal/prompts"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// ReviewResult holds the review content and usage information
type ReviewResult struct {
	Content      string
	TokensUsed   int
	InputTokens  int
	OutputTokens int
	Cost         float64
}

// CodeReviewer is a struct that represents a code reviewer agent.
type CodeReviewer struct {
	llm llms.Model
}

// NewCodeReviewer creates a new CodeReviewer instance.
func NewCodeReviewer() (*CodeReviewer, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable is not set")
	}

	// Configure for OpenRouter
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-3.5-sonnet"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenRouter client: %w", err)
	}

	return &CodeReviewer{
		llm: llm,
	}, nil
}

// Review performs a code review on the provided code.
func (cr *CodeReviewer) Review(code string, filename string) (*ReviewResult, error) {
	// Get the formatted prompt
	prompt, err := prompts.CodeReview(code, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to format prompt: %w", err)
	}

	// Call the LLM with GenerateContent for detailed response
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}
	
	resp, err := cr.llm.GenerateContent(ctx, messages,
		llms.WithTemperature(0.3),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract the response content
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Content
	}

	// Create result with content
	result := &ReviewResult{
		Content: content,
	}

	// Check if usage information is available
	if resp.Usage != nil {
		result.InputTokens = resp.Usage.PromptTokens
		result.OutputTokens = resp.Usage.CompletionTokens
		result.TokensUsed = resp.Usage.TotalTokens
		
		// OpenRouter might include cost in the response
		if cost, ok := resp.Usage.Extensions["cost"].(float64); ok {
			result.Cost = cost
		}
	}

	// Debug logging if enabled
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf("Full LLM Response: %+v\n", resp)
		if resp.Usage != nil {
			fmt.Printf("Usage Details: %+v\n", resp.Usage)
		}
	}

	return result, nil
}
