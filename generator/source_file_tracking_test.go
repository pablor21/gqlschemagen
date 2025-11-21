package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceFileTracking(t *testing.T) {
	tmpDir := t.TempDir()

	// Create models directory with source files
	modelsDir := filepath.Join(tmpDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("Failed to create models directory: %v", err)
	}

	// Write user.go
	userFile := filepath.Join(modelsDir, "user.go")
	userCode := `package models

// @gqlType
type User struct {
	ID   string
	Name string
}
`
	if err := os.WriteFile(userFile, []byte(userCode), 0644); err != nil {
		t.Fatalf("Failed to write user.go: %v", err)
	}

	// Write product.go
	productFile := filepath.Join(modelsDir, "product.go")
	productCode := `package models

// @gqlType
type Product struct {
	ID    string
	Title string
	Price float64
}
`
	if err := os.WriteFile(productFile, []byte(productCode), 0644); err != nil {
		t.Fatalf("Failed to write product.go: %v", err)
	}

	// Write enum.go
	enumFile := filepath.Join(modelsDir, "enum.go")
	enumCode := `package models

// @gqlEnum
type Status string

const (
	StatusPending Status = "PENDING"
	StatusActive  Status = "ACTIVE"
)
`
	if err := os.WriteFile(enumFile, []byte(enumCode), 0644); err != nil {
		t.Fatalf("Failed to write enum.go: %v", err)
	}

	// Parse the directory
	p := NewParser()
	if err := p.Walk(modelsDir); err != nil {
		t.Fatalf("Failed to parse models: %v", err)
	}
	p.MatchEnumConstants()

	// Create output directory
	outDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Configure generator
	config := &Config{
		GenStrategy: "single",
		Output:      filepath.Join(outDir, "schema.graphql"),
		ModelPath:   "models",
	}

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	// Verify we tracked source files for all items
	if len(gen.GeneratedItems) == 0 {
		t.Fatal("No items were generated")
	}

	t.Logf("Generated %d items", len(gen.GeneratedItems))

	// Check that each item has a source file
	for _, item := range gen.GeneratedItems {
		t.Logf("Item: %s (kind: %s, source: %s)", item.GQLName, item.GQLKind, item.GoSourceFile)

		if item.GoSourceFile == "" {
			t.Errorf("Item %s (kind: %s) has no source file", item.GQLName, item.GQLKind)
			continue
		}

		// Verify the source file exists
		if _, err := os.Stat(item.GoSourceFile); err != nil {
			t.Errorf("Source file does not exist for %s: %s", item.GQLName, item.GoSourceFile)
		}

		// Verify the source file matches the expected file
		switch item.GoTypeName {
		case "User":
			if !strings.HasSuffix(item.GoSourceFile, "user.go") {
				t.Errorf("User type should come from user.go, got: %s", item.GoSourceFile)
			}
		case "Product":
			if !strings.HasSuffix(item.GoSourceFile, "product.go") {
				t.Errorf("Product type should come from product.go, got: %s", item.GoSourceFile)
			}
		case "Status":
			if !strings.HasSuffix(item.GoSourceFile, "enum.go") {
				t.Errorf("Status enum should come from enum.go, got: %s", item.GoSourceFile)
			}
		}
	}
}
