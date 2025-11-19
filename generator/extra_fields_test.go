package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// parseGoCode parses Go source code and populates the parser
func parseGoCode(p *Parser, filename string, code string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, code, parser.ParseComments)
	if err != nil {
		return err
	}

	pkgName := f.Name.Name

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			s, ok := t.Type.(*ast.StructType)
			if !ok {
				continue
			}
			name := t.Name.Name
			p.StructTypes[name] = t
			p.Structs[name] = s
			p.PackageNames[name] = pkgName
			p.TypeToDecl[name] = genDecl
			p.TypeNames = append(p.TypeNames, name)
		}
	}
	return nil
}

func TestTypeExtraField(t *testing.T) {
	tests := []struct {
		name      string
		goCode    string
		wantType  string
		wantInput string
	}{
		{
			name: "basic type extra field",
			goCode: `package test
/**
 * @gqlType(name:"User")
 * @gqlInput(name:"UserInput")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",description:"User posts")
 */
type User struct {
	ID string
}`,
			wantType: `type User {
	id: String!
	"""User posts"""
	posts: [Post!]!
}`,
			wantInput: `input UserInput {
	id: String!
}`,
		},
		{
			name: "input extra field",
			goCode: `package test
/**
 * @gqlType(name:"User")
 * @gqlInput(name:"UserInput")
 * @gqlInputExtraField(name:"password",type:"String!",description:"Password")
 */
type User struct {
	ID string
}`,
			wantType: `type User {
	id: String!
}`,
			wantInput: `input UserInput {
	id: String!
	"""Password"""
	password: String!
}`,
		},
		{
			name: "extra field with on parameter",
			goCode: `package test
/**
 * @gqlType(name:"Article")
 * @gqlType(name:"BlogPost")
 * @gqlTypeExtraField(name:"author",type:"User",description:"Author",on:"Article")
 * @gqlTypeExtraField(name:"writer",type:"User",description:"Writer",on:"BlogPost")
 */
type Content struct {
	ID string
}`,
			wantType: `type Article {
	id: String!
	"""Author"""
	author: User
}

type BlogPost {
	id: String!
	"""Writer"""
	writer: User
}`,
		},
		{
			name: "multiple on targets",
			goCode: `package test
/**
 * @gqlType(name:"Order")
 * @gqlType(name:"Invoice")
 * @gqlType(name:"Receipt")
 * @gqlTypeExtraField(name:"customer",type:"User!",description:"Customer",on:"Order,Invoice")
 */
type Transaction struct {
	ID string
}`,
			wantType: `type Order {
	id: String!
	"""Customer"""
	customer: User!
}

type Invoice {
	id: String!
	"""Customer"""
	customer: User!
}

type Receipt {
	id: String!
}`,
		},
		{
			name: "input extra field with on parameter",
			goCode: `package test
/**
 * @gqlInput(name:"CreateProductInput")
 * @gqlInput(name:"UpdateProductInput")
 * @gqlInputExtraField(name:"vendorId",type:"ID!",description:"Vendor",on:"CreateProductInput")
 */
type Product struct {
	ID string
}`,
			wantInput: `input CreateProductInput {
	id: String!
	"""Vendor"""
	vendorId: ID!
}

input UpdateProductInput {
	id: String!
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			if err := parseGoCode(parser, "test.go", tt.goCode); err != nil {
				t.Fatalf("Failed to parse Go code: %v", err)
			}

			cfg := NewConfig()
			cfg.FieldCase = FieldCaseCamel
			cfg.UseJsonTag = false

			gen := NewGenerator(parser, cfg)

			var result strings.Builder
			for _, typeName := range parser.TypeNames {
				typeSpec := parser.StructTypes[typeName]
				d := ParseDirectives(typeSpec, parser.TypeToDecl[typeName])

				// Generate types
				if d.HasTypeDirective {
					for _, typeDef := range d.Types {
						typeContent := gen.generateTypeFromDef(typeSpec, parser.Structs[typeName], d, typeDef)
						result.WriteString(typeContent)
					}
				}

				// Generate inputs
				if d.HasInputDirective {
					for _, inputDef := range d.Inputs {
						inputContent := gen.generateInputFromDef(typeSpec, parser.Structs[typeName], d, inputDef)
						result.WriteString(inputContent)
					}
				}
			}

			output := result.String()

			if tt.wantType != "" && !containsNormalized(output, tt.wantType) {
				t.Errorf("Expected type output not found.\nWant:\n%s\n\nGot:\n%s", tt.wantType, output)
			}

			if tt.wantInput != "" && !containsNormalized(output, tt.wantInput) {
				t.Errorf("Expected input output not found.\nWant:\n%s\n\nGot:\n%s", tt.wantInput, output)
			}
		})
	}
}

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

// Helper functions
func containsNormalized(s, substr string) bool {
	s = normalizeWhitespace(s)
	substr = normalizeWhitespace(substr)
	return strings.Contains(s, substr)
}

func normalizeWhitespace(s string) string {
	// Remove excessive whitespace while preserving structure
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}
