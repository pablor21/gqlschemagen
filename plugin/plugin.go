// Package plugin provides a gqlgen plugin that automatically generates GraphQL schemas
// from Go structs with special annotations.
//
// This plugin scans your Go code for structs and generates corresponding GraphQL types,
// inputs, and mutations based on struct tags and directives. It supports field name
// transformations, custom directives, and automatic input generation.
//
// # Installation
//
//	go get github.com/pablor21/gqlschemagen/plugin
//
// # Usage
//
// gqlgen plugins must be registered programmatically in a custom code generation entrypoint.
// Create a file named generate.go in your project root:
//
//	//go:build ignore
//
//	package main
//
//	import (
//		"github.com/99designs/gqlgen/api"
//		"github.com/99designs/gqlgen/codegen/config"
//		"github.com/pablor21/gqlschemagen/plugin"
//	)
//
//	func main() {
//		cfg, _ := config.LoadConfigFromDefaultLocations()
//
//		// Load configuration from gqlschemagen.yml (recommended)
//		p, err := plugin.LoadConfigFromFile("gqlschemagen.yml")
//		if err != nil {
//			panic(err)
//		}
//
//		// Or configure programmatically
//		// p := plugin.New()
//		// p.Packages = []string{"./graph/models"}
//
//		// Generate with the plugin
//		if err := api.Generate(cfg, api.AddPlugin(p)); err != nil {
//			panic(err)
//		}
//	}
//
// Then run: go run generate.go
//
// # Configuration
//
// Create a gqlschemagen.yml file (separate from gqlgen.yml to avoid parsing conflicts):
//
//	packages:
//	  - ./graph/models
//	generator:
//	  strategy: single
//	  output: graph/schema/generated.graphql
//	  field_case: camel
//	  use_json_tag: true
//
// For detailed documentation and examples, see:
// https://github.com/pablor21/gqlschemagen/tree/master/plugin
package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"gopkg.in/yaml.v3"

	"github.com/pablor21/gqlschemagen/generator"
)

var _ plugin.Plugin = &Plugin{}

// Plugin implements the gqlgen plugin interface
type Plugin struct {
	cfg *generator.Config
	// Paths to scan for Go structs
	Packages []string `yaml:"packages"`
}

// YAMLConfig represents the configuration structure in gqlschemagen.yml
type YAMLConfig struct {
	// Packages to scan for Go structs
	Packages []string `yaml:"packages"`
	// Generator configuration
	Generator *generator.Config `yaml:"generator,omitempty"`
}

// New creates a new instance of the plugin with default configuration
func New() *Plugin {
	cfg := generator.NewConfig()
	cfg.GenStrategy = generator.GenStrategySingle
	cfg.Output = "graph/schema/gqlschemagen.graphqls"
	return &Plugin{
		cfg: cfg,
	}
}

// LoadConfigFromFile loads plugin configuration from a YAML file (typically gqlschemagen.yml)
// If the file doesn't exist or can't be read, it returns a plugin with default configuration
func LoadConfigFromFile(filename string) (*Plugin, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default config
			return New(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlCfg YAMLConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Use provided generator config or create default
	var cfg *generator.Config
	if yamlCfg.Generator != nil {
		cfg = yamlCfg.Generator
		cfg.Normalize()
	} else {
		cfg = generator.NewConfig()
		cfg.GenStrategy = generator.GenStrategySingle
		cfg.Output = "graph/schema/gqlschemagen.graphqls"
	}

	return &Plugin{
		cfg:      cfg,
		Packages: yamlCfg.Packages,
	}, nil
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "gqlschemagen"
}

// MutateConfig is called before code generation
// This is where we generate the GraphQL schemas from Go structs
func (p *Plugin) MutateConfig(cfg *config.Config) error {
	// If no packages specified, skip
	if len(p.Packages) == 0 {
		return nil
	}

	// Generate schemas for each package
	for _, pkgPath := range p.Packages {
		if err := p.generateSchema(pkgPath); err != nil {
			return fmt.Errorf("failed to generate schema for package %s: %w", pkgPath, err)
		}
	}

	return nil
}

// GenerateSchemas generates GraphQL schemas from configured packages
// This can be called standalone before gqlgen runs
func (p *Plugin) GenerateSchemas() error {
	return p.MutateConfig(nil)
}

// generateSchema generates GraphQL schema from a Go package
func (p *Plugin) generateSchema(pkgPath string) error {
	// Clone config and set input
	genCfg := *p.cfg
	genCfg.Packages = []string{pkgPath}
	genCfg.Output = filepath.Clean(genCfg.Output)

	// Validate configuration
	if err := genCfg.Validate(); err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}

	// Parse Go package
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(pkgPath)); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Generate schema
	engine := generator.NewGenerator(parser, &genCfg)
	if err := engine.Run(); err != nil {
		return fmt.Errorf("generation error: %w", err)
	}

	fmt.Printf("Generated schema from %s to %s\n", pkgPath, genCfg.Output)
	return nil
}
