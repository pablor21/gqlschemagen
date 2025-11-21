package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoGenWithEnums tests that enums are auto-generated and used correctly in both types and inputs
func TestAutoGenWithEnums(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gqlschemagen-enum-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	modelsDir := filepath.Join(tmpDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create an enum type
	statusFile := `package models

/**
 * @gqlEnum
 */
type Status string

const (
	StatusPending Status = "PENDING"
	StatusActive  Status = "ACTIVE"
	StatusDone    Status = "DONE"
)
`
	if err := os.WriteFile(filepath.Join(modelsDir, "status.go"), []byte(statusFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a type that references the enum
	taskFile := `package models

/**
 * @gqlType
 */
type Task struct {
	ID     string
	Title  string
	Status Status
}
`
	if err := os.WriteFile(filepath.Join(modelsDir, "task.go"), []byte(taskFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an input that references the enum
	updateTaskFile := `package models

/**
 * @gqlInput
 */
type UpdateTaskInput struct {
	Title  string
	Status Status
}
`
	if err := os.WriteFile(filepath.Join(modelsDir, "update_task.go"), []byte(updateTaskFile), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	if err := parser.Walk(modelsDir); err != nil {
		t.Fatal(err)
	}

	// Match enum constants
	parser.MatchEnumConstants()

	t.Logf("Found %d enums", len(parser.EnumNames))
	for _, enumName := range parser.EnumNames {
		t.Logf("Enum: %s", enumName)
	}

	config := NewConfig()
	config.Output = filepath.Join(tmpDir, "schema")
	config.OutputFileName = "schema.graphqls"
	config.GenStrategy = GenStrategySingle
	config.AutoGenerate.Enabled = true
	config.AutoGenerate.Strategy = AutoGenReferenced
	config.AutoGenerate.OutOfScopeTypes = OutOfScopeIgnore

	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatal(err)
	}

	schemaPath := filepath.Join(tmpDir, "schema", "schema.graphqls")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)

	t.Logf("Generated schema:\n%s", schema)

	// Verify enum was generated
	if !strings.Contains(schema, "enum Status") {
		t.Error("Expected Status enum to be auto-generated")
	}

	// Verify enum values
	if !strings.Contains(schema, "PENDING") {
		t.Error("Expected PENDING enum value")
	}

	// Verify Task type uses the enum
	if !strings.Contains(schema, "type Task") {
		t.Error("Expected Task type to be generated")
	}
	if !strings.Contains(schema, "status: Status!") {
		t.Error("Expected Task to have status field with Status enum type")
	}

	// Verify UpdateTaskInput uses the enum (same as type, not StatusInput)
	if !strings.Contains(schema, "input UpdateTaskInput") {
		t.Error("Expected UpdateTaskInput to be generated")
	}

	// Count how many times "status: Status!" appears - should be in both type and input
	statusFieldCount := strings.Count(schema, "status: Status!")
	if statusFieldCount < 2 {
		t.Errorf("Expected 'status: Status!' to appear at least 2 times (in type and input), got %d", statusFieldCount)
	}

	// Ensure there's NO StatusInput (enums don't have input variants)
	if strings.Contains(schema, "StatusInput") {
		t.Error("Did not expect StatusInput to be generated (enums are the same for types and inputs)")
	}
}
