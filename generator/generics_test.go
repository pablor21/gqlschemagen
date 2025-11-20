package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenericsSupport(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedSchema string
		shouldContain  []string
	}{
		{
			name: "simple generic embedded struct",
			code: `package test

// Edge represents a Relay connection edge
type Edge[T any] struct {
	Node   T      ` + "`json:\"node\"`" + `
	Cursor string ` + "`json:\"cursor\"`" + `
}

// Connection represents a Relay connection
type Connection[T any] struct {
	Edges    []*Edge[T] ` + "`json:\"edges\"`" + `
	PageInfo *PageInfo  ` + "`json:\"pageInfo\"`" + `
}

type PageInfo struct {
	HasNextPage bool ` + "`json:\"hasNextPage\"`" + `
}

type Comment struct {
	ID   string ` + "`json:\"id\"`" + `
	Text string ` + "`json:\"text\"`" + `
}

/**
 * @gqlType
 */
type CommentConnection struct {
	Connection[*Comment]
	Count int64 ` + "`gql:\"type:Int\"`" + `
}
`,
			shouldContain: []string{
				"type CommentConnection",
				"edges:",
				"pageInfo:",
				"count: Int",
			},
		},
		{
			name: "generic type alias",
			code: `package test

type Edge[T any] struct {
	Node   T      ` + "`json:\"node\"`" + `
	Cursor string ` + "`json:\"cursor\"`" + `
}

/**
 * @gqlType
 */
type User struct {
	Name string ` + "`json:\"name\"`" + `
}

/**
 * @gqlType
 */
type UserEdge struct {
	Edge[*User]
}
`,
			shouldContain: []string{
				"type UserEdge",
				"node:",
				"cursor:",
			},
		},
		{
			name: "nested generic with count field",
			code: `package test

type Edge[T any] struct {
	Node   T      ` + "`json:\"node\"`" + `
	Cursor string ` + "`json:\"cursor\"`" + `
}

type Connection[T any] struct {
	Edges    []*Edge[T] ` + "`json:\"edges\"`" + `
	PageInfo *PageInfo  ` + "`json:\"pageInfo\"`" + `
}

/**
 * @gqlType
 */
type PageInfo struct {
	HasNextPage     bool   ` + "`json:\"hasNextPage\"`" + `
	HasPreviousPage bool   ` + "`json:\"hasPreviousPage\"`" + `
	StartCursor     string ` + "`json:\"startCursor\"`" + `
	EndCursor       string ` + "`json:\"endCursor\"`" + `
}

/**
 * @gqlType
 */
type Post struct {
	ID    string ` + "`json:\"id\"`" + `
	Title string ` + "`json:\"title\"`" + `
}

/**
 * @gqlType
 */
type PostConnection struct {
	Connection[*Post]
	TotalCount int ` + "`gql:\"type:Int!\"`" + `
}
`,
			shouldContain: []string{
				"type PostConnection",
				"edges:",
				"pageInfo:",
				"totalCount: Int!",
				"hasNextPage:",
				"hasPreviousPage:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test files
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")

			// Write test file
			if err := os.WriteFile(testFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Parse the package
			parser := NewParser()
			if err := parser.Walk(tmpDir); err != nil {
				t.Fatalf("Walk() error = %v", err)
			}

			// Create config
			cfg := NewConfig()
			cfg.Output = filepath.Join(tmpDir, "schema.graphql")
			cfg.GenStrategy = GenStrategySingle

			// Create generator and generate schema
			gen := NewGenerator(parser, cfg)
			if err := gen.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			// Read generated schema file
			generated, err := os.ReadFile(cfg.Output)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			schema := string(generated)

			// Check expected content
			for _, expected := range tt.shouldContain {
				if !strings.Contains(schema, expected) {
					t.Errorf("Generated schema missing expected content: %q\n\nGenerated schema:\n%s", expected, schema)
				}
			}
		})
	}
}

func TestGenericEmbeddedFieldExpansion(t *testing.T) {
	code := `package test

type BaseStruct[T any] struct {
	Data  T      ` + "`json:\"data\"`" + `
	Count int    ` + "`json:\"count\"`" + `
	Error string ` + "`json:\"error\" gql:\"omit\"`" + `
}

type User struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

/**
 * @gqlType
 */
type UserResponse struct {
	BaseStruct[*User]
	Success bool ` + "`json:\"success\"`" + `
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	if err := parser.Walk(tmpDir); err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	cfg := NewConfig()
	cfg.Output = filepath.Join(tmpDir, "schema.graphql")
	cfg.GenStrategy = GenStrategySingle

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Read generated schema file
	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	// Should contain fields from embedded generic struct
	expectedFields := []string{
		"data:",
		"count:",
		"success:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(schema, field) {
			t.Errorf("Schema missing expected field: %q\n\nGenerated schema:\n%s", field, schema)
		}
	}

	// Should NOT contain omitted field
	if strings.Contains(schema, "error:") {
		t.Errorf("Schema should not contain omitted field 'error'\n\nGenerated schema:\n%s", schema)
	}
}
