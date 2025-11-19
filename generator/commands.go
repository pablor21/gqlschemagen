package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func GenerateFromDefaultConfig() error {
	return GenerateFromConfigFile("gqlschemagen.yml")
}

func GenerateFromConfigFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	return Generate(&cfg)
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
