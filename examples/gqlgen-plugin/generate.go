//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pablor21/gqlschemagen/plugin"
)

func main() {
	// FIRST: Load plugin config and generate schemas
	// This must happen BEFORE loading gqlgen config because
	// gqlgen validates schemas during config loading

	// Option 1: Load configuration from gqlschemagen.yml (recommended)
	p, err := plugin.LoadConfigFromFile("gqlschemagen.yml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load gqlschemagen config:", err.Error())
		os.Exit(2)
	}

	// Option 2: Programmatic configuration (if no gqlschemagen.yml)
	// p := plugin.New()
	// p.Packages = []string{"./graph/models"}

	// Generate GraphQL schemas from Go structs
	if err := p.GenerateSchemas(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to generate schemas", err.Error())
		os.Exit(2)
	}

	// NOW: Load gqlgen config (schemas are now available)
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}

	// Generate gqlgen code
	err = api.Generate(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
