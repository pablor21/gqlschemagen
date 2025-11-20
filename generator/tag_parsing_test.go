package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestComplexTagParsing tests parsing of complex gql tags with multiple list parameters
func TestComplexTagParsing(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		wantName    string
		wantInclude []string
		wantIgnore  []string
	}{
		{
			name:        "include and omit lists",
			tag:         `gql:"internal_id,include:\"PublicView,OtherView\",omit:\"UserInput,OtherInput\""`,
			wantName:    "internal_id",
			wantInclude: []string{"PublicView", "OtherView"},
			wantIgnore:  []string{"UserInput", "OtherInput"},
		},
		{
			name:        "include with single quotes",
			tag:         `gql:"field,include:'TypeA,TypeB',omit:'InputA,InputB'"`,
			wantName:    "field",
			wantInclude: []string{"TypeA", "TypeB"},
			wantIgnore:  []string{"InputA", "InputB"},
		},
		{
			name:        "include with square brackets",
			tag:         `gql:"field,include:[TypeA,TypeB],omit:[InputA,InputB]"`,
			wantName:    "field",
			wantInclude: []string{"TypeA", "TypeB"},
			wantIgnore:  []string{"InputA", "InputB"},
		},
		{
			name:        "multiple includes then omit",
			tag:         `gql:"field,include:\"TypeA,TypeB,TypeC\",omit:\"InputA,InputB\""`,
			wantName:    "field",
			wantInclude: []string{"TypeA", "TypeB", "TypeC"},
			wantIgnore:  []string{"InputA", "InputB"},
		},
		{
			name:        "include then ignore (alias for omit)",
			tag:         `gql:"data,include:\"Admin,Super\",ignore:Public"`,
			wantName:    "data",
			wantInclude: []string{"Admin", "Super"},
			wantIgnore:  []string{"Public"},
		},
		{
			name:        "ro and wo with lists",
			tag:         `gql:"secret,ro:AdminView,SuperView,wo:CreateInput"`,
			wantName:    "secret",
			wantInclude: nil,
			wantIgnore:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock field with the tag
			src := `package test
type Test struct {
	Field string ` + "`" + tt.tag + "`" + `
}`
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", src, 0)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Extract the field
			var field *ast.Field
			for _, decl := range f.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								field = structType.Fields.List[0]
							}
						}
					}
				}
			}

			if field == nil {
				t.Fatal("Failed to extract field")
			}

			// Parse the field options
			cfg := &Config{}
			opts := ParseFieldOptions(field, cfg)

			// Verify name
			if opts.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", opts.Name, tt.wantName)
			}

			// Verify include list
			if tt.wantInclude != nil {
				if len(opts.IncludeList) != len(tt.wantInclude) {
					t.Errorf("IncludeList length = %d, want %d. Got: %v", len(opts.IncludeList), len(tt.wantInclude), opts.IncludeList)
				} else {
					for i, want := range tt.wantInclude {
						if i >= len(opts.IncludeList) || opts.IncludeList[i] != want {
							t.Errorf("IncludeList[%d] = %q, want %q", i, opts.IncludeList[i], want)
						}
					}
				}
			}

			// Verify ignore list
			if tt.wantIgnore != nil {
				if len(opts.IgnoreList) != len(tt.wantIgnore) {
					t.Errorf("IgnoreList length = %d, want %d. Got: %v", len(opts.IgnoreList), len(tt.wantIgnore), opts.IgnoreList)
				} else {
					for i, want := range tt.wantIgnore {
						if i >= len(opts.IgnoreList) || opts.IgnoreList[i] != want {
							t.Errorf("IgnoreList[%d] = %q, want %q", i, opts.IgnoreList[i], want)
						}
					}
				}
			}
		})
	}
}

// TestSplitParamsWithLists tests the low-level splitting function
func TestSplitParamsWithLists(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple parameters",
			input: "name,optional,required",
			want:  []string{"name", "optional", "required"},
		},
		{
			name:  "single list quoted",
			input: `name,include:"TypeA,TypeB"`,
			want:  []string{"name", `include:"TypeA,TypeB"`},
		},
		{
			name:  "single list single quotes",
			input: `name,include:'TypeA,TypeB'`,
			want:  []string{"name", `include:'TypeA,TypeB'`},
		},
		{
			name:  "single list square brackets",
			input: `name,include:[TypeA,TypeB]`,
			want:  []string{"name", `include:[TypeA,TypeB]`},
		},
		{
			name:  "multiple lists quoted",
			input: `internal_id,include:"PublicView,OtherView",omit:"UserInput,OtherInput"`,
			want:  []string{"internal_id", `include:"PublicView,OtherView"`, `omit:"UserInput,OtherInput"`},
		},
		{
			name:  "multiple lists single quotes",
			input: `internal_id,include:'PublicView,OtherView',omit:'UserInput,OtherInput'`,
			want:  []string{"internal_id", `include:'PublicView,OtherView'`, `omit:'UserInput,OtherInput'`},
		},
		{
			name:  "multiple lists square brackets",
			input: `internal_id,include:[PublicView,OtherView],omit:[UserInput,OtherInput]`,
			want:  []string{"internal_id", `include:[PublicView,OtherView]`, `omit:[UserInput,OtherInput]`},
		},
		{
			name:  "mixed quote styles",
			input: `field,include:"A,B",omit:'C,D',ro:[E,F]`,
			want:  []string{"field", `include:"A,B"`, `omit:'C,D'`, `ro:[E,F]`},
		},
		{
			name:  "list then flag",
			input: `field,include:"A,B",optional`,
			want:  []string{"field", `include:"A,B"`, "optional"},
		},
		{
			name:  "flag then list",
			input: `field,optional,include:"A,B"`,
			want:  []string{"field", "optional", `include:"A,B"`},
		},
		{
			name:  "single value without quotes",
			input: "field,optional,include:SingleType",
			want:  []string{"field", "optional", "include:SingleType"},
		},
		{
			name:  "quoted value with comma",
			input: `name,description:"Hello, world",type:String`,
			want:  []string{"name", `description:"Hello, world"`, "type:String"},
		},
		{
			name:  "complex mixed",
			input: `id,type:ID,ro:"AdminView,UserView",description:"User ID",required`,
			want:  []string{"id", "type:ID", `ro:"AdminView,UserView"`, `description:"User ID"`, "required"},
		},
		{
			name:  "three lists in a row",
			input: `field,include:"A,B",omit:"C,D",ro:"E,F"`,
			want:  []string{"field", `include:"A,B"`, `omit:"C,D"`, `ro:"E,F"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitParamsWithLists(tt.input)

			if len(got) != len(tt.want) {
				t.Errorf("splitParamsWithLists() length = %d, want %d\nGot:  %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitParamsWithLists()[%d] = %q, want %q\nFull result: %v",
						i, got[i], tt.want[i], got)
				}
			}
		})
	}
}
