package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeepGenericNesting verifies that generics work with arbitrary nesting depth
// like Connection[Test[X[D]]] where we have 4 levels of nesting
func TestDeepGenericNesting(t *testing.T) {
	code := `package test

// Level 4 - innermost type
type Data struct {
	Value string ` + "`json:\"value\"`" + `
}

// Level 3 - wraps Data
type Wrapper[T any] struct {
	Item T   ` + "`json:\"item\"`" + `
	ID   int ` + "`json:\"id\"`" + `
}

// Level 2 - wraps Wrapper
type Container[T any] struct {
	Content T      ` + "`json:\"content\"`" + `
	Label   string ` + "`json:\"label\"`" + `
}

// Level 1 - wraps Container
type Response[T any] struct {
	Data   T    ` + "`json:\"data\"`" + `
	Status int  ` + "`json:\"status\"`" + `
}

/**
 * @gqlType
 * Deep nesting: Response[Container[Wrapper[*Data]]]
 */
type DeepResult struct {
	Response[Container[Wrapper[*Data]]]
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

	cfg := NewConfig()
	cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	cfg.GenStrategy = GenStrategySingle
	cfg.AutoGenerate.Enabled = true
	cfg.AutoGenerate.Strategy = AutoGenReferenced

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// With arbitrary-depth generic resolution, all levels should be fully expanded and inlined
	// Response[Container[Wrapper[*Data]]] should expand to:
	// - item: Data! (from Wrapper[*Data] innermost)
	// - id: Int! (from Wrapper)
	// - label: String! (from Container)
	// - status: Int! (from Response)
	// - cached: Boolean! (direct field)
	expected := []string{
		"type DeepResult",
		"item:",     // From Wrapper[*Data].Item after full resolution
		"id:",       // From Wrapper[*Data].ID
		"label:",    // From Container[Wrapper[*Data]].Label
		"status:",   // From Response[...].Status
		"cached:",   // Direct field
		"type Data", // Auto-generated
		"value:",    // From Data.Value
	}

	for _, exp := range expected {
		if !strings.Contains(schema, exp) {
			t.Errorf("Schema missing: %q\n\nGenerated:\n%s", exp, schema)
		}
	}

	// Verify the types are correctly resolved (not generic type parameters)
	if strings.Contains(schema, ": T!") || strings.Contains(schema, ": T)") {
		t.Errorf("Schema contains unresolved type parameter 'T'\n\nGenerated:\n%s", schema)
	}
}

// TestExtremelyDeepNesting tests 5+ levels of generic nesting
func TestExtremelyDeepNesting(t *testing.T) {
	code := `package test

type A struct {
	Name string ` + "`json:\"name\"`" + `
}

type B[T any] struct {
	B_Field T   ` + "`json:\"b_field\"`" + `
	B_Value int ` + "`json:\"b_value\"`" + `
}

type C[T any] struct {
	C_Field  T      ` + "`json:\"c_field\"`" + `
	C_String string ` + "`json:\"c_string\"`" + `
}

type D[T any] struct {
	D_Field T    ` + "`json:\"d_field\"`" + `
	D_Bool  bool ` + "`json:\"d_bool\"`" + `
}

type E[T any] struct {
	E_Field T      ` + "`json:\"e_field\"`" + `
	E_Label string ` + "`json:\"e_label\"`" + `
}

/**
 * @gqlType
 * 5 levels deep: E[D[C[B[*A]]]]
 */
type VeryDeepResult struct {
	E[D[C[B[*A]]]]
	Top bool ` + "`json:\"top\"`" + `
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
	cfg.AutoGenerate.Enabled = true
	cfg.AutoGenerate.Strategy = AutoGenReferenced

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// All 5 levels are fully expanded, and generic fields are recursively inlined
	// E[D[C[B[*A]]]] expands all the way down, leaving only:
	// - The innermost concrete field: b_field: A! (from B[*A])
	// - All non-generic fields from each level
	expected := []string{
		"type VeryDeepResult",
		"b_field:",  // From B[*A] - the only generic field that survives (innermost concrete type)
		"b_value:",  // From B
		"c_string:", // From C
		"d_bool:",   // From D
		"e_label:",  // From E
		"top:",      // Direct field
		"type A",    // Auto-generated
		"name:",     // From A
	}

	for _, exp := range expected {
		if !strings.Contains(schema, exp) {
			t.Errorf("Schema missing: %q\n\nGenerated:\n%s", exp, schema)
		}
	}

	// Verify no unresolved type parameters
	if strings.Contains(schema, ": T!") || strings.Contains(schema, ": T)") {
		t.Errorf("Schema contains unresolved type parameter 'T'\n\nGenerated:\n%s", schema)
	}
}
