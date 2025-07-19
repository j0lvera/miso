package agents

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

// CodeReviewer is a struct that represents a code reviewer agent.
type CodeReviewer struct {
	llm      llms.Model
	template *prompts.PromptTemplate
}

func NewCodeReviewer() (*CodeReviewer, error) {
	llm, err := openai.New()
	if err != nil {
		return nil, err
	}

	template := prompts.NewPromptTemplate(
		`
	You are an expert Go code reviewer. Analyze this Go code for:

	1. **Bugs and Logic Issues**: Potential runtime errors, nil pointer dereferences, race conditions
	2. **Performance**: Inefficient algorithms, unnecessary allocations, string concatenation issues
	3. **Style**: Go idioms, naming conventions, error handling patterns
	4. **Security**: Input validation, sensitive data handling

	Code to review:
	'''go
	{{.code}}
	'''

	File: {{.filename}}

	Provide specific, actionable feedback. For each issue:
	- Explain WHY it's a problem
	- Show HOW to fix it with code examples
	- Rate severity: Critical, Warning, Suggestion

	Focus on the most important issues first.`,
		[]string{"code", "filename"},
	)

	return &CodeReviewer{
		llm:      llm,
		template: &template,
	}, nil
}
