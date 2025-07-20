package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitClient provides an interface for Git operations needed for code review.
// It wraps go-git functionality to provide diff and commit information.
type GitClient struct {
	repo *git.Repository
}

// NewGitClient creates a new GitClient instance for the current directory.
// Returns an error if the current directory is not a Git repository.
func NewGitClient() (*GitClient, error) {
	// Debug: print current working directory
	if os.Getenv("DEBUG") == "true" {
		wd, _ := os.Getwd()
		fmt.Printf("[DEBUG] Current working directory: %s\n", wd)
		
		// Check if .git exists
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Printf("[DEBUG] .git directory does not exist\n")
		} else {
			fmt.Printf("[DEBUG] .git directory found\n")
		}
	}
	
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository in current directory: %w", err)
	}
	return &GitClient{repo: repo}, nil
}

// GetChangedFiles returns a list of files that changed between two Git references.
// References can be commit hashes, branch names, or symbolic refs like HEAD.
func (g *GitClient) GetChangedFiles(baseRef, headRef string) ([]string, error) {
	// Resolve references to commits
	baseCommit, err := g.resolveCommit(baseRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base ref %s: %w", baseRef, err)
	}

	headCommit, err := g.resolveCommit(headRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head ref %s: %w", headRef, err)
	}

	// Get the diff
	patch, err := baseCommit.Patch(headCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch: %w", err)
	}

	// Extract file names
	var files []string
	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()
		if to != nil {
			files = append(files, to.Path())
		} else if from != nil {
			// File was deleted
			continue
		}
	}

	return files, nil
}

// GetFileDiff returns the raw diff text for a specific file between two references.
// The returned string contains the unified diff format suitable for review.
func (g *GitClient) GetFileDiff(baseRef, headRef, filePath string) (string, error) {
	baseCommit, err := g.resolveCommit(baseRef)
	if err != nil {
		return "", err
	}

	headCommit, err := g.resolveCommit(headRef)
	if err != nil {
		return "", err
	}

	patch, err := baseCommit.Patch(headCommit)
	if err != nil {
		return "", err
	}

	// Get the full patch as string and extract the file-specific part
	fullPatch := patch.String()
	
	// For now, return the full patch - we can parse it later if needed
	// This is sufficient for providing diff context to the LLM
	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()
		if (to != nil && to.Path() == filePath) || (from != nil && from.Path() == filePath) {
			// Return the full patch for now - contains all the diff info
			return fullPatch, nil
		}
	}

	return "", fmt.Errorf("no diff found for file: %s", filePath)
}

// GetFileDiffData returns structured diff information for a specific file.
// The returned DiffData contains parsed hunks, line changes, and metadata.
func (g *GitClient) GetFileDiffData(baseRef, headRef, filePath string) (*DiffData, error) {
	// Get the raw diff first
	rawDiff, err := g.GetFileDiff(baseRef, headRef, filePath)
	if err != nil {
		return nil, err
	}

	// Parse the raw diff into structured data
	diffData, err := ParseDiff(rawDiff, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff for %s: %w", filePath, err)
	}

	return diffData, nil
}

// resolveCommit resolves a reference string to a commit object
func (g *GitClient) resolveCommit(ref string) (*object.Commit, error) {
	// Handle HEAD specially
	if ref == "HEAD" {
		head, err := g.repo.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
		return g.repo.CommitObject(head.Hash())
	}

	// Try to resolve as a reference (branch, tag)
	reference, err := g.repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err == nil {
		return g.repo.CommitObject(reference.Hash())
	}

	// Try remote branch
	reference, err = g.repo.Reference(plumbing.ReferenceName("refs/remotes/origin/"+ref), true)
	if err == nil {
		return g.repo.CommitObject(reference.Hash())
	}

	// Try as a commit hash
	if len(ref) >= 4 { // Minimum hash length
		hash := plumbing.NewHash(ref)
		commit, err := g.repo.CommitObject(hash)
		if err == nil {
			return commit, nil
		}
	}

	// Try as a short hash
	if len(ref) >= 4 && len(ref) < 40 {
		iter, err := g.repo.CommitObjects()
		if err != nil {
			return nil, err
		}
		defer iter.Close()

		var foundCommit *object.Commit
		err = iter.ForEach(func(c *object.Commit) error {
			if strings.HasPrefix(c.Hash.String(), ref) {
				foundCommit = c
				return fmt.Errorf("found") // Break the loop
			}
			return nil
		})

		if foundCommit != nil {
			return foundCommit, nil
		}
	}

	return nil, fmt.Errorf("unable to resolve reference: %s", ref)
}

// ParseGitRange parses a Git range specification into base and head references.
// Supports formats like "main..feature", "HEAD~1", or single references.
// Returns "HEAD~1" and "HEAD" as defaults for empty input.
func ParseGitRange(rangeStr string) (base, head string) {
	if rangeStr == "" {
		return "HEAD~1", "HEAD"
	}

	// Handle .. syntax
	if strings.Contains(rangeStr, "..") {
		parts := strings.Split(rangeStr, "..")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	// Handle single ref (compare with HEAD)
	return rangeStr, "HEAD"
}
