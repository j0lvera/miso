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
		`You are an expert code reviewer. Perform a two-pass review on the provided code.

**FIRST PASS - General Code Health**
Identify general issues based on the following criteria:
- **General Issues & Idiomatic Patterns:** Check for correctness, clarity, and adherence to the language's idiomatic style.
- **Performance:** Look for potential performance bottlenecks or inefficient code.
- **Security:** Identify potential security vulnerabilities.
- **Code Smells:** Find any indicators of deeper problems in the code design.

**SECOND PASS - Architecture Compliance**
Review the code against the provided Architecture Guides. If no guides are provided, skip this pass.

**Output Format:**
Return your review as a JSON array of suggestion objects.
- Provide only actionable suggestions for improvement. Do not comment on code that is already good.
- Sort the suggestions in the final JSON array from most critical to least critical.

Each object must have the following fields:
- "id": A unique identifier for the suggestion (e.g., "miso-1A", "miso-1B").
- "title": A concise, one-line summary of the issue, including a severity emoji (e.g., "🔴 Critical", "🟡 Warning", "💡 Suggestion", "❌ Violation", "⚠️ Deviation").
- "body": A detailed explanation of the issue in markdown format. This should explain what's wrong and why it matters.
- "original": (Optional) The exact code to be replaced.
- "suggestion": (Optional) The new code.

The "body", "original", and "suggestion" fields must be valid JSON strings, meaning all newlines inside them must be escaped as \\n.

**Example JSON Output:**
[
  {
    "id": "miso-1A",
    "title": "🔴 Critical: Lack of Error Handling",
    "body": "The function `+"`doSomething`"+` can return an error that is not being checked. This could lead to unexpected behavior.",
    "original": "result := doSomething()",
    "suggestion": "result, err := doSomething()\\nif err != nil {\\n  return err\\n}"
  }
]

If you find no issues, return an empty JSON array: [].
Do not add any introductory text or markdown formatting around the JSON array.

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
