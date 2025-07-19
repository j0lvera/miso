package agents

import (
	promps "github.com/j0lvera/go-review/internal/prompts"
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

	template := promps.CodeReview("", "file.tsx")

	return &CodeReviewer{
		llm:      llm,
		template: template,
	}, nil
}
