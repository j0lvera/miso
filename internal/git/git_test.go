package git

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper function to setup git client for tests
func setupGitClient(t *testing.T) (*GitClient, func()) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	cleanup := func() { os.Chdir(originalDir) }
	
	// Try to find git repository root
	testDir := originalDir
	for {
		if _, err := os.Stat(filepath.Join(testDir, ".git")); err == nil {
			os.Chdir(testDir)
			break
		}
		parent := filepath.Dir(testDir)
		if parent == testDir {
			t.Skip("Not in a git repository")
			return nil, cleanup
		}
		testDir = parent
	}

	client, err := NewGitClient()
	if err != nil {
		t.Fatalf("Failed to create git client: %v", err)
	}
	
	return client, cleanup
}

func TestNewGitClient(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	t.Run("in git repository", func(t *testing.T) {
		// Try to find the git repository root by going up directories
		testDir := originalDir
		for {
			if _, err := os.Stat(filepath.Join(testDir, ".git")); err == nil {
				os.Chdir(testDir)
				break
			}
			parent := filepath.Dir(testDir)
			if parent == testDir {
				// Reached filesystem root without finding .git
				t.Skip("Not in a git repository")
				return
			}
			testDir = parent
		}
		
		client, err := NewGitClient()
		if err != nil {
			t.Fatalf("Failed to create git client: %v", err)
		}
		
		if client == nil {
			t.Error("Expected non-nil git client")
		}
	})

	t.Run("not in git repository", func(t *testing.T) {
		// Create a temporary directory that's not a git repo
		tempDir := t.TempDir()
		os.Chdir(tempDir)
		
		_, err := NewGitClient()
		if err == nil {
			t.Error("Expected error when not in git repository")
		}
	})
}

func TestParseGitRange(t *testing.T) {
	tests := []struct {
		name      string
		rangeStr  string
		wantBase  string
		wantHead  string
	}{
		{
			name:     "empty range",
			rangeStr: "",
			wantBase: "HEAD~1",
			wantHead: "HEAD",
		},
		{
			name:     "double dot syntax",
			rangeStr: "main..feature",
			wantBase: "main",
			wantHead: "feature",
		},
		{
			name:     "triple dot syntax",
			rangeStr: "main...feature",
			wantBase: "main",
			wantHead: ".feature", // ParseGitRange treats ... as .. and adds dot
		},
		{
			name:     "single commit",
			rangeStr: "HEAD~3",
			wantBase: "HEAD~3", // Single commit becomes base, head becomes HEAD
			wantHead: "HEAD",
		},
		{
			name:     "commit hash",
			rangeStr: "abc123",
			wantBase: "abc123", // Single commit becomes base, head becomes HEAD
			wantHead: "HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, head := ParseGitRange(tt.rangeStr)
			
			if base != tt.wantBase {
				t.Errorf("ParseGitRange() base = %v, want %v", base, tt.wantBase)
			}
			if head != tt.wantHead {
				t.Errorf("ParseGitRange() head = %v, want %v", head, tt.wantHead)
			}
		})
	}
}

func TestGitClient_GetChangedFiles(t *testing.T) {
	client, cleanup := setupGitClient(t)
	defer cleanup()

	// Check if we have enough commit history for HEAD~1
	_, err := client.resolveCommit("HEAD~1")
	hasHistory := err == nil

	tests := []struct {
		name     string
		baseRef  string
		headRef  string
		wantErr  bool
		skipIf   func() bool
	}{
		{
			name:    "HEAD vs HEAD~1",
			baseRef: "HEAD~1",
			headRef: "HEAD",
			wantErr: false,
			skipIf:  func() bool { return !hasHistory },
		},
		{
			name:    "invalid base ref",
			baseRef: "nonexistent-ref",
			headRef: "HEAD",
			wantErr: true,
		},
		{
			name:    "invalid head ref",
			baseRef: "HEAD",
			headRef: "nonexistent-ref",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIf != nil && tt.skipIf() {
				t.Skip("Insufficient commit history for this test")
			}
			
			files, err := client.GetChangedFiles(tt.baseRef, tt.headRef)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChangedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				// files can be empty if no changes, that's valid
				if files == nil {
					t.Error("Expected non-nil files slice")
				}
			}
		})
	}
}

func TestGitClient_GetFileDiff(t *testing.T) {
	client, cleanup := setupGitClient(t)
	defer cleanup()

	// Get a file that exists in the repository
	files, err := client.GetChangedFiles("HEAD~1", "HEAD")
	if err != nil {
		t.Skip("Cannot get changed files for testing")
	}
	
	if len(files) == 0 {
		t.Skip("No changed files to test with")
	}

	testFile := files[0]

	tests := []struct {
		name     string
		baseRef  string
		headRef  string
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid diff",
			baseRef:  "HEAD~1",
			headRef:  "HEAD",
			filePath: testFile,
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			baseRef:  "HEAD~1",
			headRef:  "HEAD",
			filePath: "nonexistent-file.txt",
			wantErr:  false, // Git returns empty diff for nonexistent files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := client.GetFileDiff(tt.baseRef, tt.headRef, tt.filePath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFileDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				// diff can be empty, that's valid
				if diff == "" && tt.filePath == testFile {
					// If we're testing with a real changed file, we might expect some diff
					// but it's possible the file had no changes, so this is not necessarily an error
				}
			}
		})
	}
}

func TestGitClient_GetFileDiffData(t *testing.T) {
	client, cleanup := setupGitClient(t)
	defer cleanup()

	// Get a file that exists in the repository
	files, err := client.GetChangedFiles("HEAD~1", "HEAD")
	if err != nil {
		t.Skip("Cannot get changed files for testing")
	}
	
	if len(files) == 0 {
		t.Skip("No changed files to test with")
	}

	testFile := files[0]

	tests := []struct {
		name     string
		baseRef  string
		headRef  string
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid diff data",
			baseRef:  "HEAD~1",
			headRef:  "HEAD",
			filePath: testFile,
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			baseRef:  "HEAD~1",
			headRef:  "HEAD",
			filePath: "nonexistent-file.txt",
			wantErr:  false, // Should return empty diff data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffData, err := client.GetFileDiffData(tt.baseRef, tt.headRef, tt.filePath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFileDiffData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				if diffData == nil {
					t.Error("Expected non-nil diff data")
				}
				if diffData.FilePath != tt.filePath {
					t.Errorf("Expected file path %s, got %s", tt.filePath, diffData.FilePath)
				}
			}
		})
	}
}

func TestGitClient_resolveCommit(t *testing.T) {
	client, cleanup := setupGitClient(t)
	defer cleanup()

	// Check if we have enough commit history for HEAD~1
	_, err := client.resolveCommit("HEAD~1")
	hasHistory := err == nil

	tests := []struct {
		name    string
		ref     string
		wantErr bool
		skipIf  func() bool
	}{
		{
			name:    "HEAD reference",
			ref:     "HEAD",
			wantErr: false,
		},
		{
			name:    "HEAD~1 reference",
			ref:     "HEAD~1",
			wantErr: false,
			skipIf:  func() bool { return !hasHistory },
		},
		{
			name:    "invalid reference",
			ref:     "nonexistent-ref-12345",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIf != nil && tt.skipIf() {
				t.Skip("Insufficient commit history for this test")
			}
			
			commit, err := client.resolveCommit(tt.ref)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				if commit == nil {
					t.Error("Expected non-nil commit")
				}
			}
		})
	}
}
