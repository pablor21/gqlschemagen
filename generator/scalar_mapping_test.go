package generator

import (
	"testing"
)

func TestScalarMapping(t *testing.T) {
	config := &Config{
		Scalars: map[string]ScalarMapping{
			"ID": {
				Model: []string{
					"github.com/google/uuid.UUID",
					"github.com/gofrs/uuid.UUID",
				},
			},
			"DateTime": {
				Model: []string{
					"time.Time",
				},
			},
		},
	}

	tests := []struct {
		name         string
		goTypePath   string
		expectedName string
	}{
		{
			name:         "Google UUID maps to ID",
			goTypePath:   "github.com/google/uuid.UUID",
			expectedName: "ID",
		},
		{
			name:         "Gofrs UUID maps to ID",
			goTypePath:   "github.com/gofrs/uuid.UUID",
			expectedName: "ID",
		},
		{
			name:         "time.Time maps to DateTime",
			goTypePath:   "time.Time",
			expectedName: "DateTime",
		},
		{
			name:         "Unmapped type returns empty",
			goTypePath:   "github.com/example/CustomType",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetScalarForGoType(tt.goTypePath)
			if result != tt.expectedName {
				t.Errorf("GetScalarForGoType(%q) = %q, want %q", tt.goTypePath, result, tt.expectedName)
			}
		})
	}
}

func TestIsBuiltInScalar(t *testing.T) {
	tests := []struct {
		name       string
		scalarName string
		expected   bool
	}{
		{"Int is built-in", "Int", true},
		{"Float is built-in", "Float", true},
		{"String is built-in", "String", true},
		{"Boolean is built-in", "Boolean", true},
		{"ID is built-in", "ID", true},
		{"DateTime is custom", "DateTime", false},
		{"UUID is custom", "UUID", false},
		{"JSON is custom", "JSON", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltInScalar(tt.scalarName)
			if result != tt.expected {
				t.Errorf("IsBuiltInScalar(%q) = %v, want %v", tt.scalarName, result, tt.expected)
			}
		})
	}
}

func TestGetUsedCustomScalars(t *testing.T) {
	config := &Config{
		KnownScalars: []string{"DateTime", "Upload"},
		Scalars: map[string]ScalarMapping{
			"ID": {
				Model: []string{"github.com/google/uuid.UUID"},
			},
			"DateTime": {
				Model: []string{"time.Time"},
			},
			"Upload": {
				Model: []string{"github.com/99designs/gqlgen/graphql.Upload"},
			},
			"CustomScalar": {
				Model: []string{"github.com/example/pkg.CustomType"},
			},
			"Int": {
				Model: []string{"int64"},
			},
		},
	}

	customScalars := config.GetUsedCustomScalars()

	// Should only return CustomScalar
	// - ID and Int are built-in GraphQL scalars
	// - DateTime and Upload are in known_scalars
	expectedCount := 1 // Only CustomScalar
	if len(customScalars) != expectedCount {
		t.Errorf("GetUsedCustomScalars() returned %d scalars, want %d. Got: %v", len(customScalars), expectedCount, customScalars)
	}

	// Check that CustomScalar is in the list
	found := false
	for _, scalar := range customScalars {
		if scalar == "CustomScalar" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GetUsedCustomScalars() did not include 'CustomScalar'")
	}

	// Check that built-in scalars and known_scalars are not included
	for _, scalar := range customScalars {
		if scalar == "ID" || scalar == "Int" {
			t.Errorf("GetUsedCustomScalars() should not include built-in scalar %q", scalar)
		}
		if scalar == "DateTime" || scalar == "Upload" {
			t.Errorf("GetUsedCustomScalars() should not include known_scalar %q", scalar)
		}
	}
}
