package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDeprecatedField tests deprecated field directive generation
func TestDeprecatedField(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType(name:"User")
 */
type User struct {
	ID          string  ` + "`" + `gql:"id,type:ID"` + "`" + `
	Name        string  ` + "`" + `gql:"name"` + "`" + `
	Email       string  ` + "`" + `gql:"email,deprecated"` + "`" + `
	OldField    string  ` + "`" + `gql:"oldField,deprecated:\"Use newField instead\""` + "`" + `
	ActiveField string  ` + "`" + `gql:"activeField"` + "`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	parser := NewParser()
	if err := parser.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "email deprecated without reason",
			contains: "email: String! @deprecated",
			desc:     "Email field should have @deprecated directive without reason",
		},
		{
			name:     "oldField deprecated with reason",
			contains: `oldField: String! @deprecated(reason: "Use newField instead")`,
			desc:     "OldField should have @deprecated directive with reason",
		},
		{
			name:     "activeField not deprecated",
			contains: "activeField: String!",
			desc:     "ActiveField should not have @deprecated directive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(schema, tt.contains) {
				t.Errorf("Schema should contain %q: %s\nGenerated schema:\n%s",
					tt.contains, tt.desc, schema)
			}
		})
	}
}

// TestDeprecatedWithDescription tests deprecated with description
func TestDeprecatedWithDescription(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType(name:"Product")
 */
type Product struct {
	ID    string  ` + "`" + `gql:"id,type:ID"` + "`" + `
	Price float64 ` + "`" + `gql:"price,description:\"Product price\",deprecated:\"Use priceV2 field\""` + "`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	parser := NewParser()
	if err := parser.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	if !strings.Contains(schema, `"""Product price"""`) {
		t.Errorf("Schema should contain description\nGenerated schema:\n%s", schema)
	}

	if !strings.Contains(schema, `price: Float! @deprecated(reason: "Use priceV2 field")`) {
		t.Errorf("Schema should contain deprecated directive\nGenerated schema:\n%s", schema)
	}
}

// TestDeprecatedWithCommasInDescription tests that commas in descriptions work
func TestDeprecatedWithCommasInDescription(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType(name:"Order")
 */
type Order struct {
	ID       string ` + "`" + `gql:"id,type:ID"` + "`" + `
	Status   string ` + "`" + `gql:"status,description:\"Order status: pending, processing, completed, or cancelled\""` + "`" + `
	OldNotes string ` + "`" + `gql:"oldNotes,description:\"Legacy notes field, now deprecated\",deprecated:\"Use notes array instead, as it supports multiple entries\""` + "`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
	}

	parser := NewParser()
	if err := parser.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "description with commas",
			contains: "Order status: pending, processing, completed, or cancelled",
			desc:     "Status description should contain all comma-separated values",
		},
		{
			name:     "deprecated reason with commas",
			contains: "Use notes array instead, as it supports multiple entries",
			desc:     "Deprecated reason should contain all comma-separated values",
		},
		{
			name:     "oldNotes has both description and deprecated",
			contains: "Legacy notes field, now deprecated",
			desc:     "OldNotes should have description with commas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(schema, tt.contains) {
				t.Errorf("Schema should contain %q: %s\nGenerated schema:\n%s",
					tt.contains, tt.desc, schema)
			}
		})
	}
}

// TestDeprecatedInInput tests deprecated in input types
func TestDeprecatedInInput(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlInput(name:"UpdateUserInput")
 */
type UpdateUserInput struct {
	Name     string  ` + "`" + `gql:"name"` + "`" + `
	Email    string  ` + "`" + `gql:"email"` + "`" + `
	Username *string ` + "`" + `gql:"username,deprecated:\"Use email instead\""` + "`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: false,
	}

	parser := NewParser()
	if err := parser.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	if !strings.Contains(schema, "input UpdateUserInput") {
		t.Errorf("Schema should contain UpdateUserInput\nGenerated schema:\n%s", schema)
	}

	if !strings.Contains(schema, `username: String! @deprecated(reason: "Use email instead")`) {
		t.Errorf("Schema should contain deprecated username field\nGenerated schema:\n%s", schema)
	}
}
