package agents

import (
	"context"
	"fmt"
	"os"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/git"
	"github.com/j0lvera/miso/internal/prompts"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// ReviewResult holds the review content and token usage information from an LLM call.
// Provides details about the review content and associated costs.
type ReviewResult struct {
	Content      string
	TokensUsed   int
	InputTokens  int
	OutputTokens int
	Cost         float64
}

// CodeReviewer represents an AI-powered code reviewer agent.
// It uses large language models to provide intelligent code review feedback.
type CodeReviewer struct {
	llm llms.Model
}

// NewCodeReviewer creates a new CodeReviewer instance with OpenRouter configuration.
// Requires OPENROUTER_API_KEY environment variable to be set.
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
		return nil, fmt.Errorf(
			"failed to initialize OpenRouter client: %w", err,
		)
	}

	return &CodeReviewer{
		llm: llm,
	}, nil
}

// Review performs a comprehensive code review on the provided code.
// Uses configured review guides and patterns to provide contextual feedback.
func (cr *CodeReviewer) Review(
	cfg *config.Config, code string, filename string,
) (*ReviewResult, error) {
	// Get the formatted prompt
	prompt, err := prompts.CodeReview(cfg, code, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to format prompt: %w", err)
	}

	return cr.callLLM(prompt)
}

// ReviewDiff performs a focused code review on the provided diff data.
// Analyzes only the changes rather than the full file, using diff-specific guides.
func (cr *CodeReviewer) ReviewDiff(
	cfg *config.Config, diffData *git.DiffData, filename string,
) (*ReviewResult, error) {
	// Get the formatted diff prompt
	prompt, err := prompts.DiffReview(cfg, diffData, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to format diff prompt: %w", err)
	}

	return cr.callLLM(prompt)
}

// callLLM is a helper method to make LLM calls and parse responses
func (cr *CodeReviewer) callLLM(prompt string) (*ReviewResult, error) {
	// Call the LLM with GenerateContent for detailed response
	ctx := context.Background()
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := cr.llm.GenerateContent(
		ctx, messages,
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

	// Check if usage information is available in the response
	// Based on the debug output, OpenRouter returns these fields in GenerationInfo
	if len(resp.Choices) > 0 && resp.Choices[0].GenerationInfo != nil {
		genInfo := resp.Choices[0].GenerationInfo

		// Extract the actual fields from the GenerationInfo map
		// The values might be int or float64, so we need to handle both
		if completionTokens, ok := genInfo["CompletionTokens"].(int); ok {
			result.OutputTokens = completionTokens
		} else if completionTokens, ok := genInfo["CompletionTokens"].(float64); ok {
			result.OutputTokens = int(completionTokens)
		}

		if promptTokens, ok := genInfo["PromptTokens"].(int); ok {
			result.InputTokens = promptTokens
		} else if promptTokens, ok := genInfo["PromptTokens"].(float64); ok {
			result.InputTokens = int(promptTokens)
		}

		if totalTokens, ok := genInfo["TotalTokens"].(int); ok {
			result.TokensUsed = totalTokens
		} else if totalTokens, ok := genInfo["TotalTokens"].(float64); ok {
			result.TokensUsed = int(totalTokens)
		}

		// Don't print debug here - we'll do it after the review content
	}

	return result, nil
}
