package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenericsWithAutoGeneration tests auto-generation with generic types
// Note: Generic type parameters (T, K, V) that aren't resolved show as out-of-scope
// This is expected behavior - the tool expands embedded generic structs but doesn't
// fully resolve uninstantiated type parameters
func TestGenericsWithAutoGeneration(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		autoGenStrategy  AutoGenerateStrategy
		maxDepth         int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "auto-generate generic type parameters",
			code: `package test

type Response[T any] struct {
	Data    T      ` + "`json:\"data\"`" + `
	Success bool   ` + "`json:\"success\"`" + `
	Message string ` + "`json:\"message\"`" + `
}

type User struct {
	ID       string ` + "`json:\"id\"`" + `
	Username string ` + "`json:\"username\"`" + `
	Email    string ` + "`json:\"email\"`" + `
}

/**
 * @gqlType
 */
type UserResponse struct {
	Response[*User]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type UserResponse",
				"data:",
				"success:",
				"message:",
				"type User",
				"username:",
				"email:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate nested generic dependencies",
			code: `package test

type Edge[T any] struct {
	Node   T      ` + "`json:\"node\"`" + `
	Cursor string ` + "`json:\"cursor\"`" + `
}

type Connection[T any] struct {
	Edges      []*Edge[T] ` + "`json:\"edges\"`" + `
	TotalCount int        ` + "`json:\"totalCount\"`" + `
}

type Product struct {
	ID    string  ` + "`json:\"id\"`" + `
	Name  string  ` + "`json:\"name\"`" + `
	Price float64 ` + "`json:\"price\"`" + `
}

/**
 * @gqlType
 */
type ProductConnection struct {
	Connection[*Product]
	HasMore bool ` + "`json:\"hasMore\"`" + `
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        3,
			shouldContain: []string{
				"type ProductConnection",
				"edges:",
				"totalCount:",
				"hasMore:",
				"type Product",
				"name:",
				"price:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate with multiple generic parameters",
			code: `package test

type Pair[K any, V any] struct {
	Key   K ` + "`json:\"key\"`" + `
	Value V ` + "`json:\"value\"`" + `
}

type StringValue struct {
	Text string ` + "`json:\"text\"`" + `
}

type IntKey struct {
	Number int ` + "`json:\"number\"`" + `
}

/**
 * @gqlType
 */
type ConfigPair struct {
	Pair[*IntKey, *StringValue]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type ConfigPair",
				"key:",
				"value:",
				"type IntKey",
				"number:",
				"type StringValue",
				"text:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate with depth limit",
			code: `package test

type Container[T any] struct {
	Item T ` + "`json:\"item\"`" + `
}

type Level1 struct {
	Data    *Level2 ` + "`json:\"data\"`" + `
	Message string  ` + "`json:\"message\"`" + `
}

type Level2 struct {
	Detail *Level3 ` + "`json:\"detail\"`" + `
	Code   int     ` + "`json:\"code\"`" + `
}

type Level3 struct {
	Value string ` + "`json:\"value\"`" + `
}

/**
 * @gqlType
 */
type MyContainer struct {
	Container[*Level1]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2, // Only Level1 and Level2, not Level3
			shouldContain: []string{
				"type MyContainer",
				"item:",
				"type Level1",
				"message:",
				"type Level2",
				"code:",
			},
			shouldNotContain: []string{
				"type Level3",
			},
		},
		{
			name: "auto-generate generic with input types",
			code: `package test

type Result[T any] struct {
	Data  T    ` + "`json:\"data\"`" + `
	Error *Err ` + "`json:\"error\"`" + `
}

type Err struct {
	Code    int    ` + "`json:\"code\"`" + `
	Message string ` + "`json:\"message\"`" + `
}

/**
 * @gqlType
 * @gqlInput(name: "CreateUserInput")
 */
type CreateUser struct {
	Username string ` + "`json:\"username\"`" + `
	Email    string ` + "`json:\"email\"`" + `
}

/**
 * @gqlType
 */
type CreateUserResult struct {
	Result[*CreateUser]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type CreateUserResult",
				"data:",
				"error:",
				"type CreateUser",
				"username:",
				"email:",
				"input CreateUserInput",
				"type Err",
				"code:",
				"message:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate generic with circular references",
			code: `package test

type Node[T any] struct {
	Value    T      ` + "`json:\"value\"`" + `
	Children []*T   ` + "`json:\"children\" gql:\"omit\"`" + `
	Label    string ` + "`json:\"label\"`" + `
}

type TreeItem struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

/**
 * @gqlType
 */
type TreeNode struct {
	Node[*TreeItem]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type TreeNode",
				"value:",
				"label:",
				"type TreeItem",
				"name:",
			},
			shouldNotContain: []string{
				"children:",
			},
		},
		{
			name: "auto-generate generic with slice types",
			code: `package test

type Collection[T any] struct {
	Items []T    ` + "`json:\"items\"`" + `
	Total int    ` + "`json:\"total\"`" + `
	Empty bool   ` + "`json:\"empty\"`" + `
}

type Tag struct {
	Name  string ` + "`json:\"name\"`" + `
	Color string ` + "`json:\"color\"`" + `
}

/**
 * @gqlType
 */
type TagCollection struct {
	Collection[*Tag]
	Featured bool ` + "`json:\"featured\"`" + `
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type TagCollection",
				"items:",
				"total:",
				"empty:",
				"featured:",
				"type Tag",
				"name:",
				"color:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate with omitted fields",
			code: `package test

type Wrapper[T any] struct {
	Content  T      ` + "`json:\"content\"`" + `
	Metadata *Meta  ` + "`json:\"metadata\"`" + `
}

type Meta struct {
	CreatedAt string ` + "`json:\"createdAt\"`" + `
	UpdatedAt string ` + "`json:\"updatedAt\"`" + `
}

type Article struct {
	Title   string        ` + "`json:\"title\"`" + `
	Content string        ` + "`json:\"content\"`" + `
	Private string        ` + "`json:\"private\" gql:\"omit\"`" + `
}

/**
 * @gqlType
 */
type ArticleWrapper struct {
	Wrapper[*Article]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        3,
			shouldContain: []string{
				"type ArticleWrapper",
				"content:",
				"metadata:",
				"type Article",
				"title:",
				"type Meta",
				"createdAt:",
			},
			shouldNotContain: []string{
				"private:",
			},
		},
		{
			name: "auto-generate generic with custom field types",
			code: `package test

type Box[T any] struct {
	Value   T      ` + "`json:\"value\"`" + `
	IsEmpty bool   ` + "`json:\"isEmpty\"`" + `
	Count   int64  ` + "`json:\"count\" gql:\"type:Int\"`" + `
}

type Widget struct {
	ID    string ` + "`json:\"id\" gql:\"type:ID!\"`" + `
	Label string ` + "`json:\"label\"`" + `
}

/**
 * @gqlType
 */
type WidgetBox struct {
	Box[*Widget]
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type WidgetBox",
				"value:",
				"isEmpty:",
				"count: Int",
				"type Widget",
				"id: ID!",
				"label:",
			},
			shouldNotContain: []string{},
		},
		{
			name: "auto-generate generic with map types",
			code: `package test

type Dict[K comparable, V any] struct {
	Entries map[K]V ` + "`json:\"entries\" gql:\"ignore\"`" + `
	Size    int     ` + "`json:\"size\"`" + `
}

type Setting struct {
	Key   string ` + "`json:\"key\"`" + `
	Value string ` + "`json:\"value\"`" + `
}

/**
 * @gqlType
 */
type SettingDict struct {
	Dict[string, *Setting]
	Items []*Setting ` + "`json:\"items\"`" + `
}
`,
			autoGenStrategy: AutoGenReferenced,
			maxDepth:        2,
			shouldContain: []string{
				"type SettingDict",
				"size:",
				"items:",
				"type Setting",
				"key:",
				"value:",
			},
			shouldNotContain: []string{
				"entries:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.go")

			if err := os.WriteFile(testFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			parser := NewParser()
			if err := parser.Walk(tmpDir); err != nil {
				t.Fatalf("Walk() error = %v", err)
			}

			cfg := NewConfig()
			cfg.Output = filepath.Join(tmpDir, "schema.graphql")
			cfg.GenStrategy = GenStrategySingle
			cfg.AutoGenerate.Enabled = true
			cfg.AutoGenerate.Strategy = tt.autoGenStrategy
			cfg.AutoGenerate.MaxDepth = tt.maxDepth

			gen := NewGenerator(parser, cfg)
			if err := gen.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			generated, err := os.ReadFile(cfg.Output)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			schema := string(generated)

			// Check expected content
			for _, expected := range tt.shouldContain {
				if !strings.Contains(schema, expected) {
					t.Errorf("Schema missing expected content: %q\n\nGenerated schema:\n%s", expected, schema)
				}
			}

			// Check content that should NOT be present
			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(schema, unexpected) {
					t.Errorf("Schema contains unexpected content: %q\n\nGenerated schema:\n%s", unexpected, schema)
				}
			}
		})
	}
}

func TestGenericConstraintsWithAutoGen(t *testing.T) {
	code := `package test

type Comparable interface {
	comparable
}

type Number interface {
	~int | ~int64 | ~float64
}

type Container[T Comparable] struct {
	Key   T      ` + "`json:\"key\"`" + `
	Label string ` + "`json:\"label\"`" + `
}

type NumericBox[T Number] struct {
	Value T      ` + "`json:\"value\" gql:\"type:Float\"`" + `
	Max   T      ` + "`json:\"max\" gql:\"type:Float\"`" + `
	Min   T      ` + "`json:\"min\" gql:\"type:Float\"`" + `
}

/**
 * @gqlType
 */
type StringContainer struct {
	Container[string]
	Extra string ` + "`json:\"extra\"`" + `
}

/**
 * @gqlType
 */
type IntBox struct {
	NumericBox[int]
	Overflow bool ` + "`json:\"overflow\"`" + `
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
	cfg.AutoGenerate.Enabled = true
	cfg.AutoGenerate.Strategy = AutoGenReferenced
	cfg.AutoGenerate.MaxDepth = 2

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	expectedFields := []string{
		"type StringContainer",
		"key:",
		"label:",
		"extra:",
		"type IntBox",
		"value: Float",
		"max: Float",
		"min: Float",
		"overflow:",
	}

	for _, expected := range expectedFields {
		if !strings.Contains(schema, expected) {
			t.Errorf("Schema missing expected: %q\n\nGenerated:\n%s", expected, schema)
		}
	}
}

func TestComplexGenericNestingWithAutoGen(t *testing.T) {
	code := `package test

// Generic repository pattern
type Repository[T any] struct {
	Items []*T ` + "`json:\"items\"`" + `
	Count int  ` + "`json:\"count\"`" + `
}

// Generic response wrapper
type Response[T any] struct {
	Data    T      ` + "`json:\"data\"`" + `
	Status  int    ` + "`json:\"status\"`" + `
	Message string ` + "`json:\"message\"`" + `
}

// Actual data models
type Author struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

type Post struct {
	ID       string  ` + "`json:\"id\"`" + `
	Title    string  ` + "`json:\"title\"`" + `
	Author   *Author ` + "`json:\"author\"`" + `
	Comments int     ` + "`json:\"comments\"`" + `
}

/**
 * @gqlType
 */
type PostRepositoryResponse struct {
	Response[*Repository[Post]]
	Cached bool ` + "`json:\"cached\"`" + `
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
	cfg.AutoGenerate.Enabled = true
	cfg.AutoGenerate.Strategy = AutoGenReferenced
	cfg.AutoGenerate.MaxDepth = 4 // Need depth for nested generics

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	expected := []string{
		"type PostRepositoryResponse",
		"data:",
		"status:",
		"message:",
		"cached:",
		"items:",
		"count:",
		"type Post",
		"title:",
		"author:",
		"type Author",
		"name:",
	}

	for _, exp := range expected {
		if !strings.Contains(schema, exp) {
			t.Errorf("Schema missing: %q\n\nGenerated:\n%s", exp, schema)
		}
	}
}

func TestGenericsWithAutoGenInputTypes(t *testing.T) {
	code := `package test

type Payload[T any] struct {
	Data T ` + "`json:\"data\"`" + `
}

/**
 * @gqlType
 * @gqlInput(name: "UserInput")
 */
type User struct {
	Username string ` + "`json:\"username\"`" + `
	Email    string ` + "`json:\"email\"`" + `
	Age      int    ` + "`json:\"age\" gql:\"omit:UserInput\"`" + `
}

/**
 * @gqlType
 */
type UserPayload struct {
	Payload[*User]
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
	cfg.AutoGenerate.Enabled = true
	cfg.AutoGenerate.Strategy = AutoGenReferenced
	cfg.AutoGenerate.MaxDepth = 2

	gen := NewGenerator(parser, cfg)
	if err := gen.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	generated, err := os.ReadFile(cfg.Output)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	schema := string(generated)

	shouldContain := []string{
		"type UserPayload",
		"data:",
		"success:",
		"type User",
		"username:",
		"email:",
		"age:",
		"input UserInput",
	}

	for _, expected := range shouldContain {
		if !strings.Contains(schema, expected) {
			t.Errorf("Schema missing: %q\n\nGenerated:\n%s", expected, schema)
		}
	}

	// Verify age is excluded from input but present in type
	lines := strings.Split(schema, "\n")
	inInputBlock := false
	inputHasAge := false

	for _, line := range lines {
		if strings.Contains(line, "input UserInput") {
			inInputBlock = true
		}
		if inInputBlock && strings.TrimSpace(line) == "}" {
			inInputBlock = false
		}
		if inInputBlock && strings.Contains(line, "age:") {
			inputHasAge = true
		}
	}

	if inputHasAge {
		t.Error("UserInput should not contain 'age' field due to omit directive")
	}
}
