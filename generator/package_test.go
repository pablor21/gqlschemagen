package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
