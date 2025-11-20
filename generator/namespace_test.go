package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileNamespaceExtraction(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "user.go")
	content := `package models

/**
 * @gqlNamespace(name:"user/auth")
 */

/**
 * @gqlType(name:"User")
 */
type User struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(p.TypeNamespaces) != 1 {
		t.Errorf("Expected 1 namespace entry, got %d", len(p.TypeNamespaces))
	}

	namespace, ok := p.TypeNamespaces["User"]
	if !ok {
		t.Fatal("Expected User type to have namespace")
	}

	if namespace != "user/auth" {
		t.Errorf("Expected namespace 'user/auth', got '%s'", namespace)
	}
}

func TestTypeNamespaceOverride(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "models.go")
	content := `package models

/**
 * @gqlNamespace(name:"common")
 */

/**
 * @gqlType(name:"User",namespace:"user/special")
 */
type User struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}

/**
 * @gqlType(name:"Product")
 */
type Product struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Generate schema
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir
	config.GenStrategy = GenStrategyMultiple

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// User should be in user/special.graphql (type-level override)
	userPath := filepath.Join(outDir, "user", "special.graphql")
	if _, err := os.Stat(userPath); err != nil {
		t.Errorf("Expected user/special.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(userPath)
		if !strings.Contains(string(content), "type User") {
			t.Error("Expected user/special.graphql to contain User type")
		}
	}

	// Product should be in common.graphql (file-level namespace)
	productPath := filepath.Join(outDir, "common.graphql")
	if _, err := os.Stat(productPath); err != nil {
		t.Errorf("Expected common.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(productPath)
		if !strings.Contains(string(content), "type Product") {
			t.Error("Expected common.graphql to contain Product type")
		}
	}
}

func TestEnumNamespace(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "status.go")
	content := `package models

/**
 * @gqlNamespace(name:"common/enums")
 */

/**
 * @gqlEnum
 */
type Status string

const (
	// @gqlEnumValue(name:"ACTIVE")
	StatusActive Status = "ACTIVE"
	// @gqlEnumValue(name:"INACTIVE")
	StatusInactive Status = "INACTIVE"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	p.MatchEnumConstants()

	if len(p.EnumNamespaces) != 1 {
		t.Errorf("Expected 1 enum namespace entry, got %d", len(p.EnumNamespaces))
	}

	namespace, ok := p.EnumNamespaces["Status"]
	if !ok {
		t.Fatal("Expected Status enum to have namespace")
	}

	if namespace != "common/enums" {
		t.Errorf("Expected namespace 'common/enums', got '%s'", namespace)
	}

	// Generate schema
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Enum should be in common/enums.graphql
	enumPath := filepath.Join(outDir, "common", "enums.graphql")
	if _, err := os.Stat(enumPath); err != nil {
		t.Errorf("Expected common/enums.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(enumPath)
		if !strings.Contains(string(content), "enum Status") {
			t.Error("Expected common/enums.graphql to contain Status enum")
		}
	}
}

func TestEnumNamespaceOverride(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "status.go")
	content := `package models

/**
 * @gqlNamespace(name:"common")
 */

/**
 * @gqlEnum(namespace:"special/status")
 */
type Status string

const (
	// @gqlEnumValue(name:"ACTIVE")
	StatusActive Status = "ACTIVE"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	p.MatchEnumConstants()

	// Generate schema
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Enum should be in special/status.graphql (directive-level override)
	enumPath := filepath.Join(outDir, "special", "status.graphql")
	if _, err := os.Stat(enumPath); err != nil {
		t.Errorf("Expected special/status.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(enumPath)
		if !strings.Contains(string(content), "enum Status") {
			t.Error("Expected special/status.graphql to contain Status enum")
		}
	}
}

func TestNamespaceCombining(t *testing.T) {
	tmpDir := t.TempDir()

	// File 1 with api/v1 namespace
	file1 := filepath.Join(tmpDir, "user.go")
	content1 := `package models

/**
 * @gqlNamespace(name:"api/v1")
 */

/**
 * @gqlType(name:"User")
 */
type User struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}
`
	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// File 2 with same api/v1 namespace
	file2 := filepath.Join(tmpDir, "product.go")
	content2 := `package models

/**
 * @gqlNamespace(name:"api/v1")
 */

/**
 * @gqlType(name:"Product")
 */
type Product struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}
`
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Debug: Check namespaces
	t.Logf("TypeNamespaces: %v", p.TypeNamespaces)
	t.Logf("TypeNames: %v", p.TypeNames)

	// Generate schema
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Debug: List all generated files
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			t.Logf("Generated file: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk output directory: %v", err)
	}

	// Both should be in api/v1.graphql
	apiPath := filepath.Join(outDir, "api", "v1.graphql")
	if _, err := os.Stat(apiPath); err != nil {
		t.Errorf("Expected api/v1.graphql to exist: %v", err)
		return
	}

	content, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("Failed to read api/v1.graphql: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "type User") {
		t.Error("Expected api/v1.graphql to contain User type")
	}
	if !strings.Contains(contentStr, "type Product") {
		t.Error("Expected api/v1.graphql to contain Product type")
	}
}

func TestCustomNamespaceSeparator(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "user.go")
	content := `package models

/**
 * @gqlNamespace(name:"user.auth")
 */

/**
 * @gqlType(name:"User")
 */
type User struct {
	ID string ` + "`gql:\"id,type:ID\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Generate schema with dot separator
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir
	config.NamespaceSeparator = "." // Use dot as separator

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Should create user/auth.graphql (dots converted to slashes)
	userPath := filepath.Join(outDir, "user", "auth.graphql")
	if _, err := os.Stat(userPath); err != nil {
		t.Errorf("Expected user/auth.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(userPath)
		if !strings.Contains(string(content), "type User") {
			t.Error("Expected user/auth.graphql to contain User type")
		}
	}
}

func TestInputNamespace(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "input.go")
	content := `package models

/**
 * @gqlNamespace(name:"inputs")
 */

/**
 * @gqlInput(name:"CreateUserInput")
 */
type CreateUser struct {
	Name string ` + "`gql:\"name\"`" + `
}

/**
 * @gqlInput(name:"UpdateUserInput",namespace:"inputs/user")
 */
type UpdateUser struct {
	Name string ` + "`gql:\"name\"`" + `
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	p := NewParser()
	if err := p.Walk(PkgDir(tmpDir)); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Generate schema
	outDir := filepath.Join(tmpDir, "out")
	config := NewConfig()
	config.Output = outDir

	gen := NewGenerator(p, config)
	if err := gen.Run(); err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// CreateUserInput should be in inputs.graphql (file-level namespace)
	inputsPath := filepath.Join(outDir, "inputs.graphql")
	if _, err := os.Stat(inputsPath); err != nil {
		t.Errorf("Expected inputs.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(inputsPath)
		if !strings.Contains(string(content), "input CreateUserInput") {
			t.Error("Expected inputs.graphql to contain CreateUserInput")
		}
	}

	// UpdateUserInput should be in inputs/user.graphql (directive-level override)
	updatePath := filepath.Join(outDir, "inputs", "user.graphql")
	if _, err := os.Stat(updatePath); err != nil {
		t.Errorf("Expected inputs/user.graphql to exist: %v", err)
	} else {
		content, _ := os.ReadFile(updatePath)
		if !strings.Contains(string(content), "input UpdateUserInput") {
			t.Error("Expected inputs/user.graphql to contain UpdateUserInput")
		}
	}
}
