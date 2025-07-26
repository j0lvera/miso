package prompts

import (
	"fmt"
	"strings"

	"github.com/j0lvera/miso/internal/config"
	"github.com/j0lvera/miso/internal/git"
	"github.com/j0lvera/miso/internal/resolver"
	"github.com/tmc/langchaingo/prompts"
)

func DiffReview(
	cfg *config.Config, diffData *git.DiffData, filename string,
) (string, error) {
	// Use resolver to get diff-specific guides
	res := resolver.NewResolver(cfg)
	guides, err := res.GetDiffGuides(filename)
	if err != nil {
		return "", fmt.Errorf("failed to get diff guides: %w", err)
	}

	// Fallback to regular guides if no diff-specific guides
	if len(guides) == 0 {
		guides, err = res.GetGuides(filename)
		if err != nil {
			return "", fmt.Errorf("failed to get guides: %w", err)
		}
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

	// Format the diff for review
	formattedDiff := diffData.FormatForReview()

	// Analyze the changes
	addedLines := diffData.GetAddedLines()
	removedLines := diffData.GetRemovedLines()

	changesSummary := fmt.Sprintf(
		"Changes Summary:\n- Added lines: %d\n- Removed lines: %d\n- Total hunks: %d",
		len(addedLines), len(removedLines), len(diffData.Hunks),
	)

	template := prompts.NewPromptTemplate(
		`You are an expert code reviewer analyzing specific changes in a pull request. Focus on reviewing ONLY the changes shown in the diff, not the entire file.

**CHANGE ANALYSIS FOCUS:**
1. **Breaking Changes**: Does removing code break existing functionality?
2. **New Code Quality**: Are the additions following best practices?
3. **Consistency**: Do changes match the existing codebase patterns?
4. **Security**: Do changes introduce security vulnerabilities?
5. **Performance**: Do changes impact performance negatively?

**REVIEW GUIDELINES:**
- Focus on the specific lines being added (+) and removed (-)
- Consider the context around changes (unchanged lines)
- Flag potential breaking changes from removals
- Ensure new code follows established patterns
- Check for proper error handling in new code
- Verify imports and dependencies are appropriate

**Output Format:**
Return your review as a JSON array of suggestion objects. Each object must have the following fields:
- "id": A unique identifier for the suggestion (e.g., "miso-1A", "miso-1B").
- "title": A concise, one-line summary of the issue, including a severity emoji (e.g., "🔴 Breaking", "🟡 Risky", "🟢 Safe", "🔴 Critical", "🟡 Warning", "💡 Suggestion", "❌ Inconsistent", "⚠️ Minor Issue").
- "body": A detailed explanation of the issue in markdown format. The body must explain what's wrong, why it matters, and how to fix it. For code fixes, use this specific format:
`+"```original\n"+`[the exact code to be replaced]`+"\n```\n"+"```suggestion\n"+`[the new code]`+"\n```"+`

**Example JSON Output:**
[
  {
    "id": "miso-1A",
    "title": "🔴 Breaking: Function signature changed",
    "body": "The signature of `+"`calculateTotal`"+` was changed, which will break existing callers.\n\n`+"```original\n"+`-func calculateTotal(price int, quantity int)`+"\n```\n"+"```suggestion\n"+`+func calculateTotal(price float64, quantity int)`+"\n```"+`"
  }
]

If you find no issues, return an empty JSON array: [].
Do not add any introductory text or markdown formatting around the JSON array.

**DIFF TO REVIEW:**
{{.changes_summary}}

{{.formatted_diff}}

File: {{.filename}}{{.guide}}`,
		[]string{"changes_summary", "formatted_diff", "filename", "guide"},
	)

	// Format the template with the provided values
	return template.Format(
		map[string]any{
			"changes_summary": changesSummary,
			"formatted_diff":  formattedDiff,
			"filename":        filename,
			"guide":           combinedGuides.String(),
		},
	)
}
