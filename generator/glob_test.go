package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobPatternSupport(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   models/
	//     user.go
	//   internal/
	//     domain/
	//       entities/
	//         product.go
	//     services/
	//       models/
	//         order.go

	testFiles := map[string]string{
		"models/user.go": `package models
/**
 * @gqlType(name:"User")
 */
type User struct {
	ID string
}`,
		"internal/domain/entities/product.go": `package entities
/**
 * @gqlType(name:"Product")
 */
type Product struct {
	ID string
}`,
		"internal/services/models/order.go": `package models
/**
 * @gqlType(name:"Order")
 */
type Order struct {
	ID string
}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		wantTypeCount int
		wantTypes     []string
	}{
		{
			name:          "direct path",
			pattern:       filepath.Join(tmpDir, "models"),
			wantTypeCount: 1,
			wantTypes:     []string{"User"},
		},
		{
			name:          "recursive glob - all models",
			pattern:       filepath.Join(tmpDir, "**/models"),
			wantTypeCount: 2,
			wantTypes:     []string{"User", "Order"},
		},
		{
			name:          "recursive glob - specific file",
			pattern:       filepath.Join(tmpDir, "**/*.go"),
			wantTypeCount: 3,
			wantTypes:     []string{"User", "Product", "Order"},
		},
		{
			name:          "recursive glob - entities",
			pattern:       filepath.Join(tmpDir, "**/entities/*.go"),
			wantTypeCount: 1,
			wantTypes:     []string{"Product"},
		},
		{
			name:          "simple glob",
			pattern:       filepath.Join(tmpDir, "*/user.go"),
			wantTypeCount: 1,
			wantTypes:     []string{"User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			if err := parser.Walk(tt.pattern); err != nil {
				t.Fatalf("Walk() error = %v", err)
			}

			if len(parser.TypeNames) != tt.wantTypeCount {
				t.Errorf("Walk() found %d types, want %d. Types: %v",
					len(parser.TypeNames), tt.wantTypeCount, parser.TypeNames)
			}

			// Check that all expected types are found
			typeMap := make(map[string]bool)
			for _, typeName := range parser.TypeNames {
				typeMap[typeName] = true
			}

			for _, wantType := range tt.wantTypes {
				if !typeMap[wantType] {
					t.Errorf("Expected type %q not found. Found types: %v", wantType, parser.TypeNames)
				}
			}
		})
	}
}

func TestWalkGlobPattern(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test structure
	testFiles := map[string]string{
		"models/user.go": `package models
type User struct { ID string }`,
		"internal/models/product.go": `package models
type Product struct { ID string }`,
		"internal/deep/models/order.go": `package models
type Order struct { ID string }`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	parser := NewParser()
	pattern := filepath.Join(tmpDir, "**/models/*.go")

	err := parser.Walk(pattern)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Should find all 3 files
	expectedTypes := 3
	if len(parser.TypeNames) != expectedTypes {
		t.Errorf("Walk() found %d types, want %d", len(parser.TypeNames), expectedTypes)
	}
}

func TestWalkInvalidPattern(t *testing.T) {
	parser := NewParser()

	// Test invalid pattern with multiple **
	pattern := "/**/models/**/*.go"
	err := parser.Walk(pattern)

	if err == nil {
		t.Error("Expected error for invalid pattern, got nil")
	}
}

func TestWalkNonExistentPath(t *testing.T) {
	parser := NewParser()

	// This should not error - glob just returns no matches
	pattern := "/nonexistent/path/**/*.go"
	err := parser.Walk(pattern)

	// Should complete without error (just no files found)
	if err != nil {
		t.Errorf("Walk() unexpected error = %v", err)
	}

	if len(parser.TypeNames) != 0 {
		t.Errorf("Expected 0 types for nonexistent path, got %d", len(parser.TypeNames))
	}
}
