package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/git"
	"github.com/j0lvera/miso/internal/prompts"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	website = "https://github.com/j0lvera/miso"
	name    = "miso"
)

// headerTransport is a custom http.RoundTripper to add headers to requests.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

// RoundTrip adds custom headers to the request before sending it.
func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	for k, v := range t.headers {
		req2.Header.Set(k, v)
	}
	return t.base.RoundTrip(req2)
}

// Suggestion represents a single review comment from the LLM.
type Suggestion struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Original   string `json:"original,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ReviewResult holds the review content and token usage information from an LLM call.
// Provides details about the review content and associated costs.
type ReviewResult struct {
	Suggestions  []Suggestion
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

	// Set custom headers for OpenRouter
	headers := map[string]string{
		"HTTP-Referer": website,
		"X-Title":      name,
	}

	// Create a custom transport to add headers
	transport := &headerTransport{
		base:    http.DefaultTransport,
		headers: headers,
	}

	// Create a custom HTTP client
	client := &http.Client{
		Transport: transport,
	}

	// Configure for OpenRouter
	llm, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-3.5-sonnet"),
		openai.WithHTTPClient(client),
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

	// Find the start of the JSON array to strip any leading text.
	startIndex := strings.Index(content, "[")
	if startIndex == -1 {
		return nil, fmt.Errorf("failed to find start of JSON array in LLM response\nRaw response:\n%s", content)
	}

	// Find the end of the JSON array
	endIndex := strings.LastIndex(content, "]")
	if endIndex == -1 {
		return nil, fmt.Errorf("failed to find end of JSON array in LLM response\nRaw response:\n%s", content)
	}

	jsonStr := content[startIndex : endIndex+1]

	var suggestions []Suggestion
	if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w\nRaw response:\n%s", err, content)
	}

	// Create result with content
	result := &ReviewResult{
		Suggestions: suggestions,
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
