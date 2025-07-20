package prompts

import (
	"fmt"
	"strings"

	"github.com/j0lvera/go-review/internal/config"
	"github.com/j0lvera/go-review/internal/resolver"
	"github.com/tmc/langchaingo/prompts"
)

func CodeReview(cfg *config.Config, code string, filename string) (string, error) {
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
			combinedGuides.WriteString(fmt.Sprintf("\n=== %s ===\n%s\n", guideName, content))
		}
	}

	template := prompts.NewPromptTemplate(
		`You are an expert Frontend code reviewer specializing in Enterprise React applications. Perform a two-pass review:

**FIRST PASS - React Best Practices**
Identify general issues unrelated to our architecture guide:
- Runtime errors, null safety, type issues
- Performance problems (unnecessary re-renders, large bundles)
- Security vulnerabilities (XSS, sensitive data exposure)
- React anti-patterns (direct DOM manipulation, improper hooks usage)

**SECOND PASS - Architecture Compliance**  
Review against our React Architecture guide:
- ✓ Page structure matches the "Page body example"
- ✓ Exports follow the PostsPage/Page pattern
- ✓ Business logic (CRUD operations) in Page component
- ✓ Data fetching uses recommended patterns
- ✓ Modal state management with ts-pattern

**Report Format:**
## First Pass: General Issues
[🔴 Critical | 🟡 Warning | 💡 Suggestion]

## Second Pass: Architecture Violations
[❌ Violation | ⚠️ Deviation]

For each issue provide:
- What's wrong and severity
- Why it matters
- How to fix (with code)

Keep it concise - actionable issues only.

Code to review:
'''tsx
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

