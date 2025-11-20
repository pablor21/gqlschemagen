package main

import (
	"log"

	"github.com/pablor21/gqlschemagen/generator"
)

func main() {
	// Example: Load config programmatically
	cfg, err := generator.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Customize config programmatically if needed
	cfg.UseGqlGenDirectives = true
	cfg.FieldCase = generator.FieldCaseCamel

	// Generate with custom config
	if err := generator.Generate(cfg); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	log.Println("Schema generated successfully!")
}
