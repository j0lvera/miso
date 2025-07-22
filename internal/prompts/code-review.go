package prompts

import (
	"fmt"
	"strings"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/resolver"
	"github.com/tmc/langchaingo/prompts"
)

func CodeReview(cfg *config.Config, code string, filename string) (
	string, error,
) {
	// Use resolver
	res := resolver.NewResolver(cfg)
	guides, err := res.GetGuides(filename)
	if err != nil {
		return "", fmt.Errorf("failed to get guides: %w", err)
	}

	// Load guide content
	guideContent, err := res.LoadGuideContent(guides)
	if err != nil {
		return "", fmt.Errorf("failed to load guide content: %w", err)
	}

	// Combine all guide content
	var combinedGuides strings.Builder
	if len(guideContent) > 0 {
		combinedGuides.WriteString("\n\n**Architecture Guides:**\n")
		for guideName, content := range guideContent {
			combinedGuides.WriteString(
				fmt.Sprintf(
					"\n=== %s ===\n%s\n", guideName, content,
				),
			)
		}
	}

	template := prompts.NewPromptTemplate(
		`You are an expert code reviewer. Perform a two-pass review on the provided code. Do not add any introductory text, just the review.

**FIRST PASS - General Code Health**
Identify general issues based on the following criteria:
- **General Issues & Idiomatic Patterns:** Check for correctness, clarity, and adherence to the language's idiomatic style.
- **Performance:** Look for potential performance bottlenecks or inefficient code.
- **Security:** Identify potential security vulnerabilities.
- **Code Smells:** Find any indicators of deeper problems in the code design.

**SECOND PASS - Architecture Compliance**
Review the code against the provided Architecture Guides. If no guides are provided, skip this pass.

**Report Format:**
# 🍲 miso Code review

## First Pass: General Issues
[🔴 Critical | 🟡 Warning | 💡 Suggestion]

## Second Pass: Architecture Violations
[❌ Violation | ⚠️ Deviation]

For each issue provide:
- What's wrong and severity
- Why it matters
- How to fix. Use this specific format for the fix, with the original code and your suggested change:
`+"```original\n"+`[the exact code to be replaced]`+"\n```\n"+"```suggestion\n"+`[the new code]`+"\n```"+`

Keep it concise - actionable issues only.

Code to review:
'''
{{.code}}
'''

File: {{.filename}}{{.guide}}`,
		[]string{"code", "filename", "guide"},
	)

	// Format the template with the provided values
	return template.Format(
		map[string]any{
			"code":     code,
			"filename": filename,
			"guide":    combinedGuides.String(),
		},
	)
}
