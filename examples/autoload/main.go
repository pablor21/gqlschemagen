package main

import (
	"log"

	"github.com/pablor21/gqlschemagen/generator"
)

func main() {
	// Example 1: Auto-load config from current directory or parent directories
	// Searches for: .gqlschemagen.yml, gqlschemagen.yml, or gqlschemagen.yaml
	if err := generator.GenerateFromDefaultConfig(); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	log.Println("Schema generated successfully!")
}
