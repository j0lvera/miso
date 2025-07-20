package prompts

import (
	"fmt"
	"strings"

	"github.com/j0lvera/go-review/internal/config"
	"github.com/j0lvera/go-review/internal/git"
	"github.com/j0lvera/go-review/internal/resolver"
	"github.com/tmc/langchaingo/prompts"
)

func DiffReview(diffData *git.DiffData, filename string) (string, error) {
	// Try to load configuration
	parser := config.NewParser()
	cfg, err := parser.Load()
	if err != nil || len(cfg.Patterns) == 0 {
		// Use default legacy config for backward compatibility
		cfg = getDefaultLegacyConfig()
	}

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
			combinedGuides.WriteString(fmt.Sprintf("\n=== %s ===\n%s\n", guideName, content))
		}
	}

	// Format the diff for review
	formattedDiff := diffData.FormatForReview()

	// Analyze the changes
	addedLines := diffData.GetAddedLines()
	removedLines := diffData.GetRemovedLines()
	
	changesSummary := fmt.Sprintf("Changes Summary:\n- Added lines: %d\n- Removed lines: %d\n- Total hunks: %d", 
		len(addedLines), len(removedLines), len(diffData.Hunks))

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

**DIFF TO REVIEW:**
{{.changes_summary}}

{{.formatted_diff}}

**REPORT FORMAT:**
## Change Impact Analysis
[🔴 Breaking | 🟡 Risky | 🟢 Safe]

## Code Quality Issues
[🔴 Critical | 🟡 Warning | 💡 Suggestion]

## Consistency & Patterns
[❌ Inconsistent | ⚠️ Minor Issue | ✅ Good]

For each issue provide:
- Specific line numbers from the diff
- What's wrong and severity level
- Why it matters for this change
- How to fix (with code examples)

Focus on actionable feedback for the specific changes shown.

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
