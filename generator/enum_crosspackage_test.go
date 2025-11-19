package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnumCrossPackage(t *testing.T) {
	// Create a temp directory structure for test packages
	tmpDir := t.TempDir()

	// Create types package with enum type declaration
	typesDir := filepath.Join(tmpDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatal(err)
	}

	typesFile := filepath.Join(typesDir, "status.go")
	typesContent := `package types

// Status represents the processing status
// @gqlEnum
type Status string
`
	if err := os.WriteFile(typesFile, []byte(typesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create constants package with enum values in a different package
	constsDir := filepath.Join(tmpDir, "constants")
	if err := os.MkdirAll(constsDir, 0755); err != nil {
		t.Fatal(err)
	}

	constsFile := filepath.Join(constsDir, "status_values.go")
	constsContent := `package constants

import "../types"

const (
	StatusPending  types.Status = "PENDING"
	StatusActive   types.Status = "ACTIVE"
	StatusComplete types.Status = "COMPLETE"
)
`
	if err := os.WriteFile(constsFile, []byte(constsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse both packages
	p := NewParser()
	if err := p.Walk(typesDir); err != nil {
		t.Fatalf("Walk types package failed: %v", err)
	}
	if err := p.Walk(constsDir); err != nil {
		t.Fatalf("Walk constants package failed: %v", err)
	}

	// Match enum constants after all packages are parsed
	p.MatchEnumConstants()

	// Verify the enum was created
	if len(p.EnumTypes) != 1 {
		t.Fatalf("Expected 1 enum type, got %d", len(p.EnumTypes))
	}

	statusEnum, ok := p.EnumTypes["Status"]
	if !ok {
		t.Fatal("Status enum not found")
	}

	if statusEnum.Name != "Status" {
		t.Errorf("Expected enum name 'Status', got '%s'", statusEnum.Name)
	}

	if len(statusEnum.Values) != 3 {
		t.Fatalf("Expected 3 enum values, got %d", len(statusEnum.Values))
	}

	// Check the values
	expectedValues := map[string]string{
		"StatusPending":  "PENDING",
		"StatusActive":   "ACTIVE",
		"StatusComplete": "COMPLETE",
	}

	for _, v := range statusEnum.Values {
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
	if !strings.Contains(schema, "enum Status") {
		t.Error("Generated schema does not contain 'enum Status'")
	}
	if !strings.Contains(schema, "PENDING") {
		t.Error("Generated schema does not contain 'PENDING' value")
	}
	if !strings.Contains(schema, "ACTIVE") {
		t.Error("Generated schema does not contain 'ACTIVE' value")
	}
	if !strings.Contains(schema, "COMPLETE") {
		t.Error("Generated schema does not contain 'COMPLETE' value")
	}
}
