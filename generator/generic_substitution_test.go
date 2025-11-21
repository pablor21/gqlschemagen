package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenericTypeSubstitution verifies that generic type parameters are correctly substituted
// when embedded generic types are instantiated with concrete types
func TestGenericTypeSubstitution(t *testing.T) {
	code := `package test

// Generic Result wrapper
type Result[T any] struct {
	Data  T      ` + "`json:\"data\"`" + `
	Error string ` + "`json:\"error\"`" + `
}

// Concrete User type
type User struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

/**
 * @gqlType
 */
type UserResult struct {
	Result[*User]  // T should be substituted with *User
	Cached bool ` + "`json:\"cached\"`" + `
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Verify type parameters were parsed
	if len(parser.TypeParameters) == 0 {
		t.Fatal("Expected TypeParameters to be populated")
	}
	if params, ok := parser.TypeParameters["Result"]; !ok || len(params) != 1 || params[0] != "T" {
		t.Errorf("Expected Result to have type parameter [T], got: %v", params)
	}

	cfg := NewConfig()
	cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	cfg.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// The data field should be "User!" (substituted from T)
	// NOT "T!" (unresolved type parameter)
	expected := []string{
		"type UserResult",
		"data: User!", // T was substituted with *User -> User!
		"error: String!",
		"cached: Boolean!",
		"type User",
		"id: String!",
		"name: String!",
	}

	for _, exp := range expected {
		if !strings.Contains(schema, exp) {
			t.Errorf("Schema missing expected: %q\n\nGenerated:\n%s", exp, schema)
		}
	}

	// Should NOT contain unresolved type parameter
	if strings.Contains(schema, "data: T!") {
		t.Errorf("Schema should not contain unresolved type parameter 'T!'\n\nGenerated:\n%s", schema)
	}
}

// TestUnresolvedGenericTypeFallback verifies the config option for unresolved type parameters
func TestUnresolvedGenericTypeFallback(t *testing.T) {
	code := `package test

// Generic container
type Container[T any] struct {
	Value T ` + "`json:\"value\"`" + `
}

/**
 * @gqlType
 * Note: Container is used standalone without type argument, so T remains unresolved
 */
type Container struct {
	Value string ` + "`json:\"value\" gql:\"type:String!\"`" + `
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	cfg := NewConfig()
	cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	cfg.GenStrategy = GenStrategySingle
	cfg.AutoGenerate.UnresolvedGenericType = "Any" // Use Any scalar for unresolved type parameters
	cfg.AutoGenerate.SuppressGenericTypeWarnings = true

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// Should contain the explicit type from gql tag
	if !strings.Contains(schema, "value: String!") {
		t.Errorf("Schema should contain 'value: String!' from gql tag\n\nGenerated:\n%s", schema)
	}
}

// TestMultipleTypeParameters verifies substitution with multiple type parameters (K, V)
func TestMultipleTypeParameters(t *testing.T) {
	code := `package test

// Generic map with key-value pairs
type Pair[K comparable, V any] struct {
	Key   K ` + "`json:\"key\"`" + `
	Value V ` + "`json:\"value\"`" + `
}

/**
 * @gqlType
 */
type StringIntPair struct {
	Pair[string, int]  // K=string, V=int
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Verify multiple type parameters were parsed
	if params, ok := parser.TypeParameters["Pair"]; !ok || len(params) != 2 {
		t.Errorf("Expected Pair to have 2 type parameters, got: %v", params)
	}

	cfg := NewConfig()
	cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	cfg.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// Both type parameters should be substituted
	expected := []string{
		"type StringIntPair",
		"key: String!", // K substituted with string
		"value: Int!",  // V substituted with int
	}

	for _, exp := range expected {
		if !strings.Contains(schema, exp) {
			t.Errorf("Schema missing expected: %q\n\nGenerated:\n%s", exp, schema)
		}
	}

	// Should NOT contain unresolved type parameters
	if strings.Contains(schema, ": K") || strings.Contains(schema, ": V") {
		t.Errorf("Schema should not contain unresolved type parameters\n\nGenerated:\n%s", schema)
	}
}
