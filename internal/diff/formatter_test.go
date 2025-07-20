package diff

import (
	"strings"
	"testing"
)

func TestFormatter_Format(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "single suggestion block",
			input: `Here is a suggestion:
- Fix:
` + "```original\n" + `uid := "mock-user-id"` + "\n```\n" + "```suggestion\n" + `uid := c.Get("user_id").(string)` + "\n```",
			wantContains: []string{
				"```diff",
				`-uid := "mock-user-id"`,
				`+uid := c.Get("user_id").(string)`,
				"```",
			},
			wantNotContain: []string{
				"```original",
				"```suggestion",
			},
		},
		{
			name: "multiple suggestion blocks",
			input: `First issue:
` + "```original\n" + `a := 1` + "\n```\n" + "```suggestion\n" + `a := 2` + "\n```" + `
Second issue:
` + "```original\n" + `b := "hello"` + "\n```\n" + "```suggestion\n" + `b := "world"` + "\n```",
			wantContains: []string{
				"-a := 1",
				"+a := 2",
				`-b := "hello"`,
				`+b := "world"`,
			},
			wantNotContain: []string{
				"```original",
				"```suggestion",
			},
		},
		{
			name:  "no suggestion blocks",
			input: "This is a review with no suggestions.",
			wantContains: []string{
				"This is a review with no suggestions.",
			},
			wantNotContain: []string{
				"```diff",
			},
		},
		{
			name: "malformed block - only original",
			input: "Here is a malformed block:\n" + "```original\n" + `a := 1` + "\n```",
			wantContains: []string{
				"```original",
				"a := 1",
			},
			wantNotContain: []string{
				"```diff",
			},
		},
		{
			name: "multiline diff",
			input: "Multiline change:\n" + "```original\n" + `line one
line two
line three` + "\n```\n" + "```suggestion\n" + `line one
line two is changed
line three` + "\n```",
			wantContains: []string{
				" line one",
				"-line two",
				"+line two is changed",
				" line three",
			},
			wantNotContain: []string{
				"```original",
				"```suggestion",
			},
		},
	}

	formatter := NewFormatter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.Format(tt.input)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("Format() output does not contain expected string.\nGot:\n%s\n\nWant to contain:\n%s", got, want)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("Format() output contains unexpected string.\nGot:\n%s\n\nShould not contain:\n%s", got, notWant)
				}
			}
		})
	}
}
