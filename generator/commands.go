package generator

import (
	"fmt"
	"path/filepath"
)

// GenerateFromDefaultConfig searches for a config file in the current directory
// and parent directories, then generates the schema. If no config file is found,
// it uses default configuration values.
func GenerateFromDefaultConfig() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	return Generate(cfg)
}

// GenerateFromConfigFile loads a specific config file and generates the schema
func GenerateFromConfigFile(configPath string) error {
	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		return err
	}
	return Generate(cfg)
}

// Generate runs the schema generation with the provided configuration
func Generate(cfg *Config) error {
	// Normalize configuration
	cfg.Normalize()

	// Set default output based on strategy if not specified
	if cfg.Output == "" {
		if cfg.GenStrategy == GenStrategySingle {
			cfg.Output = "graph/schema/gqlschemagen.graphqls"
		} else {
			cfg.Output = "graph/schema"
		}
	}

	// Clean output path
	cfg.Output = filepath.Clean(cfg.Output)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}

	// Parse all packages
	parser := NewParser()
	for _, pkgPath := range cfg.Packages {
		if err := parser.Walk(PkgDir(pkgPath)); err != nil {
			return fmt.Errorf("parse error for package %s: %w", pkgPath, err)
		}
	}

	// Match enum constants after all packages are parsed (supports cross-package enums)
	parser.MatchEnumConstants()

	// Generate schema
	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		return fmt.Errorf("generation error: %w", err)
	}

	return nil
}
