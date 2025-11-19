package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnumCrossFile(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Create types.go with enum type declaration
	typesFile := filepath.Join(tmpDir, "types.go")
	typesContent := `package testpkg

// @gqlEnum
type Color string
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create constants.go with enum values in a separate file
	constsFile := filepath.Join(tmpDir, "constants.go")
	constsContent := `package testpkg

const (
	ColorRed    Color = "RED"
	ColorGreen  Color = "GREEN"
	ColorBlue   Color = "BLUE"
)
`
	if err := os.WriteFile(constsFile, []byte(constsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the directory
	p := NewParser()
	if err := p.Walk(tmpDir); err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Match enum constants (required after parsing)
	p.MatchEnumConstants()

	// Verify the enum was created
	if len(p.EnumTypes) != 1 {
		t.Fatalf("Expected 1 enum type, got %d", len(p.EnumTypes))
	}

	colorEnum, ok := p.EnumTypes["Color"]
	if !ok {
		t.Fatal("Color enum not found")
	}

	if colorEnum.Name != "Color" {
		t.Errorf("Expected enum name 'Color', got '%s'", colorEnum.Name)
	}

	if len(colorEnum.Values) != 3 {
		t.Fatalf("Expected 3 enum values, got %d", len(colorEnum.Values))
	}

	// Check the values
	expectedValues := map[string]string{
		"ColorRed":   "RED",
		"ColorGreen": "GREEN",
		"ColorBlue":  "BLUE",
	}

	for _, v := range colorEnum.Values {
		expected, ok := expectedValues[v.GoName]
		if !ok {
			t.Errorf("Unexpected enum value: %s", v.GoName)
			continue
		}
		if v.GraphQLName != expected {
			t.Errorf("Expected GraphQL name '%s', got '%s'", expected, v.GraphQLName)
		}
	}

	// Generate schema
	config := &Config{
		FieldCase:   FieldCaseCamel,
		GenStrategy: GenStrategySingle,
		Output:      filepath.Join(tmpDir, "schema.graphqls"),
	}
	g := NewGenerator(p, config)
	if err := g.Run(); err != nil {
		t.Fatalf("Generator Run() error = %v", err)
	}

	// Read generated schema
	schemaBytes, err := os.ReadFile(config.Output)
	if err != nil {
		t.Fatalf("Failed to read generated schema: %v", err)
	}
	schema := string(schemaBytes)

	// Verify the enum appears in the schema
	if !strings.Contains(schema, "enum Color") {
		t.Error("Generated schema does not contain 'enum Color'")
	}
	if !strings.Contains(schema, "RED") {
		t.Error("Generated schema does not contain 'RED' value")
	}
	if !strings.Contains(schema, "GREEN") {
		t.Error("Generated schema does not contain 'GREEN' value")
	}
	if !strings.Contains(schema, "BLUE") {
		t.Error("Generated schema does not contain 'BLUE' value")
	}
}
