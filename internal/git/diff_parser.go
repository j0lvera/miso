package git

import (
	"fmt"
	"strconv"
	"strings"
)

// DiffData represents structured information about a file diff.
// It contains metadata about the file changes and parsed diff hunks.
type DiffData struct {
	FilePath    string      `json:"file_path"`
	OldFilePath string      `json:"old_file_path,omitempty"`
	NewFilePath string      `json:"new_file_path,omitempty"`
	IsNew       bool        `json:"is_new"`
	IsDeleted   bool        `json:"is_deleted"`
	IsRenamed   bool        `json:"is_renamed"`
	Hunks       []DiffHunk  `json:"hunks"`
}

// DiffHunk represents a contiguous section of changes in a diff.
// Each hunk contains line-by-line changes with context.
type DiffHunk struct {
	OldStart int        `json:"old_start"`
	OldCount int        `json:"old_count"`
	NewStart int        `json:"new_start"`
	NewCount int        `json:"new_count"`
	Header   string     `json:"header"`
	Lines    []DiffLine `json:"lines"`
}

// DiffLine represents a single line in a diff with its type and position.
// Lines can be added, removed, context, or metadata.
type DiffLine struct {
	Type    DiffLineType `json:"type"`
	Content string       `json:"content"`
	OldNum  int          `json:"old_num,omitempty"`
	NewNum  int          `json:"new_num,omitempty"`
}

// DiffLineType represents the type of a diff line (added, removed, context, etc.).
type DiffLineType string

const (
	DiffLineAdded     DiffLineType = "added"
	DiffLineRemoved   DiffLineType = "removed"
	DiffLineContext   DiffLineType = "context"
	DiffLineNoNewline DiffLineType = "no_newline"
)

// ParseDiff parses a unified diff string into structured DiffData.
// The input should be in standard unified diff format with @@ hunk headers.
func ParseDiff(diffText, filePath string) (*DiffData, error) {
	lines := strings.Split(diffText, "\n")
	
	diff := &DiffData{
		FilePath: filePath,
		Hunks:    []DiffHunk{},
	}
	
	var currentHunk *DiffHunk
	oldLineNum := 0
	newLineNum := 0
	
	for i, line := range lines {
		// Skip empty lines at the end
		if i == len(lines)-1 && line == "" {
			continue
		}
		
		// Parse file headers
		if strings.HasPrefix(line, "--- ") {
			diff.OldFilePath = strings.TrimPrefix(line, "--- ")
			if diff.OldFilePath == "/dev/null" {
				diff.IsNew = true
			}
			continue
		}
		
		if strings.HasPrefix(line, "+++ ") {
			diff.NewFilePath = strings.TrimPrefix(line, "+++ ")
			if diff.NewFilePath == "/dev/null" {
				diff.IsDeleted = true
			}
			continue
		}
		
		// Parse hunk headers (@@ -old_start,old_count +new_start,new_count @@)
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				diff.Hunks = append(diff.Hunks, *currentHunk)
			}
			
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse hunk header: %w", err)
			}
			
			currentHunk = hunk
			oldLineNum = hunk.OldStart
			newLineNum = hunk.NewStart
			continue
		}
		
		// Parse diff lines
		if currentHunk != nil && len(line) > 0 {
			diffLine := DiffLine{
				Content: line[1:], // Remove the +/- prefix
			}
			
			switch line[0] {
			case '+':
				diffLine.Type = DiffLineAdded
				diffLine.NewNum = newLineNum
				newLineNum++
			case '-':
				diffLine.Type = DiffLineRemoved
				diffLine.OldNum = oldLineNum
				oldLineNum++
			case ' ':
				diffLine.Type = DiffLineContext
				diffLine.OldNum = oldLineNum
				diffLine.NewNum = newLineNum
				oldLineNum++
				newLineNum++
			case '\\':
				diffLine.Type = DiffLineNoNewline
				diffLine.Content = line
			default:
				// Skip unrecognized lines
				continue
			}
			
			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}
	
	// Add the last hunk
	if currentHunk != nil {
		diff.Hunks = append(diff.Hunks, *currentHunk)
	}
	
	return diff, nil
}

// parseHunkHeader parses a hunk header line like "@@ -1,4 +1,6 @@"
func parseHunkHeader(line string) (*DiffHunk, error) {
	// Store the original line for the Header field
	originalLine := line
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") || !strings.Contains(line, "@@") {
		return nil, fmt.Errorf("invalid hunk header: %s", line)
	}
	
	// Extract the range part between @@
	parts := strings.Split(line, "@@")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid hunk header format: %s", line)
	}
	
	rangeStr := strings.TrimSpace(parts[1])
	
	// Parse old and new ranges
	ranges := strings.Split(rangeStr, " ")
	if len(ranges) != 2 {
		return nil, fmt.Errorf("invalid range format: %s", rangeStr)
	}
	
	oldRange := strings.TrimPrefix(ranges[0], "-")
	newRange := strings.TrimPrefix(ranges[1], "+")
	
	oldStart, oldCount, err := parseRange(oldRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old range: %w", err)
	}
	
	newStart, newCount, err := parseRange(newRange)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new range: %w", err)
	}
	
	return &DiffHunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
		Header:   originalLine,
		Lines:    []DiffLine{},
	}, nil
}

// parseRange parses a range like "1,4" or "1" into start and count
func parseRange(rangeStr string) (int, int, error) {
	if rangeStr == "" {
		return 0, 0, nil
	}
	
	parts := strings.Split(rangeStr, ",")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	
	count := 1 // Default count is 1 if not specified
	if len(parts) > 1 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}
	
	return start, count, nil
}

// GetAddedLines returns all lines that were added in this diff.
// Useful for analyzing what new code was introduced.
func (d *DiffData) GetAddedLines() []DiffLine {
	var added []DiffLine
	for _, hunk := range d.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == DiffLineAdded {
				added = append(added, line)
			}
		}
	}
	return added
}

// GetRemovedLines returns all lines that were removed in this diff.
// Useful for analyzing what code was deleted.
func (d *DiffData) GetRemovedLines() []DiffLine {
	var removed []DiffLine
	for _, hunk := range d.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == DiffLineRemoved {
				removed = append(removed, line)
			}
		}
	}
	return removed
}

// GetContextLines returns all unchanged context lines from the diff.
// Context lines provide surrounding code for better understanding.
func (d *DiffData) GetContextLines() []DiffLine {
	var context []DiffLine
	for _, hunk := range d.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == DiffLineContext {
				context = append(context, line)
			}
		}
	}
	return context
}

// FormatForReview formats the diff data into a human-readable string for code review.
// The output includes file status, change summary, and formatted diff hunks.
func (d *DiffData) FormatForReview() string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("File: %s\n", d.FilePath))
	
	if d.IsNew {
		result.WriteString("Status: New file\n")
	} else if d.IsDeleted {
		result.WriteString("Status: Deleted file\n")
	} else if d.IsRenamed {
		result.WriteString(fmt.Sprintf("Status: Renamed from %s\n", d.OldFilePath))
	}
	
	result.WriteString("\nChanges:\n")
	
	for _, hunk := range d.Hunks {
		result.WriteString(fmt.Sprintf("\n@@ -%d,%d +%d,%d @@", 
			hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))
		if hunk.Header != "" {
			result.WriteString(" " + hunk.Header)
		}
		result.WriteString("\n")
		
		for _, line := range hunk.Lines {
			switch line.Type {
			case DiffLineAdded:
				result.WriteString(fmt.Sprintf("+%s\n", line.Content))
			case DiffLineRemoved:
				result.WriteString(fmt.Sprintf("-%s\n", line.Content))
			case DiffLineContext:
				result.WriteString(fmt.Sprintf(" %s\n", line.Content))
			}
		}
	}
	
	return result.String()
}
