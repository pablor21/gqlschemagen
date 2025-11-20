package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// Enum Generation Tests
// ============================================================================

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

	// Match enum constants
	parser.MatchEnumConstants()

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

	// Match enum constants
	parser.MatchEnumConstants()

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

	// Match enum constants
	parser.MatchEnumConstants()

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

	// Match enum constants
	parser.MatchEnumConstants()

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

	// Match enum constants
	parser.MatchEnumConstants()

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

// ============================================================================
// Deprecated Field Tests
// ============================================================================

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

// ============================================================================
// Package Strategy Tests
// ============================================================================

func TestPackageStrategy(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "TestPackageStrategy")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create package structure
	usersDir := filepath.Join(tmpDir, "users")
	postsDir := filepath.Join(tmpDir, "posts")
	err = os.MkdirAll(usersDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create users dir: %v", err)
	}
	err = os.MkdirAll(postsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create posts dir: %v", err)
	}

	// Create users/user.go
	userGo := `package users

/**
 * @gqlType()
 */
type User struct {
	ID    string ` + "`gql:\"type:ID\"`" + `
	Email string
	Name  string
}
`
	if err := os.WriteFile(filepath.Join(usersDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatalf("Failed to write user.go: %v", err)
	}

	// Create users/profile.go
	profileGo := `package users

/**
 * @gqlType()
 */
type Profile struct {
	UserID string ` + "`gql:\"type:ID\"`" + `
	Bio    string
}
`
	if err := os.WriteFile(filepath.Join(usersDir, "profile.go"), []byte(profileGo), 0644); err != nil {
		t.Fatalf("Failed to write profile.go: %v", err)
	}

	// Create posts/post.go
	postGo := `package posts

/**
 * @gqlType()
 */
type Post struct {
	ID     string ` + "`gql:\"type:ID\"`" + `
	Title  string
	Author string
}
`
	if err := os.WriteFile(filepath.Join(postsDir, "post.go"), []byte(postGo), 0644); err != nil {
		t.Fatalf("Failed to write post.go: %v", err)
	}

	// Create posts/comment.go
	commentGo := `package posts

/**
 * @gqlType()
 */
type Comment struct {
	ID      string ` + "`gql:\"type:ID\"`" + `
	PostID  string ` + "`gql:\"type:ID\"`" + `
	Content string
}
`
	if err := os.WriteFile(filepath.Join(postsDir, "comment.go"), []byte(commentGo), 0644); err != nil {
		t.Fatalf("Failed to write comment.go: %v", err)
	}

	// Parse the packages
	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}

	// Configure generator with package strategy
	outDir := filepath.Join(tmpDir, "out")
	config := &Config{
		Output:              outDir,
		GenStrategy:         GenStrategyPackage,
		UseGqlGenDirectives: true,
		FieldCase:           FieldCaseCamel,
	}

	// Generate
	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Verify output files exist
	usersSchema := filepath.Join(outDir, "users.graphql")
	postsSchema := filepath.Join(outDir, "posts.graphql")

	if !FileExists(usersSchema) {
		t.Errorf("Expected users.graphql to be generated")
	}

	if !FileExists(postsSchema) {
		t.Errorf("Expected posts.graphql to be generated")
	}

	// Read and verify users.graphql
	usersContent, err := os.ReadFile(usersSchema)
	if err != nil {
		t.Fatalf("Failed to read users.graphql: %v", err)
	}

	usersStr := string(usersContent)
	if !strings.Contains(usersStr, "type User") {
		t.Errorf("users.graphql should contain 'type User'")
	}
	if !strings.Contains(usersStr, "type Profile") {
		t.Errorf("users.graphql should contain 'type Profile'")
	}

	// Read and verify posts.graphql
	postsContent, err := os.ReadFile(postsSchema)
	if err != nil {
		t.Fatalf("Failed to read posts.graphql: %v", err)
	}

	postsStr := string(postsContent)
	if !strings.Contains(postsStr, "type Post") {
		t.Errorf("posts.graphql should contain 'type Post'")
	}
	if !strings.Contains(postsStr, "type Comment") {
		t.Errorf("posts.graphql should contain 'type Comment'")
	}

	// Ensure users.graphql doesn't contain posts types
	if strings.Contains(usersStr, "type Post") {
		t.Errorf("users.graphql should not contain 'type Post'")
	}
	if strings.Contains(usersStr, "type Comment") {
		t.Errorf("users.graphql should not contain 'type Comment'")
	}

	// Ensure posts.graphql doesn't contain users types
	if strings.Contains(postsStr, "type User") {
		t.Errorf("posts.graphql should not contain 'type User'")
	}
	if strings.Contains(postsStr, "type Profile") {
		t.Errorf("posts.graphql should not contain 'type Profile'")
	}
}

func TestPackageStrategyWithEnums(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "TestPackageStrategyWithEnums")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create package structure
	usersDir := filepath.Join(tmpDir, "users")
	err = os.MkdirAll(usersDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create users dir: %v", err)
	}

	// Create users/role.go with enum
	roleGo := `package users

/**
 * @gqlEnum()
 */
type Role string

const (
	RoleAdmin  Role = "ADMIN"  // @gqlEnumValue(name:"ADMIN")
	RoleEditor Role = "EDITOR" // @gqlEnumValue(name:"EDITOR")
	RoleViewer Role = "VIEWER" // @gqlEnumValue(name:"VIEWER")
)

/**
 * @gqlType()
 */
type User struct {
	ID   string ` + "`gql:\"type:ID\"`" + `
	Name string
	Role Role
}
`
	if err := os.WriteFile(filepath.Join(usersDir, "role.go"), []byte(roleGo), 0644); err != nil {
		t.Fatalf("Failed to write role.go: %v", err)
	}

	// Parse the packages
	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}

	// Match enums
	parser.MatchEnumConstants()

	// Configure generator with package strategy
	outDir := filepath.Join(tmpDir, "out")
	config := &Config{
		Output:              outDir,
		GenStrategy:         GenStrategyPackage,
		UseGqlGenDirectives: true,
		FieldCase:           FieldCaseCamel,
	}

	// Generate
	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Verify output file exists
	usersSchema := filepath.Join(outDir, "users.graphql")
	if !FileExists(usersSchema) {
		t.Fatalf("Expected users.graphql to be generated")
	}

	// Read and verify users.graphql
	usersContent, err := os.ReadFile(usersSchema)
	if err != nil {
		t.Fatalf("Failed to read users.graphql: %v", err)
	}

	usersStr := string(usersContent)

	// Check for enum
	if !strings.Contains(usersStr, "enum Role") {
		t.Errorf("users.graphql should contain 'enum Role'")
	}
	if !strings.Contains(usersStr, "ADMIN") {
		t.Errorf("users.graphql should contain 'ADMIN' enum value")
	}
	if !strings.Contains(usersStr, "EDITOR") {
		t.Errorf("users.graphql should contain 'EDITOR' enum value")
	}
	if !strings.Contains(usersStr, "VIEWER") {
		t.Errorf("users.graphql should contain 'VIEWER' enum value")
	}

	// Check for type
	if !strings.Contains(usersStr, "type User") {
		t.Errorf("users.graphql should contain 'type User'")
	}
	if !strings.Contains(usersStr, "role: Role!") {
		t.Errorf("users.graphql should contain 'role: Role!' field")
	}
}

// ============================================================================
// Namespace and Conflict Tests
// ============================================================================

func TestNamespaceAndPackageConflict(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "TestNamespaceAndPackageConflict")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create models directory
	modelsDir := filepath.Join(tmpDir, "models")
	err = os.MkdirAll(modelsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create models dir: %v", err)
	}

	// Create models/user.go with namespace "models"
	userGo := `package models

/**
 * @gqlNamespace(name:"models")
 */

/**
 * @gqlType()
 */
type User struct {
	ID   string ` + "`gql:\"type:ID\"`" + `
	Name string
}
`
	if err := os.WriteFile(filepath.Join(modelsDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatalf("Failed to write user.go: %v", err)
	}

	// Create models/product.go without namespace (should use package strategy)
	productGo := `package models

/**
 * @gqlType()
 */
type Product struct {
	ID    string ` + "`gql:\"type:ID\"`" + `
	Title string
	Price float64
}
`
	if err := os.WriteFile(filepath.Join(modelsDir, "product.go"), []byte(productGo), 0644); err != nil {
		t.Fatalf("Failed to write product.go: %v", err)
	}

	// Parse the packages
	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}

	// Configure generator with package strategy
	outDir := filepath.Join(tmpDir, "out")
	config := &Config{
		Output:              outDir,
		GenStrategy:         GenStrategyPackage,
		UseGqlGenDirectives: true,
		FieldCase:           FieldCaseCamel,
		NamespaceSeparator:  "/",
	}

	// Generate
	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Both should be in models.graphql
	modelsSchema := filepath.Join(outDir, "models.graphql")
	if !FileExists(modelsSchema) {
		t.Fatalf("Expected models.graphql to be generated")
	}

	// Read and verify models.graphql contains both types
	content, err := os.ReadFile(modelsSchema)
	if err != nil {
		t.Fatalf("Failed to read models.graphql: %v", err)
	}

	contentStr := string(content)

	t.Logf("Generated models.graphql content:\n%s", contentStr)

	// Should contain User type (from namespace)
	if !strings.Contains(contentStr, "type User") {
		t.Errorf("models.graphql should contain 'type User' (from namespace)")
	}
	// Should contain Product type (from package strategy)
	if !strings.Contains(contentStr, "type Product") {
		t.Errorf("models.graphql should contain 'type Product' (from package strategy)")
	}

	// Verify both types are complete
	if !strings.Contains(contentStr, "id: ID") {
		t.Errorf("models.graphql should contain ID fields")
	}
	if !strings.Contains(contentStr, "name: String") {
		t.Errorf("models.graphql should contain User's name field")
	}
	if !strings.Contains(contentStr, "title: String") {
		t.Errorf("models.graphql should contain Product's title field")
	}
	if !strings.Contains(contentStr, "price: Float") {
		t.Errorf("models.graphql should contain Product's price field")
	}
}

func TestMultipleNamespacesToSameFile(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "TestMultipleNamespacesToSameFile")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create different packages but same namespace
	usersDir := filepath.Join(tmpDir, "users")
	authDir := filepath.Join(tmpDir, "auth")
	err = os.MkdirAll(usersDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create users dir: %v", err)
	}
	err = os.MkdirAll(authDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create auth dir: %v", err)
	}

	// Create users/user.go with namespace "api"
	userGo := `package users

/**
 * @gqlNamespace(name:"api")
 */

/**
 * @gqlType()
 */
type User struct {
	ID   string ` + "`gql:\"type:ID\"`" + `
	Name string
}
`
	if err := os.WriteFile(filepath.Join(usersDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatalf("Failed to write user.go: %v", err)
	}

	// Create auth/token.go with same namespace "api"
	tokenGo := `package auth

/**
 * @gqlNamespace(name:"api")
 */

/**
 * @gqlType()
 */
type Token struct {
	Value     string
	ExpiresAt string
}
`
	if err := os.WriteFile(filepath.Join(authDir, "token.go"), []byte(tokenGo), 0644); err != nil {
		t.Fatalf("Failed to write token.go: %v", err)
	}

	// Parse the packages
	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Failed to parse packages: %v", err)
	}

	// Configure generator
	outDir := filepath.Join(tmpDir, "out")
	config := &Config{
		Output:              outDir,
		GenStrategy:         GenStrategyPackage, // Strategy doesn't matter when namespaces are present
		UseGqlGenDirectives: true,
		FieldCase:           FieldCaseCamel,
		NamespaceSeparator:  "/",
	}

	// Generate
	gen := NewGenerator(parser, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Both should be in api.graphql (not users.graphql or auth.graphql)
	apiSchema := filepath.Join(outDir, "api.graphql")
	if !FileExists(apiSchema) {
		t.Fatalf("Expected api.graphql to be generated")
	}

	// Verify users.graphql and auth.graphql don't exist
	if FileExists(filepath.Join(outDir, "users.graphql")) {
		t.Errorf("users.graphql should not exist when namespace is specified")
	}
	if FileExists(filepath.Join(outDir, "auth.graphql")) {
		t.Errorf("auth.graphql should not exist when namespace is specified")
	}

	// Read and verify api.graphql contains both types
	content, err := os.ReadFile(apiSchema)
	if err != nil {
		t.Fatalf("Failed to read api.graphql: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "type User") {
		t.Errorf("api.graphql should contain 'type User'")
	}
	if !strings.Contains(contentStr, "type Token") {
		t.Errorf("api.graphql should contain 'type Token'")
	}
}

// ============================================================================
// Extra Fields Tests
// ============================================================================

func TestShouldApplyExtraField(t *testing.T) {
	tests := []struct {
		name       string
		ef         ExtraField
		targetName string
		want       bool
	}{
		{
			name:       "no filter - empty on",
			ef:         ExtraField{On: []string{}},
			targetName: "User",
			want:       true,
		},
		{
			name:       "wildcard filter",
			ef:         ExtraField{On: []string{"*"}},
			targetName: "User",
			want:       true,
		},
		{
			name:       "exact match",
			ef:         ExtraField{On: []string{"User"}},
			targetName: "User",
			want:       true,
		},
		{
			name:       "no match",
			ef:         ExtraField{On: []string{"Article"}},
			targetName: "User",
			want:       false,
		},
		{
			name:       "multiple targets - match",
			ef:         ExtraField{On: []string{"Article", "BlogPost"}},
			targetName: "BlogPost",
			want:       true,
		},
		{
			name:       "multiple targets - no match",
			ef:         ExtraField{On: []string{"Article", "BlogPost"}},
			targetName: "User",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldApplyExtraField(tt.ef, tt.targetName)
			if got != tt.want {
				t.Errorf("shouldApplyExtraField() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Single-Line Comment and Case-Insensitive Directive Tests
// ============================================================================

func TestSingleLineCommentDirectives(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @gqlType(name:"User")
type User struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
	Name string
}

// @gqlEnum
type Status string

const (
	StatusActive Status = "ACTIVE" // @gqlEnumValue(name:"ACTIVE")
)
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	parser.MatchEnumConstants()

	// Check that User type was recognized
	if len(parser.TypeNames) == 0 {
		t.Error("Expected at least 1 type")
	}

	// Check that Status enum was recognized
	if len(parser.EnumTypes) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(parser.EnumTypes))
	}

	statusEnum, ok := parser.EnumTypes["Status"]
	if !ok {
		t.Fatal("Status enum not found")
	}

	if len(statusEnum.Values) != 1 {
		t.Errorf("Expected 1 enum value, got %d", len(statusEnum.Values))
	}

	// Generate schema
	outFile := filepath.Join(tmpDir, "schema.graphql")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
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

	if !strings.Contains(schema, "type User") {
		t.Error("Schema should contain User type from single-line comment directive")
	}

	if !strings.Contains(schema, "enum Status") {
		t.Error("Schema should contain Status enum from single-line comment directive")
	}
}

func TestCapitalizedGqlDirectives(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @GqlType(name:"Product")
 * @GqlInput(name:"ProductInput")
 */
type Product struct {
	ID    string ` + "`gql:\"id,type:ID\"`" + `
	Title string
}

/**
 * @GqlEnum
 */
type Role string

const (
	RoleAdmin Role = "ADMIN" // @GqlEnumValue(name:"ADMIN")
)
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	parser.MatchEnumConstants()

	// Check that Product type was recognized
	if len(parser.TypeNames) == 0 {
		t.Error("Expected at least 1 type")
	}

	// Check that Role enum was recognized
	if len(parser.EnumTypes) != 1 {
		t.Fatalf("Expected 1 enum, got %d", len(parser.EnumTypes))
	}

	roleEnum, ok := parser.EnumTypes["Role"]
	if !ok {
		t.Fatal("Role enum not found")
	}

	if len(roleEnum.Values) != 1 {
		t.Errorf("Expected 1 enum value, got %d", len(roleEnum.Values))
	}

	// Generate schema
	outFile := filepath.Join(tmpDir, "schema.graphql")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
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

	if !strings.Contains(schema, "type Product") {
		t.Error("Schema should contain Product type from @GqlType directive")
	}

	if !strings.Contains(schema, "input ProductInput") {
		t.Error("Schema should contain ProductInput from @GqlInput directive")
	}

	if !strings.Contains(schema, "enum Role") {
		t.Error("Schema should contain Role enum from @GqlEnum directive")
	}
}

func TestMixedCommentStyles(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

// @GqlType(name:"User")
type User struct {
	ID string
}

/**
 * @gqlType(name:"Post")
 */
type Post struct {
	ID string
}

// @GqlEnum
type Status string

const (
	StatusActive Status = "ACTIVE" // @GqlEnumValue(name:"ACTIVE")
)

/**
 * @gqlEnum
 */
type Priority int

const (
	PriorityHigh Priority = 1 // @gqlEnumValue(name:"HIGH")
)
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	parser.MatchEnumConstants()

	// Check that types were recognized (at least the 2 with directives)
	if len(parser.TypeNames) < 2 {
		t.Errorf("Expected at least 2 types, got %d", len(parser.TypeNames))
	}

	// Check that both enums were recognized
	if len(parser.EnumTypes) != 2 {
		t.Fatalf("Expected 2 enums, got %d", len(parser.EnumTypes))
	}

	// Verify Status enum
	statusEnum, ok := parser.EnumTypes["Status"]
	if !ok {
		t.Error("Status enum not found")
	} else if len(statusEnum.Values) != 1 {
		t.Errorf("Status: expected 1 enum value, got %d", len(statusEnum.Values))
	}

	// Verify Priority enum
	priorityEnum, ok := parser.EnumTypes["Priority"]
	if !ok {
		t.Error("Priority enum not found")
	} else if len(priorityEnum.Values) != 1 {
		t.Errorf("Priority: expected 1 enum value, got %d", len(priorityEnum.Values))
	}

	// Generate schema
	outFile := filepath.Join(tmpDir, "schema.graphql")
	cfg := &Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         GenStrategySingle,
		UseGqlGenDirectives: true,
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

	if !strings.Contains(schema, "type User") {
		t.Error("Schema should contain User type from single-line @GqlType")
	}

	if !strings.Contains(schema, "type Post") {
		t.Error("Schema should contain Post type from block @gqlType")
	}

	if !strings.Contains(schema, "enum Status") {
		t.Error("Schema should contain Status enum from single-line @GqlEnum")
	}

	if !strings.Contains(schema, "enum Priority") {
		t.Error("Schema should contain Priority enum from block @gqlEnum")
	}
}
