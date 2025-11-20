package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestParseDirectivesWithArraySyntaxIntegration(t *testing.T) {
	tests := []struct {
		name           string
		sourceCode     string
		wantTypeExtra  int
		wantInputExtra int
		checkOn        func(t *testing.T, directives StructDirectives)
	}{
		{
			name: "TypeExtraField with array syntax double quotes",
			sourceCode: `package test
/**
 * @gqlType(name:"User")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",on:["User","Admin"])
 */
type User struct {
	ID string
}`,
			wantTypeExtra:  1,
			wantInputExtra: 0,
			checkOn: func(t *testing.T, directives StructDirectives) {
				if len(directives.TypeExtraFields) != 1 {
					t.Fatalf("Expected 1 TypeExtraField, got %d", len(directives.TypeExtraFields))
				}
				ef := directives.TypeExtraFields[0]
				if ef.Name != "posts" {
					t.Errorf("Expected field name 'posts', got %q", ef.Name)
				}
				if len(ef.On) != 2 || ef.On[0] != "User" || ef.On[1] != "Admin" {
					t.Errorf("Expected On to be ['User','Admin'], got %v", ef.On)
				}
			},
		},
		{
			name: "TypeExtraField with array syntax single quotes",
			sourceCode: `package test
/**
 * @gqlType(name:"Article")
 * @gqlTypeExtraField(name:"tags",type:"[String!]!",on:['Article','BlogPost'])
 */
type Content struct {
	ID string
}`,
			wantTypeExtra:  1,
			wantInputExtra: 0,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.TypeExtraFields[0]
				if len(ef.On) != 2 || ef.On[0] != "Article" || ef.On[1] != "BlogPost" {
					t.Errorf("Expected On to be ['Article','BlogPost'], got %v", ef.On)
				}
			},
		},
		{
			name: "InputExtraField with empty array",
			sourceCode: `package test
/**
 * @gqlInput(name:"UserInput")
 * @gqlInputExtraField(name:"password",type:"String!",on:[])
 */
type User struct {
	ID string
}`,
			wantTypeExtra:  0,
			wantInputExtra: 1,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.InputExtraFields[0]
				if ef.On != nil {
					t.Errorf("Expected On to be nil for empty array, got %v", ef.On)
				}
			},
		},
		{
			name: "ExtraField with wildcard in array",
			sourceCode: `package test
/**
 * @gqlType(name:"Task")
 * @gqlExtraField(name:"timestamp",type:"String!",on:["*"])
 */
type Task struct {
	ID string
}`,
			wantTypeExtra:  1,
			wantInputExtra: 1,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.TypeExtraFields[0]
				if len(ef.On) != 1 || ef.On[0] != "*" {
					t.Errorf("Expected On to be ['*'], got %v", ef.On)
				}
			},
		},
		{
			name: "Mixed formats - comma separated still works",
			sourceCode: `package test
/**
 * @gqlType(name:"Event")
 * @gqlTypeExtraField(name:"timestamp",type:"String!",on:"Event,Notification")
 */
type Event struct {
	ID string
}`,
			wantTypeExtra:  1,
			wantInputExtra: 0,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.TypeExtraFields[0]
				if len(ef.On) != 2 || ef.On[0] != "Event" || ef.On[1] != "Notification" {
					t.Errorf("Expected On to be ['Event','Notification'], got %v", ef.On)
				}
			},
		},
		{
			name: "Array with spaces",
			sourceCode: `package test
/**
 * @gqlType(name:"Report")
 * @gqlTypeExtraField(name:"data",type:"String!",on:[ "Report" , "Analysis" ])
 */
type Report struct {
	ID string
}`,
			wantTypeExtra:  1,
			wantInputExtra: 0,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.TypeExtraFields[0]
				if len(ef.On) != 2 || ef.On[0] != "Report" || ef.On[1] != "Analysis" {
					t.Errorf("Expected On to be ['Report','Analysis'], got %v", ef.On)
				}
			},
		},
		{
			name: "Empty string for on parameter",
			sourceCode: `package test
/**
 * @gqlInput(name:"OrderInput")
 * @gqlInputExtraField(name:"hidden",type:"String",on:"")
 */
type Order struct {
	ID string
}`,
			wantTypeExtra:  0,
			wantInputExtra: 1,
			checkOn: func(t *testing.T, directives StructDirectives) {
				ef := directives.InputExtraFields[0]
				if ef.On != nil {
					t.Errorf("Expected On to be nil for empty string, got %v", ef.On)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.sourceCode, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			// Find the type declaration
			var typeSpec *ast.TypeSpec
			var genDecl *ast.GenDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if gd, ok := n.(*ast.GenDecl); ok {
					genDecl = gd
					if len(gd.Specs) > 0 {
						if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
							typeSpec = ts
							return false
						}
					}
				}
				return true
			})

			if typeSpec == nil {
				t.Fatalf("Failed to find type spec in source")
			}

			directives := ParseDirectives(typeSpec, genDecl)

			if len(directives.TypeExtraFields) != tt.wantTypeExtra {
				t.Errorf("Expected %d TypeExtraFields, got %d", tt.wantTypeExtra, len(directives.TypeExtraFields))
			}

			if len(directives.InputExtraFields) != tt.wantInputExtra {
				t.Errorf("Expected %d InputExtraFields, got %d", tt.wantInputExtra, len(directives.InputExtraFields))
			}

			// Run custom check function
			if tt.checkOn != nil {
				tt.checkOn(t, directives)
			}
		})
	}
}

func TestRealWorldArraySyntaxScenarios(t *testing.T) {
	sourceCode := `package test

/**
 * @gqlType(name:"User")
 * @gqlType(name:"Admin")
 * @gqlInput(name:"CreateUserInput")
 * @gqlInput(name:"UpdateUserInput")
 * @gqlTypeExtraField(name:"permissions",type:"[String!]!",on:["User","Admin"])
 * @gqlTypeExtraField(name:"lastLogin",type:"String",on:['Admin'])
 * @gqlInputExtraField(name:"password",type:"String!",on:["CreateUserInput"])
 * @gqlExtraField(name:"createdAt",type:"String!",on:["*"])
 */
type User struct {
	ID       string
	Username string
	Email    string
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", sourceCode, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	var typeSpec *ast.TypeSpec
	var genDecl *ast.GenDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if gd, ok := n.(*ast.GenDecl); ok {
			genDecl = gd
			if len(gd.Specs) > 0 {
				if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
					typeSpec = ts
					return false
				}
			}
		}
		return true
	})

	if typeSpec == nil {
		t.Fatalf("Failed to find type spec")
	}

	directives := ParseDirectives(typeSpec, genDecl)

	// Should have 3 TypeExtraFields: permissions (for User,Admin), lastLogin (for Admin), createdAt (for *)
	if len(directives.TypeExtraFields) != 3 {
		t.Errorf("Expected 3 TypeExtraFields, got %d", len(directives.TypeExtraFields))
	}

	// Should have 2 InputExtraFields: password (for CreateUserInput), createdAt (for *)
	if len(directives.InputExtraFields) != 2 {
		t.Errorf("Expected 2 InputExtraFields, got %d", len(directives.InputExtraFields))
	}

	// Verify each field
	for _, ef := range directives.TypeExtraFields {
		switch ef.Name {
		case "permissions":
			if len(ef.On) != 2 || ef.On[0] != "User" || ef.On[1] != "Admin" {
				t.Errorf("permissions field: expected On=['User','Admin'], got %v", ef.On)
			}
		case "lastLogin":
			if len(ef.On) != 1 || ef.On[0] != "Admin" {
				t.Errorf("lastLogin field: expected On=['Admin'], got %v", ef.On)
			}
		case "createdAt":
			if len(ef.On) != 1 || ef.On[0] != "*" {
				t.Errorf("createdAt field: expected On=['*'], got %v", ef.On)
			}
		}
	}

	for _, ef := range directives.InputExtraFields {
		switch ef.Name {
		case "password":
			if len(ef.On) != 1 || ef.On[0] != "CreateUserInput" {
				t.Errorf("password field: expected On=['CreateUserInput'], got %v", ef.On)
			}
		case "createdAt":
			if len(ef.On) != 1 || ef.On[0] != "*" {
				t.Errorf("createdAt field: expected On=['*'], got %v", ef.On)
			}
		}
	}
}

func TestArraySyntaxEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		onValue     string
		expectedLen int
		expected    []string
	}{
		{
			name:        "single quote empty",
			onValue:     "''",
			expectedLen: 0,
			expected:    nil,
		},
		{
			name:        "bracket empty",
			onValue:     "[]",
			expectedLen: 0,
			expected:    nil,
		},
		{
			name:        "bracket with only spaces",
			onValue:     "[   ]",
			expectedLen: 0,
			expected:    nil,
		},
		{
			name:        "mixed quotes in array not supported but should not crash",
			onValue:     `["Type1",'Type2']`,
			expectedLen: 2,
			expected:    []string{"Type1", "Type2"},
		},
		{
			name:        "underscore names",
			onValue:     `["Create_User_Input","Update_User_Input"]`,
			expectedLen: 2,
			expected:    []string{"Create_User_Input", "Update_User_Input"},
		},
		{
			name:        "numeric suffixes",
			onValue:     `["UserV1","UserV2","UserV3"]`,
			expectedLen: 3,
			expected:    []string{"UserV1", "UserV2", "UserV3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseListValue(tt.onValue)

			if result == nil && tt.expected == nil {
				return // Both nil, test passes
			}

			if (result == nil && tt.expected != nil) || (result != nil && tt.expected == nil) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				return
			}

			if len(result) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d", tt.expectedLen, len(result))
				return
			}

			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("At index %d: expected %q, got %q", i, tt.expected[i], val)
				}
			}
		})
	}
}
