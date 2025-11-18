package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablor21/gqlschemagen/generator"
)

func TestNew(t *testing.T) {
	p := New()

	if p == nil {
		t.Fatal("New() returned nil")
	}

	if p.cfg == nil {
		t.Fatal("Plugin config is nil")
	}

	if p.cfg.GenStrategy != generator.GenStrategySingle {
		t.Errorf("Expected GenStrategy to be %s, got %s", generator.GenStrategySingle, p.cfg.GenStrategy)
	}

	if p.cfg.Output != "graph/schema/gqlschemagen.graphqls" {
		t.Errorf("Expected Output to be 'graph/schema/gqlschemagen.graphqls', got %s", p.cfg.Output)
	}

	if len(p.Packages) != 0 {
		t.Errorf("Expected Packages to be empty, got %d items", len(p.Packages))
	}
}

func TestName(t *testing.T) {
	p := New()
	name := p.Name()

	if name != "gqlschemagen" {
		t.Errorf("Expected plugin name to be 'gqlschemagen', got '%s'", name)
	}
}

func TestMutateConfig_NoPackages(t *testing.T) {
	p := New()

	// Should return nil when no packages specified
	err := p.MutateConfig(nil)
	if err != nil {
		t.Errorf("Expected nil error when no packages specified, got: %v", err)
	}
}

func TestGenerateSchema(t *testing.T) {
	// Create a temporary directory for output
	tmpDir, err := os.MkdirTemp("", "gqlschemagen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	p := New()
	// Set output to a file path, not just directory
	p.cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	p.cfg.GenStrategy = generator.GenStrategySingle
	p.cfg.FieldCase = generator.FieldCaseCamel

	// Use the examples package from the parent directory
	examplesPath := filepath.Join("..", "examples", "cli", "models")
	if _, err := os.Stat(examplesPath); os.IsNotExist(err) {
		t.Skip("Examples directory not found, skipping test")
	}

	// Generate schema from examples package
	err = p.generateSchema(examplesPath)
	if err != nil {
		t.Fatalf("generateSchema failed: %v", err)
	}

	// Check if output file was created
	if _, err := os.Stat(p.cfg.Output); os.IsNotExist(err) {
		t.Errorf("Expected output file %s to be created", p.cfg.Output)
	}

	// Read and validate the generated schema
	content, err := os.ReadFile(p.cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Check for expected types in the schema
	expectedTypes := []string{
		"type User",
		"type Post",
		"type Comment",
		"type UserProfile",
	}

	for _, expectedType := range expectedTypes {
		if !contains(schema, expectedType) {
			t.Errorf("Expected schema to contain '%s', but it doesn't", expectedType)
		}
	}

	// The example models don't have sensitive fields like SecureUser did,
	// so we'll just verify the schema was generated
	if len(schema) == 0 {
		t.Error("Generated schema is empty")
	}
}

func TestValidation(t *testing.T) {
	p := New()
	p.cfg.FieldCase = "invalid"

	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "gqlschemagen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	p.cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	p.cfg.Input = "../examples"

	// Should fail validation with invalid field case
	err = p.generateSchema("../examples")
	if err == nil {
		t.Error("Expected validation error for invalid field case, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
