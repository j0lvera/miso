package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromString(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid config",
			yaml: `
content_defaults:
  strategy: "first_lines"
  lines: 50
patterns:
  - name: "go-files"
    filename: "\\.go$"
    context:
      - go.md
`,
			wantErr: false,
		},
		{
			name: "invalid strategy",
			yaml: `
content_defaults:
  strategy: "invalid"
patterns:
  - name: "test"
    filename: "\\.go$"
    context:
      - go.md
`,
			wantErr: true,
		},
		{
			name: "missing pattern name",
			yaml: `
patterns:
  - filename: "\\.go$"
    context:
      - go.md
`,
			wantErr: true,
		},
		{
			name: "no filename or content",
			yaml: `
patterns:
  - name: "test"
    context:
      - go.md
`,
			wantErr: true,
		},
		{
			name: "smart strategy with wrong lines count",
			yaml: `
patterns:
  - name: "test"
    filename: "\\.go$"
    content_strategy: "smart"
    content_lines: [100, 100]
    context:
      - go.md
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.LoadFromString(tt.yaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	
	// Create a test config file
	configPath := filepath.Join(tmpDir, "go-review.yml")
	configContent := `
content_defaults:
  strategy: "first_lines"
  lines: 30
patterns:
  - name: "test-files"
    filename: "_test\\.go$"
    context:
      - testing.md
    stop: true
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	parser := NewParser()
	config, err := parser.LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	// Verify loaded config
	if config.ContentDefaults.Strategy != "first_lines" {
		t.Errorf("Expected strategy 'first_lines', got %s", config.ContentDefaults.Strategy)
	}
	
	if config.ContentDefaults.Lines != 30 {
		t.Errorf("Expected lines 30, got %d", config.ContentDefaults.Lines)
	}
	
	if len(config.Patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(config.Patterns))
	}
	
	if config.Patterns[0].Stop != true {
		t.Errorf("Expected stop flag to be true")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.ContentDefaults.Strategy != "first_lines" {
		t.Errorf("Expected default strategy 'first_lines', got %s", config.ContentDefaults.Strategy)
	}
	
	if config.ContentDefaults.Lines != 50 {
		t.Errorf("Expected default lines 50, got %d", config.ContentDefaults.Lines)
	}
}
