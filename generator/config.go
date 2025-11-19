package generator

import "fmt"

// FieldCase determines how field names are formatted
type FieldCase string

const (
	FieldCaseCamel    FieldCase = "camel"
	FieldCaseSnake    FieldCase = "snake"
	FieldCasePascal   FieldCase = "pascal"
	FieldCaseOriginal FieldCase = "original"
	FieldCaseNone     FieldCase = "none" // Keep struct field name untouched
)

// GenStrategy determines output file strategy
type GenStrategy string

const (
	GenStrategySingle   GenStrategy = "single"
	GenStrategyMultiple GenStrategy = "multiple"
)

// Config controls how the schema generator behaves
type Config struct {
	// Packages to scan for Go structs (supports glob: /models/**/*.go)
	Packages []string `yaml:"packages"`

	// Output directory or file path
	Output string `yaml:"output"`

	// Field name case transformation (camel, snake, pascal)
	FieldCase FieldCase `yaml:"field_case"`

	// Use json struct tag for field names if gql tag is not present
	UseJsonTag bool `yaml:"use_json_tag"`

	// Generate gqlgen directives (@goModel, @goField, @goTag)
	UseGqlGenDirectives bool `yaml:"use_gqlgen_directives"`

	// Base path for @goModel directive (e.g., "github.com/user/project/models")
	// If empty, uses the actual package path
	ModelPath string `yaml:"model_path"`

	// StripPrefix is a comma-separated list of prefixes to strip from type names
	// e.g. "DB,Pg" will convert "DBUser" to "User" and "PgPost" to "Post"
	// Only applies when @gqlType or @gqlInput doesn't specify a custom name
	StripPrefix string `yaml:"strip_prefix"`

	// StripSuffix is a comma-separated list of suffixes to strip from type names
	// e.g. "DTO,Entity,Model" will convert "UserDTO" to "User" and "PostEntity" to "Post"
	// Only applies when @gqlType or @gqlInput doesn't specify a custom name
	StripSuffix string `yaml:"strip_suffix"`

	// AddTypePrefix is a prefix to add to GraphQL type names
	// e.g. "Gql" will convert "User" to "GqlUser"
	// Only applies when @gqlType doesn't specify a custom name
	AddTypePrefix string `yaml:"add_type_prefix"`

	// AddTypeSuffix is a suffix to add to GraphQL type names
	// e.g. "Type" will convert "User" to "UserType"
	// Only applies when @gqlType doesn't specify a custom name
	AddTypeSuffix string `yaml:"add_type_suffix"`

	// AddInputPrefix is a prefix to add to GraphQL input names
	// e.g. "Gql" will convert "UserInput" to "GqlUserInput"
	// Only applies when @gqlInput doesn't specify a custom name
	AddInputPrefix string `yaml:"add_input_prefix"`

	// AddInputSuffix is a suffix to add to GraphQL input names
	// e.g. "Payload" will convert "CreateUser" to "CreateUserPayload"
	// Only applies when @gqlInput doesn't specify a custom name
	AddInputSuffix string `yaml:"add_input_suffix"`

	// Generation strategy: single file or multiple files
	GenStrategy GenStrategy `yaml:"strategy"`

	// Schema file name pattern (for multiple strategy)
	// Supports: {model_name}, {type_name}
	// Default: "{model_name}.graphqls"
	SchemaFileName string `yaml:"schema_file_name"`

	// Include empty types (types with no fields)
	IncludeEmptyTypes bool `yaml:"include_empty_types"`

	// Skip existing files
	SkipExisting bool `yaml:"skip_existing"`

	// Generate inputs automatically
	GenInputs bool `yaml:"gen_inputs"`

	// Generate empty structs
	GenerateEmptyStructs bool `yaml:"generate_empty_structs"`

	// GQLKeep preserved sections marker
	KeepBeginMarker      string `yaml:"keep_begin_marker"`
	KeepEndMarker        string `yaml:"keep_end_marker"`
	KeepSectionPlacement string `yaml:"keep_section_placement"`
}

// NewConfig creates a new Config with defaults
func NewConfig() *Config {
	return &Config{
		FieldCase:           FieldCaseCamel,
		UseJsonTag:          true,
		UseGqlGenDirectives: false,
		GenStrategy:         GenStrategyMultiple,
		SchemaFileName:      "{model_name}.graphqls",
		IncludeEmptyTypes:   false,
	}
}

// Normalize ensures config values are valid
func (c *Config) Normalize() {
	if c.GenStrategy == "" {
		c.GenStrategy = GenStrategyMultiple
	}

	// Set defaults
	if c.FieldCase == "" {
		c.FieldCase = FieldCaseCamel
	}
	if c.SchemaFileName == "" {
		c.SchemaFileName = "{model_name}.graphqls"
	}

	if c.KeepBeginMarker == "" {
		c.KeepBeginMarker = "# @gqlKeepBegin"
	}
	if c.KeepEndMarker == "" {
		c.KeepEndMarker = "# @gqlKeepEnd"
	}

	if c.KeepSectionPlacement == "" {
		c.KeepSectionPlacement = "end"
	}

}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate required fields
	if len(c.Packages) == 0 {
		return fmt.Errorf("packages is required (at least one package path must be specified)")
	}

	// Validate strategy
	if c.GenStrategy != "" && c.GenStrategy != GenStrategySingle && c.GenStrategy != GenStrategyMultiple {
		return fmt.Errorf("invalid strategy: %s (must be 'single' or 'multiple')", c.GenStrategy)
	}

	// Validate field case
	if c.FieldCase != "" && c.FieldCase != FieldCaseCamel && c.FieldCase != FieldCaseSnake &&
		c.FieldCase != FieldCasePascal && c.FieldCase != FieldCaseOriginal && c.FieldCase != FieldCaseNone {
		return fmt.Errorf("invalid field-case: %s (must be 'camel', 'snake', 'pascal', 'original', or 'none')", c.FieldCase)
	}

	if c.KeepSectionPlacement != "start" && c.KeepSectionPlacement != "end" {
		return fmt.Errorf("invalid keep_section_placement: %s (must be 'start' or 'end')", c.KeepSectionPlacement)
	}

	return nil
}
