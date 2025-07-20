package git

import (
	"strings"
	"testing"
)

func TestParseDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffText string
		filePath string
		want     *DiffData
		wantErr  bool
	}{
		{
			name: "simple addition",
			diffText: `@@ -1,3 +1,4 @@
 line1
 line2
+added line
 line3`,
			filePath: "test.go",
			want: &DiffData{
				FilePath: "test.go",
				Hunks: []DiffHunk{
					{
						OldStart: 1,
						OldCount: 3,
						NewStart: 1,
						NewCount: 4,
						Header:   "@@ -1,3 +1,4 @@",
						Lines: []DiffLine{
							{Type: DiffLineContext, Content: "line1", OldNum: 1, NewNum: 1},
							{Type: DiffLineContext, Content: "line2", OldNum: 2, NewNum: 2},
							{Type: DiffLineAdded, Content: "added line", NewNum: 3},
							{Type: DiffLineContext, Content: "line3", OldNum: 3, NewNum: 4},
						},
					},
				},
			},
		},
		{
			name: "simple removal",
			diffText: `@@ -1,4 +1,3 @@
 line1
 line2
-removed line
 line3`,
			filePath: "test.go",
			want: &DiffData{
				FilePath: "test.go",
				Hunks: []DiffHunk{
					{
						OldStart: 1,
						OldCount: 4,
						NewStart: 1,
						NewCount: 3,
						Header:   "@@ -1,4 +1,3 @@",
						Lines: []DiffLine{
							{Type: DiffLineContext, Content: "line1", OldNum: 1, NewNum: 1},
							{Type: DiffLineContext, Content: "line2", OldNum: 2, NewNum: 2},
							{Type: DiffLineRemoved, Content: "removed line", OldNum: 3},
							{Type: DiffLineContext, Content: "line3", OldNum: 4, NewNum: 3},
						},
					},
				},
			},
		},
		{
			name:     "empty diff",
			diffText: "",
			filePath: "test.go",
			want: &DiffData{
				FilePath: "test.go",
				Hunks:    []DiffHunk{},
			},
		},
		{
			name: "multiple hunks",
			diffText: `@@ -1,2 +1,3 @@
 line1
+added1
 line2
@@ -10,2 +11,2 @@
-old line
+new line
 line11`,
			filePath: "test.go",
			want: &DiffData{
				FilePath: "test.go",
				Hunks: []DiffHunk{
					{
						OldStart: 1,
						OldCount: 2,
						NewStart: 1,
						NewCount: 3,
						Header:   "@@ -1,2 +1,3 @@",
						Lines: []DiffLine{
							{Type: DiffLineContext, Content: "line1", OldNum: 1, NewNum: 1},
							{Type: DiffLineAdded, Content: "added1", NewNum: 2},
							{Type: DiffLineContext, Content: "line2", OldNum: 2, NewNum: 3},
						},
					},
					{
						OldStart: 10,
						OldCount: 2,
						NewStart: 11,
						NewCount: 2,
						Header:   "@@ -10,2 +11,2 @@",
						Lines: []DiffLine{
							{Type: DiffLineRemoved, Content: "old line", OldNum: 10},
							{Type: DiffLineAdded, Content: "new line", NewNum: 11},
							{Type: DiffLineContext, Content: "line11", OldNum: 11, NewNum: 12},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDiff(tt.diffText, tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareDiffData(got, tt.want) {
				t.Errorf("ParseDiff() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDiffData_GetAddedLines(t *testing.T) {
	diff := &DiffData{
		Hunks: []DiffHunk{
			{
				Lines: []DiffLine{
					{Type: DiffLineContext, Content: "context"},
					{Type: DiffLineAdded, Content: "added1"},
					{Type: DiffLineRemoved, Content: "removed"},
					{Type: DiffLineAdded, Content: "added2"},
				},
			},
		},
	}

	added := diff.GetAddedLines()
	if len(added) != 2 {
		t.Errorf("Expected 2 added lines, got %d", len(added))
	}
	if added[0].Content != "added1" || added[1].Content != "added2" {
		t.Errorf("Unexpected added lines content")
	}
}

func TestDiffData_GetRemovedLines(t *testing.T) {
	diff := &DiffData{
		Hunks: []DiffHunk{
			{
				Lines: []DiffLine{
					{Type: DiffLineContext, Content: "context"},
					{Type: DiffLineAdded, Content: "added"},
					{Type: DiffLineRemoved, Content: "removed1"},
					{Type: DiffLineRemoved, Content: "removed2"},
				},
			},
		},
	}

	removed := diff.GetRemovedLines()
	if len(removed) != 2 {
		t.Errorf("Expected 2 removed lines, got %d", len(removed))
	}
	if removed[0].Content != "removed1" || removed[1].Content != "removed2" {
		t.Errorf("Unexpected removed lines content")
	}
}

func TestDiffData_FormatForReview(t *testing.T) {
	diff := &DiffData{
		FilePath: "test.go",
		IsNew:    false,
		Hunks: []DiffHunk{
			{
				Header: "@@ -1,2 +1,3 @@",
				Lines: []DiffLine{
					{Type: DiffLineContext, Content: "line1"},
					{Type: DiffLineAdded, Content: "added line"},
					{Type: DiffLineContext, Content: "line2"},
				},
			},
		},
	}

	formatted := diff.FormatForReview()
	
	// Check that it contains expected elements
	if !strings.Contains(formatted, "File: test.go") {
		t.Error("Formatted output should contain file path")
	}
	if !strings.Contains(formatted, "@@ -1,2 +1,3 @@") {
		t.Error("Formatted output should contain hunk header")
	}
	if !strings.Contains(formatted, "+added line") {
		t.Error("Formatted output should contain added line with + prefix")
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *DiffHunk
		wantErr bool
	}{
		{
			name: "valid header",
			line: "@@ -1,3 +1,4 @@",
			want: &DiffHunk{
				OldStart: 1,
				OldCount: 3,
				NewStart: 1,
				NewCount: 4,
				Header:   "@@ -1,3 +1,4 @@",
			},
		},
		{
			name: "single line change",
			line: "@@ -1 +1 @@",
			want: &DiffHunk{
				OldStart: 1,
				OldCount: 1,
				NewStart: 1,
				NewCount: 1,
				Header:   "@@ -1 +1 @@",
			},
		},
		{
			name:    "invalid header",
			line:    "not a hunk header",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHunkHeader(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHunkHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareHunk(got, tt.want) {
				t.Errorf("parseHunkHeader() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Helper functions for comparison
func compareDiffData(a, b *DiffData) bool {
	if a.FilePath != b.FilePath || a.IsNew != b.IsNew || a.IsDeleted != b.IsDeleted || a.IsRenamed != b.IsRenamed {
		return false
	}
	if len(a.Hunks) != len(b.Hunks) {
		return false
	}
	for i := range a.Hunks {
		if !compareHunk(&a.Hunks[i], &b.Hunks[i]) {
			return false
		}
	}
	return true
}

func compareHunk(a, b *DiffHunk) bool {
	if a.OldStart != b.OldStart || a.OldCount != b.OldCount || 
	   a.NewStart != b.NewStart || a.NewCount != b.NewCount || 
	   a.Header != b.Header {
		return false
	}
	if len(a.Lines) != len(b.Lines) {
		return false
	}
	for i := range a.Lines {
		if !compareLine(&a.Lines[i], &b.Lines[i]) {
			return false
		}
	}
	return true
}

func compareLine(a, b *DiffLine) bool {
	return a.Type == b.Type && a.Content == b.Content && 
		   a.OldNum == b.OldNum && a.NewNum == b.NewNum
}
