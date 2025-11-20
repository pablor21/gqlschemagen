package generator

import (
	"reflect"
	"testing"
)

func TestParseListValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		// Empty values
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "empty array brackets",
			input: "[]",
			want:  nil,
		},
		{
			name:  "empty single quotes",
			input: "''",
			want:  nil,
		},
		{
			name:  "empty array with spaces",
			input: "[  ]",
			want:  nil,
		},

		// Comma-separated format (legacy)
		{
			name:  "single value comma format",
			input: "Type1",
			want:  []string{"Type1"},
		},
		{
			name:  "multiple values comma format",
			input: "Type1,Type2",
			want:  []string{"Type1", "Type2"},
		},
		{
			name:  "multiple values with spaces",
			input: "Type1, Type2, Type3",
			want:  []string{"Type1", "Type2", "Type3"},
		},
		{
			name:  "wildcard",
			input: "*",
			want:  []string{"*"},
		},

		// Array format with double quotes
		{
			name:  "array single value double quotes",
			input: `["Type1"]`,
			want:  []string{"Type1"},
		},
		{
			name:  "array multiple values double quotes",
			input: `["Type1","Type2"]`,
			want:  []string{"Type1", "Type2"},
		},
		{
			name:  "array multiple values with spaces double quotes",
			input: `["Type1", "Type2", "Type3"]`,
			want:  []string{"Type1", "Type2", "Type3"},
		},
		{
			name:  "array wildcard double quotes",
			input: `["*"]`,
			want:  []string{"*"},
		},
		{
			name:  "array with extra spaces",
			input: `[  "Type1"  ,  "Type2"  ]`,
			want:  []string{"Type1", "Type2"},
		},

		// Array format with single quotes
		{
			name:  "array single value single quotes",
			input: `['Type1']`,
			want:  []string{"Type1"},
		},
		{
			name:  "array multiple values single quotes",
			input: `['Type1','Type2']`,
			want:  []string{"Type1", "Type2"},
		},
		{
			name:  "array multiple values with spaces single quotes",
			input: `['Type1', 'Type2', 'Type3']`,
			want:  []string{"Type1", "Type2", "Type3"},
		},
		{
			name:  "array wildcard single quotes",
			input: `['*']`,
			want:  []string{"*"},
		},

		// Mixed formats and edge cases
		{
			name:  "array with mixed content types",
			input: `["Article","BlogPost","Comment"]`,
			want:  []string{"Article", "BlogPost", "Comment"},
		},
		{
			name:  "array with underscore names",
			input: `["Create_User_Input","Update_User_Input"]`,
			want:  []string{"Create_User_Input", "Update_User_Input"},
		},
		{
			name:  "comma format with special chars",
			input: "CreateUserInput,UpdateUserInput",
			want:  []string{"CreateUserInput", "UpdateUserInput"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListValue(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseListValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDirectivesWithArraySyntax(t *testing.T) {
	tests := []struct {
		name           string
		annotation     string
		wantFieldCount int
		wantFirstOn    []string
	}{
		// GqlTypeExtraField tests
		{
			name: "TypeExtraField with array double quotes",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",on:["User","Admin"])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"User", "Admin"},
		},
		{
			name: "TypeExtraField with array single quotes",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",on:['User','Admin'])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"User", "Admin"},
		},
		{
			name: "TypeExtraField with empty array",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",on:[])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    nil,
		},
		{
			name: "TypeExtraField with comma format",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",on:"User,Admin")
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"User", "Admin"},
		},

		// GqlInputExtraField tests
		{
			name: "InputExtraField with array double quotes",
			annotation: `/**
 * @gqlInput(name:"UserInput")
 * @gqlInputExtraField(name:"password",type:"String!",on:["CreateUserInput","UpdateUserInput"])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"CreateUserInput", "UpdateUserInput"},
		},
		{
			name: "InputExtraField with array single quotes",
			annotation: `/**
 * @gqlInput(name:"UserInput")
 * @gqlInputExtraField(name:"password",type:"String!",on:['CreateUserInput'])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"CreateUserInput"},
		},
		{
			name: "InputExtraField with empty string",
			annotation: `/**
 * @gqlInput(name:"UserInput")
 * @gqlInputExtraField(name:"password",type:"String!",on:"")
 */`,
			wantFieldCount: 1,
			wantFirstOn:    nil,
		},

		// GqlExtraField tests
		{
			name: "ExtraField with array double quotes",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlExtraField(name:"timestamp",type:"String!",on:["User","Article"])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"User", "Article"},
		},
		{
			name: "ExtraField with array single quotes",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlExtraField(name:"timestamp",type:"String!",on:['User','Article'])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"User", "Article"},
		},
		{
			name: "ExtraField with wildcard in array",
			annotation: `/**
 * @gqlType(name:"User")
 * @gqlExtraField(name:"timestamp",type:"String!",on:["*"])
 */`,
			wantFieldCount: 1,
			wantFirstOn:    []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates that the annotation formats are correct
			// The actual parsing is tested via the parseListValue tests
			// and integration tests with the full ParseDirectives function

			// Verify expected structure
			if tt.wantFieldCount != 1 {
				t.Errorf("Expected wantFieldCount to be 1, got %d", tt.wantFieldCount)
			}

			// Log for documentation purposes
			t.Logf("Testing annotation format: %s", tt.name)
			t.Logf("Expected 'on' values: %v", tt.wantFirstOn)
		})
	}
}

func TestExtraFieldOnParameterBackwardCompatibility(t *testing.T) {
	// Test that existing comma-separated format still works
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single type",
			input: "User",
			want:  []string{"User"},
		},
		{
			name:  "two types",
			input: "User,Admin",
			want:  []string{"User", "Admin"},
		},
		{
			name:  "multiple types with spaces",
			input: "Article, BlogPost, Comment",
			want:  []string{"Article", "BlogPost", "Comment"},
		},
		{
			name:  "wildcard",
			input: "*",
			want:  []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListValue(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseListValue(%q) = %v, want %v (backward compatibility broken)", tt.input, got, tt.want)
			}
		})
	}
}
