package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnumGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with enum
	testFile := filepath.Join(tmpDir, "enums.go")
	content := `package models

/**
 * @gqlEnum
 * User role in the system
 */
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"  // @gqlEnumValue(name:"ADMIN", description:"Administrator with full access")
	UserRoleEditor UserRole = "editor" // @gqlEnumValue(name:"EDITOR", description:"Can edit content")
	UserRoleViewer UserRole = "viewer" // @gqlEnumValue(name:"VIEWER", description:"Read-only access")
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the file
	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Check that enum was parsed
	if len(parser.EnumNames) != 1 {
		t.Errorf("Expected 1 enum, got %d", len(parser.EnumNames))
	}

	enumType, exists := parser.EnumTypes["UserRole"]
	if !exists {
		t.Fatal("UserRole enum not found")
	}

	// Verify enum properties
	if enumType.Name != "UserRole" {
		t.Errorf("Expected enum name 'UserRole', got '%s'", enumType.Name)
	}

	if enumType.BaseType != "string" {
		t.Errorf("Expected base type 'string', got '%s'", enumType.BaseType)
	}

	if len(enumType.Values) != 3 {
		t.Fatalf("Expected 3 enum values, got %d", len(enumType.Values))
	}

	// Check first value
	if enumType.Values[0].GraphQLName != "ADMIN" {
		t.Errorf("Expected GraphQL name 'ADMIN', got '%s'", enumType.Values[0].GraphQLName)
	}

	// Generate GraphQL schema
	config := NewConfig()
	config.Output = filepath.Join(tmpDir, "schema.graphql")
	config.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Generator Run() error = %v", err)
	}

	// Read generated file
	generated, err := os.ReadFile(config.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedStr := string(generated)

	// Verify enum is in output
	if !strings.Contains(generatedStr, "enum UserRole") {
		t.Error("Generated schema doesn't contain 'enum UserRole'")
	}

	if !strings.Contains(generatedStr, "ADMIN") {
		t.Error("Generated schema doesn't contain 'ADMIN' value")
	}

	if !strings.Contains(generatedStr, "Administrator with full access") {
		t.Error("Generated schema doesn't contain admin description")
	}
}

func TestIntEnumWithIota(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "enums.go")
	content := `package models

/**
 * @gqlEnum
 * Permission level
 */
type Permission int

const (
	PermissionNone  Permission = iota // @gqlEnumValue(name:"NONE")
	PermissionRead                     // @gqlEnumValue(name:"READ")
	PermissionWrite                    // @gqlEnumValue(name:"WRITE")
	PermissionAdmin                    // @gqlEnumValue(name:"ADMIN")
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	enumType, exists := parser.EnumTypes["Permission"]
	if !exists {
		t.Fatal("Permission enum not found")
	}

	if enumType.BaseType != "int" {
		t.Errorf("Expected base type 'int', got '%s'", enumType.BaseType)
	}

	if len(enumType.Values) != 4 {
		t.Fatalf("Expected 4 enum values, got %d", len(enumType.Values))
	}

	// Verify GraphQL names
	expectedNames := []string{"NONE", "READ", "WRITE", "ADMIN"}
	for i, expected := range expectedNames {
		if enumType.Values[i].GraphQLName != expected {
			t.Errorf("Value %d: expected GraphQL name '%s', got '%s'", i, expected, enumType.Values[i].GraphQLName)
		}
	}
}

func TestEnumAutoNaming(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "enums.go")
	content := `package models

/**
 * @gqlEnum
 */
type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusPending  Status = "pending"
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	enumType, exists := parser.EnumTypes["Status"]
	if !exists {
		t.Fatal("Status enum not found")
	}

	// Verify auto-generated names (should strip "Status" prefix and convert to SCREAMING_SNAKE_CASE)
	expectedNames := []string{"ACTIVE", "INACTIVE", "PENDING"}
	for i, expected := range expectedNames {
		if enumType.Values[i].GraphQLName != expected {
			t.Errorf("Value %d: expected auto-generated name '%s', got '%s'", i, expected, enumType.Values[i].GraphQLName)
		}
	}
}

func TestEnumDeprecated(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "enums.go")
	content := `package models

/**
 * @gqlEnum
 */
type Color string

const (
	ColorRed   Color = "red"   // @gqlEnumValue(name:"RED")
	ColorGreen Color = "green" // @gqlEnumValue(name:"GREEN", deprecated:"Use LIME instead")
	ColorBlue  Color = "blue"  // @gqlEnumValue(name:"BLUE")
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	enumType, exists := parser.EnumTypes["Color"]
	if !exists {
		t.Fatal("Color enum not found")
	}

	// Check deprecated value
	if enumType.Values[1].Deprecated != "Use LIME instead" {
		t.Errorf("Expected deprecated reason 'Use LIME instead', got '%s'", enumType.Values[1].Deprecated)
	}

	// Generate and check output
	config := NewConfig()
	config.Output = filepath.Join(tmpDir, "schema.graphql")
	config.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Generator Run() error = %v", err)
	}

	generated, err := os.ReadFile(config.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedStr := string(generated)

	// Verify deprecated directive is in output
	if !strings.Contains(generatedStr, "@deprecated") {
		t.Error("Generated schema doesn't contain @deprecated directive")
	}

	if !strings.Contains(generatedStr, "Use LIME instead") {
		t.Error("Generated schema doesn't contain deprecation reason")
	}
}

func TestEnumCustomName(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "enums.go")
	content := `package models

/**
 * @gqlEnum(name:"Role", description:"User role in the system")
 */
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleEditor UserRole = "editor"
)

/**
 * @gqlEnum(name:"Status")
 */
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
	OrderStatusActive  OrderStatus = "active"
)
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Check UserRole has custom name
	userRoleEnum, exists := parser.EnumTypes["UserRole"]
	if !exists {
		t.Fatal("UserRole enum not found")
	}

	if userRoleEnum.Name != "Role" {
		t.Errorf("Expected custom enum name 'Role', got '%s'", userRoleEnum.Name)
	}

	if userRoleEnum.GoTypeName != "UserRole" {
		t.Errorf("Expected Go type name 'UserRole', got '%s'", userRoleEnum.GoTypeName)
	}

	if userRoleEnum.Description != "User role in the system" {
		t.Errorf("Expected description 'User role in the system', got '%s'", userRoleEnum.Description)
	}

	// Check OrderStatus has custom name
	orderStatusEnum, exists := parser.EnumTypes["OrderStatus"]
	if !exists {
		t.Fatal("OrderStatus enum not found")
	}

	if orderStatusEnum.Name != "Status" {
		t.Errorf("Expected custom enum name 'Status', got '%s'", orderStatusEnum.Name)
	}

	// Generate and verify
	config := NewConfig()
	config.Output = filepath.Join(tmpDir, "schema.graphql")
	config.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Generator Run() error = %v", err)
	}

	generated, err := os.ReadFile(config.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedStr := string(generated)

	// Verify custom names are in output
	if !strings.Contains(generatedStr, "enum Role") {
		t.Error("Generated schema doesn't contain 'enum Role'")
	}

	if !strings.Contains(generatedStr, "enum Status") {
		t.Error("Generated schema doesn't contain 'enum Status'")
	}

	// Verify original names are NOT in output
	if strings.Contains(generatedStr, "enum UserRole") {
		t.Error("Generated schema shouldn't contain 'enum UserRole' when custom name is set")
	}

	if strings.Contains(generatedStr, "enum OrderStatus") {
		t.Error("Generated schema shouldn't contain 'enum OrderStatus' when custom name is set")
	}
}
