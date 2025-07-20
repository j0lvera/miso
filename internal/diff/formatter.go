package diff

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Formatter processes text to replace code blocks with diffs.
type Formatter struct {
	dmp *diffmatchpatch.DiffMatchPatch
	re  *regexp.Regexp
}

// NewFormatter creates a new Formatter.
func NewFormatter() *Formatter {
	// Regex to find pairs of ```original and ```suggestion blocks.
	// The (?s) flag allows . to match newlines.
	re := regexp.MustCompile("(?s)```original\n(.*?)\n```\n```suggestion\n(.*?)\n```")
	return &Formatter{
		dmp: diffmatchpatch.New(),
		re:  re,
	}
}

// Format takes a string (e.g., LLM output) and replaces paired
// "original" and "suggestion" code blocks with a unified diff.
func (f *Formatter) Format(input string) string {
	return f.re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := f.re.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}

		original := submatches[1]
		suggestion := submatches[2]

		// Perform a line-based diff for cleaner output
		chars1, chars2, lineArray := f.dmp.DiffLinesToChars(original, suggestion)
		diffs := f.dmp.DiffMain(chars1, chars2, false)
		lineDiffs := f.dmp.DiffCharsToLines(diffs, lineArray)

		// Build a human-readable diff string
		var builder strings.Builder
		for _, diff := range lineDiffs {
			text := strings.TrimSuffix(diff.Text, "\n")
			lines := strings.Split(text, "\n")

			for _, line := range lines {
				switch diff.Type {
				case diffmatchpatch.DiffInsert:
					builder.WriteString(fmt.Sprintf("+%s\n", line))
				case diffmatchpatch.DiffDelete:
					builder.WriteString(fmt.Sprintf("-%s\n", line))
				case diffmatchpatch.DiffEqual:
					builder.WriteString(fmt.Sprintf(" %s\n", line))
				}
			}
		}
		unifiedDiff := strings.TrimSuffix(builder.String(), "\n")

		return fmt.Sprintf("```diff\n%s\n```", unifiedDiff)
	})
}
