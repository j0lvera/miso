package config

// Config represents the complete go-review configuration structure.
// It defines how files are matched and which review guides are applied.
type Config struct {
	ContentDefaults ContentDefaults `yaml:"content_defaults"`
	Patterns        []Pattern       `yaml:"patterns"`
}

// ContentDefaults defines global defaults for content scanning strategies.
// These settings apply when patterns don't specify their own content strategy.
type ContentDefaults struct {
	Strategy string `yaml:"strategy"` // first_lines, full_file, smart
	Lines    int    `yaml:"lines"`    // For first_lines strategy
}

// Pattern defines a file matching rule and associated review guides.
// Patterns are evaluated in order and can match based on filename, content, or both.
type Pattern struct {
	Name            string   `yaml:"name"`
	Filename        string   `yaml:"filename"`         // Regex for filename matching
	Content         string   `yaml:"content"`          // Regex for content matching
	ContentStrategy string   `yaml:"content_strategy"` // Override default strategy
	ContentLines    []int    `yaml:"content_lines"`    // For smart strategy: [first, last, random]
	Context         []string `yaml:"context"`          // Guide files to use
	DiffContext     []string `yaml:"diff_context"`     // Guide files for diff reviews
	Stop            bool     `yaml:"stop"`             // Stop evaluating further patterns
}

// DefaultConfig returns a configuration with sensible defaults.
// Used when no configuration file is found or as a base for new configurations.
func DefaultConfig() *Config {
	return &Config{
		ContentDefaults: ContentDefaults{
			Strategy: "first_lines",
			Lines:    50,
		},
		Patterns: []Pattern{},
	}
}
