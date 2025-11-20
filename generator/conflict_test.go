package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamespaceAndPackageConflict tests that when namespace and package strategies
// would write to the same file, the content gets merged properly
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
	} // Should contain Product type (from package strategy)
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

// TestMultipleNamespacesToSameFile tests that multiple types with the same namespace
// get merged into one file correctly
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
