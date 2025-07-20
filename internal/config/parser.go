package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Parser handles configuration file parsing and validation.
// It supports multiple configuration file formats and locations.
type Parser struct {
	configPaths []string
}

// NewParser creates a new configuration parser with default search paths.
// The parser will look for configuration files in standard locations.
func NewParser() *Parser {
	return &Parser{
		configPaths: []string{
			"miso.yml",
			"miso.yaml",
			".miso.yml",
			".miso.yaml",
		},
	}
}

// Load attempts to load configuration from default locations.
// Returns the first valid configuration found, or default config if none found.
func (p *Parser) Load() (*Config, error) {
	for _, path := range p.configPaths {
		if config, err := p.LoadFile(path); err == nil {
			return config, nil
		}
	}
	
	// No config file found, return default config
	return DefaultConfig(), nil
}

// LoadFile loads and validates configuration from a specific file path.
// Returns an error if the file doesn't exist, is invalid YAML, or fails validation.
func (p *Parser) LoadFile(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := p.validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromString loads configuration from a YAML string.
// Useful for GitHub Actions or other scenarios where config is provided as a string.
func (p *Parser) LoadFromString(yamlContent string) (*Config, error) {
	config := DefaultConfig()
	if err := yaml.Unmarshal([]byte(yamlContent), config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := p.validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// validate checks if the configuration is valid
func (p *Parser) validate(config *Config) error {
	// Validate content defaults
	validStrategies := map[string]bool{
		"first_lines": true,
		"full_file":   true,
		"smart":       true,
	}
	
	if !validStrategies[config.ContentDefaults.Strategy] {
		return fmt.Errorf("invalid default strategy: %s", config.ContentDefaults.Strategy)
	}

	// Validate patterns
	for i, pattern := range config.Patterns {
		if pattern.Name == "" {
			return fmt.Errorf("pattern %d: name is required", i)
		}

		if pattern.Filename == "" && pattern.Content == "" {
			return fmt.Errorf("pattern %s: must have either filename or content regex", pattern.Name)
		}

		if pattern.ContentStrategy != "" && !validStrategies[pattern.ContentStrategy] {
			return fmt.Errorf("pattern %s: invalid content strategy: %s", pattern.Name, pattern.ContentStrategy)
		}

		if pattern.ContentStrategy == "smart" && len(pattern.ContentLines) != 3 {
			return fmt.Errorf("pattern %s: smart strategy requires exactly 3 values for content_lines", pattern.Name)
		}

		if len(pattern.Context) == 0 && len(pattern.DiffContext) == 0 {
			return fmt.Errorf("pattern %s: must have at least one context or diff_context guide", pattern.Name)
		}
	}

	return nil
}

// FindConfigFile searches for a configuration file in the current directory and parent directories.
// Returns the path to the first configuration file found, or an error if none found.
func (p *Parser) FindConfigFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search in current directory and up to root
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		for _, configName := range p.configPaths {
			configPath := filepath.Join(dir, configName)
			if _, err := os.Stat(configPath); err == nil {
				return configPath, nil
			}
		}
	}

	return "", fmt.Errorf("no config file found")
}
