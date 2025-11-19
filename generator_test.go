package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablor21/gqlschemagen/generator"
)

// TestStripPrefixSuffix tests the StripPrefixSuffix function
func TestStripPrefixSuffix(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		prefixList string
		suffixList string
		expected   string
	}{
		{
			name:       "strip DTO suffix",
			input:      "UserDTO",
			prefixList: "",
			suffixList: "DTO,Entity",
			expected:   "User",
		},
		{
			name:       "strip Entity suffix",
			input:      "PostEntity",
			prefixList: "",
			suffixList: "DTO,Entity,Model",
			expected:   "Post",
		},
		{
			name:       "strip DB prefix",
			input:      "DBProduct",
			prefixList: "DB,Pg",
			suffixList: "",
			expected:   "Product",
		},
		{
			name:       "strip both prefix and suffix",
			input:      "DBUserDTO",
			prefixList: "DB,Pg",
			suffixList: "DTO,Entity",
			expected:   "User", // Strips both prefix and suffix
		},
		{
			name:       "no match",
			input:      "User",
			prefixList: "DB,Pg",
			suffixList: "DTO,Entity",
			expected:   "User",
		},
		{
			name:       "empty lists",
			input:      "UserDTO",
			prefixList: "",
			suffixList: "",
			expected:   "UserDTO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.StripPrefixSuffix(tt.input, tt.prefixList, tt.suffixList)
			if result != tt.expected {
				t.Errorf("StripPrefixSuffix(%q, %q, %q) = %q, want %q",
					tt.input, tt.prefixList, tt.suffixList, result, tt.expected)
			}
		})
	}
}

// TestFieldNameTransformations tests field name case transformations
func TestFieldNameTransformations(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		fieldCase generator.FieldCase
		expected  string
	}{
		{
			name:      "camel case",
			input:     "FirstName",
			fieldCase: generator.FieldCaseCamel,
			expected:  "firstName",
		},
		{
			name:      "snake case",
			input:     "FirstName",
			fieldCase: generator.FieldCaseSnake,
			expected:  "first_name",
		},
		{
			name:      "pascal case",
			input:     "FirstName",
			fieldCase: generator.FieldCasePascal,
			expected:  "FirstName",
		},
		{
			name:      "original",
			input:     "FirstName",
			fieldCase: generator.FieldCaseOriginal,
			expected:  "FirstName",
		},
		{
			name:      "none",
			input:     "FirstName",
			fieldCase: generator.FieldCaseNone,
			expected:  "FirstName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.TransformFieldName(tt.input, tt.fieldCase)
			if result != tt.expected {
				t.Errorf("TransformFieldName(%q, %v) = %q, want %q",
					tt.input, tt.fieldCase, result, tt.expected)
			}
		})
	}
}

// TestGeneratorWithStripAndAdd tests the full generator with strip and add options
func TestGeneratorWithStripAndAdd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 * @gqlInput()
 */
type UserDTO struct {
	ID   string
	Name string
}

/**
 * @gqlType(name:"CustomName")
 */
type PostEntity struct {
	ID    string
	Title string
}

/**
 * @gqlType()
 */
type DBProduct struct {
	ID    string
	Price float64
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		StripPrefix:         "DB,Pg",
		StripSuffix:         "DTO,Entity",
		AddTypePrefix:       "Gql",
		AddInputSuffix:      "Payload",
		UseJsonTag:          true,
		UseGqlGenDirectives: true,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test type name transformations
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "UserDTO stripped and prefixed",
			contains: "type GqlUser",
			desc:     "UserDTO should become GqlUser (strip DTO + add Gql prefix)",
		},
		{
			name:     "UserDTO input with suffix",
			contains: "input UserInputPayload",
			desc:     "UserDTO input should become UserInputPayload (strip DTO + add Payload suffix)",
		},
		{
			name:     "PostEntity custom name",
			contains: "type CustomName",
			desc:     "PostEntity should keep custom name (no transformations)",
		},
		{
			name:     "DBProduct stripped and prefixed",
			contains: "type GqlProduct",
			desc:     "DBProduct should become GqlProduct (strip DB + add Gql prefix)",
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

// TestJSONIgnoreTag tests that json:"-" fields are ignored
func TestJSONIgnoreTag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 */
type User struct {
	ID       string ` + "`json:\"id\"`" + `
	Name     string ` + "`json:\"name\"`" + `
	Password string ` + "`json:\"-\"`" + `
	Email    string ` + "`json:\"email\"`" + `
}

/**
 * @gqlType()
 */
type SecureUser struct {
	ID           string ` + "`json:\"id\"`" + `
	Name         string ` + "`json:\"name\"`" + `
	PasswordHash string ` + "`json:\"-\" gql:\"include\"`" + `
	ApiKey       string ` + "`json:\"-\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config with useJsonTag enabled
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          true,
		UseGqlGenDirectives: true,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test that json:"-" fields are properly handled
	tests := []struct {
		name        string
		shouldExist bool
		field       string
		desc        string
	}{
		{
			name:        "Password field ignored",
			shouldExist: false,
			field:       "password:",
			desc:        "Password field with json:\"-\" should be ignored",
		},
		{
			name:        "ApiKey field ignored",
			shouldExist: false,
			field:       "apiKey:",
			desc:        "ApiKey field with json:\"-\" should be ignored",
		},
		{
			name:        "PasswordHash field included",
			shouldExist: true,
			field:       "passwordHash:",
			desc:        "PasswordHash field with json:\"-\" and gql:\"include\" should be included",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(schema, tt.field)
			if contains != tt.shouldExist {
				if tt.shouldExist {
					t.Errorf("Schema should contain %q: %s\nGenerated schema:\n%s",
						tt.field, tt.desc, schema)
				} else {
					t.Errorf("Schema should NOT contain %q: %s\nGenerated schema:\n%s",
						tt.field, tt.desc, schema)
				}
			}
		})
	}
}

// TestFieldNamePriority tests the field name resolution priority
func TestFieldNamePriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 */
type User struct {
	Field1 string ` + "`gql:\"gqlName\" json:\"jsonName\"`" + `
	Field2 string ` + "`json:\"jsonOnly\"`" + `
	Field3 string ` + "`gql:\"gqlOnly\"`" + `
	Field4 string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          true,
		FieldCase:           generator.FieldCaseCamel,
		UseGqlGenDirectives: true,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test field name priority: gql tag > json tag > struct field
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "gql tag priority",
			contains: "gqlName:",
			desc:     "Field1 should use gql tag name (highest priority)",
		},
		{
			name:     "json tag fallback",
			contains: "jsonOnly:",
			desc:     "Field2 should use json tag name when no gql tag",
		},
		{
			name:     "gql tag only",
			contains: "gqlOnly:",
			desc:     "Field3 should use gql tag name",
		},
		{
			name:     "struct field transformation",
			contains: "field4:",
			desc:     "Field4 should use transformed struct field name (camelCase)",
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

// TestGqlIgnoreAll tests @gqlIgnoreAll directive with include flag
func TestGqlIgnoreAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 * @gqlIgnoreAll
 */
type SecureUser struct {
	ID       string ` + "`gql:\"include\"`" + `
	Email    string ` + "`gql:\"include\"`" + `
	Password string
	Internal string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          true,
		UseGqlGenDirectives: true,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test that only included fields are present
	tests := []struct {
		name        string
		shouldExist bool
		field       string
		desc        string
	}{
		{
			name:        "ID field included",
			shouldExist: true,
			field:       "id:",
			desc:        "ID field with include flag should be present",
		},
		{
			name:        "Email field included",
			shouldExist: true,
			field:       "email:",
			desc:        "Email field with include flag should be present",
		},
		{
			name:        "Password field excluded",
			shouldExist: false,
			field:       "password:",
			desc:        "Password field without include flag should be excluded",
		},
		{
			name:        "Internal field excluded",
			shouldExist: false,
			field:       "internal:",
			desc:        "Internal field without include flag should be excluded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(schema, tt.field)
			if contains != tt.shouldExist {
				if tt.shouldExist {
					t.Errorf("Schema should contain %q: %s\nGenerated schema:\n%s",
						tt.field, tt.desc, schema)
				} else {
					t.Errorf("Schema should NOT contain %q: %s\nGenerated schema:\n%s",
						tt.field, tt.desc, schema)
				}
			}
		})
	}
}

// TestModelPathConfiguration tests custom @goModel directive path
func TestModelPathConfiguration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 */
type User struct {
	ID   string
	Name string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config with custom ModelPath
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          true,
		UseGqlGenDirectives: true,
		ModelPath:           "github.com/user/myapp/domain",
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test that custom model path is used
	expectedDirective := "@goModel(model: \"github.com/user/myapp/domain.User\")"
	if !strings.Contains(schema, expectedDirective) {
		t.Errorf("Schema should contain custom model path directive %q\nGenerated schema:\n%s",
			expectedDirective, schema)
	}
}

// TestInputGeneration tests input type generation
func TestInputGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := `package test

/**
 * @gqlType()
 * @gqlInput()
 */
type CreateUser struct {
	Name  string
	Email string
}

/**
 * @gqlType()
 * @gqlInput(name:"CustomInput")
 */
type UpdateUser struct {
	ID    string
	Name  string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          true,
		UseGqlGenDirectives: true,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test input generation
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "CreateUser type",
			contains: "type CreateUser",
			desc:     "CreateUser type should be generated",
		},
		{
			name:     "CreateUserInput",
			contains: "input CreateUserInput",
			desc:     "CreateUserInput should be generated with default name",
		},
		{
			name:     "UpdateUser type",
			contains: "type UpdateUser",
			desc:     "UpdateUser type should be generated",
		},
		{
			name:     "CustomInput",
			contains: "input CustomInput",
			desc:     "Custom input name should be used",
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

// TestReferencedStructs tests that structs referenced in fields are also included in the schema
func TestReferencedStructs(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create test Go files with cross-referenced structs
	userFile := filepath.Join(tmpDir, "user.go")
	userContent := `package test

/**
 * @gqlType()
 */
type User struct {
	ID    string
	Name  string
	Posts []Post
}
`
	if err := os.WriteFile(userFile, []byte(userContent), 0644); err != nil {
		t.Fatalf("Failed to write user file: %v", err)
	}

	postFile := filepath.Join(tmpDir, "post.go")
	postContent := `package test

/**
 * @gqlType()
 */
type Post struct {
	ID      string
	UserID  string
	Title   string
	Author  *User
}
`
	if err := os.WriteFile(postFile, []byte(postContent), 0644); err != nil {
		t.Fatalf("Failed to write post file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          false,
		UseGqlGenDirectives: true,
		FieldCase:           generator.FieldCaseCamel,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test that both User and Post types are generated
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "User type exists",
			contains: "type User",
			desc:     "User type should be generated",
		},
		{
			name:     "Post type exists",
			contains: "type Post",
			desc:     "Post type should be generated",
		},
		{
			name:     "User has posts field",
			contains: "posts: [Post!]!",
			desc:     "User should have posts field referencing Post type",
		},
		{
			name:     "Post has author field",
			contains: "author: User",
			desc:     "Post should have author field referencing User type",
		},
		{
			name:     "Post has userID field",
			contains: "userID: String!",
			desc:     "Post should have userID field",
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

// Test that extra fields are generated correctly
func TestTypeExtraFieldDirectives(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create test Go file with @gqlTypeExtraField annotations
	testFile := filepath.Join(tmpDir, "user.go")
	testContent := `package test

/**
 * @gqlType()
 * @gqlTypeExtraField(name:"fullName",type:"String!",description:"Computed full name")
 * @gqlTypeExtraField(name:"avatar",type:"Avatar",description:"User avatar")
 */
type User struct {
ID        string
FirstName string
LastName  string
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "schema.graphql")

	// Create config
	cfg := &generator.Config{
		Packages:            []string{tmpDir},
		Output:              outFile,
		GenStrategy:         generator.GenStrategySingle,
		UseJsonTag:          false,
		UseGqlGenDirectives: true,
		FieldCase:           generator.FieldCaseCamel,
	}

	// Run generator
	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(tmpDir)); err != nil {
		t.Fatalf("Parser walk failed: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		t.Fatalf("Generator run failed: %v", err)
	}

	// Read generated file
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	schema := string(content)

	// Test that extra fields are generated as actual fields
	tests := []struct {
		name     string
		contains string
		desc     string
	}{
		{
			name:     "User type exists",
			contains: "type User",
			desc:     "User type should be generated",
		},
		{
			name:     "fullName extra field",
			contains: "fullName: String!",
			desc:     "fullName extra field should be added to schema",
		},
		{
			name:     "avatar extra field",
			contains: "avatar: Avatar",
			desc:     "avatar extra field should be added to schema",
		},
		{
			name:     "fullName has forceResolver",
			contains: "fullName: String! @goField(forceResolver: true)",
			desc:     "fullName should have @goField(forceResolver: true)",
		},
		{
			name:     "User has firstName field",
			contains: "firstName: String!",
			desc:     "User should have firstName field from struct",
		},
		{
			name:     "User has lastName field",
			contains: "lastName: String!",
			desc:     "User should have lastName field from struct",
		},
		{
			name:     "fullName description",
			contains: `"""Computed full name"""`,
			desc:     "fullName should have description",
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
