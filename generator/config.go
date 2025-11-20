package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var cfgFilenames = []string{".gqlschemagen.yml", "gqlschemagen.yml", "gqlschemagen.yaml"}

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
	GenStrategyPackage  GenStrategy = "package"
)

// Config controls how the schema generator behaves
type Config struct {
	// Packages to scan for Go structs (supports glob: /models/**/*.go)
	Packages []string `yaml:"packages"`

	// Output directory or file path
	Output string `yaml:"output"`

	// Output file name (for single strategy, default: "gqlschemagen.graphqls")
	OutputFileName string `yaml:"output_file_name"`

	// Output file extension (for multiple/package strategies, default: ".graphqls")
	OutputFileExtension string `yaml:"output_file_extension"`

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

	// Namespace separator for converting namespace to file paths
	// Default: "/" (e.g., "user.auth" becomes "user/auth.graphqls")
	NamespaceSeparator string `yaml:"namespace_separator"`

	// ConfigDir is the directory where the config file was loaded from.
	// This is used to resolve relative paths in the config.
	// Not marshaled to/from YAML.
	ConfigDir string `yaml:"-"`
}

// DetectGoModulePath searches for go.mod in current directory and parent directories,
// and returns the module path. Returns empty string if go.mod is not found.
func DetectGoModulePath() string {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Search up to root
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, parse it to get module path
			if modulePath := parseGoModModulePath(goModPath); modulePath != "" {
				return modulePath
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

// parseGoModModulePath extracts the module path from go.mod file
func parseGoModModulePath(goModPath string) string {
	file, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract module path (everything after "module ")
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			return modulePath
		}
	}

	return ""
}

// NewConfig creates a new Config with defaults
func NewConfig() *Config {
	return &Config{
		FieldCase:           FieldCaseCamel,
		UseJsonTag:          true,
		UseGqlGenDirectives: false,
		GenStrategy:         GenStrategyMultiple,
		SchemaFileName:      "{model_name}.graphqls",
		OutputFileName:      "gqlschemagen.graphqls",
		OutputFileExtension: ".graphqls",
		IncludeEmptyTypes:   false,
		NamespaceSeparator:  "/",
	}
}

// NewConfigWithDefaults creates a new Config with smart defaults based on the current environment.
// It auto-detects the Go module path and sets packages to current directory.
func NewConfigWithDefaults() *Config {
	cfg := NewConfig()

	// Set current directory as ConfigDir
	if cwd, err := os.Getwd(); err == nil {
		cfg.ConfigDir = cwd
	}

	// Auto-detect module path from go.mod
	if modulePath := DetectGoModulePath(); modulePath != "" {
		cfg.ModelPath = modulePath
	}

	// Set packages to current directory
	cfg.Packages = []string{"./"}

	return cfg
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
	if c.OutputFileName == "" {
		c.OutputFileName = "gqlschemagen.graphqls"
	}
	if c.OutputFileExtension == "" {
		c.OutputFileExtension = ".graphqls"
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

	if c.NamespaceSeparator == "" {
		c.NamespaceSeparator = "/"
	}

	// Note: normalizeOutputPath() removed - backward compatibility is now handled
	// in generateSingleFile() by detecting file extensions in the Output path
}

// // normalizeOutputPath provides backward compatibility for old config format
// // If Output contains a file (has extension), split it into directory and filename
// func (c *Config) normalizeOutputPath() {
// 	if c.Output == "" {
// 		return
// 	}

// 	// Check if Output looks like a file path (contains a file extension)
// 	// by checking if the last component has a dot and extension
// 	lastSlash := -1
// 	for i := len(c.Output) - 1; i >= 0; i-- {
// 		if c.Output[i] == '/' || c.Output[i] == '\\' {
// 			lastSlash = i
// 			break
// 		}
// 	}

// 	lastPart := c.Output
// 	if lastSlash >= 0 {
// 		lastPart = c.Output[lastSlash+1:]
// 	}

// 	// If the last part contains a dot and looks like a file (e.g., "schema.graphqls")
// 	// and OutputFileName wasn't explicitly set in YAML, extract it
// 	if dotIndex := -1; func() bool {
// 		for i := len(lastPart) - 1; i >= 0; i-- {
// 			if lastPart[i] == '.' {
// 				dotIndex = i
// 				return true
// 			}
// 		}
// 		return false
// 	}() && dotIndex > 0 {
// 		// This looks like a file path, extract directory and filename
// 		dir := c.Output[:lastSlash+1]
// 		if lastSlash < 0 {
// 			dir = "./"
// 		}
// 		filename := lastPart

// 		// Only override if OutputFileName is still at default value
// 		if c.OutputFileName == "gqlschemagen.graphqls" {
// 			c.OutputFileName = filename
// 			c.Output = dir
// 		}
// 	}
// }

func findCfgInDir(dir string) string {
	for _, cfgName := range cfgFilenames {
		path := filepath.Join(dir, cfgName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// FindConfig searches for the config file in this directory and all parents up the tree
// looking for the closest match. Returns the path to the config file or empty string if not found.
func FindConfig() string {
	path, _ := findCfg()
	return path
}

// findCfg searches for the config file in this directory and all parents up the tree
// looking for the closest match
func findCfg() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get working dir to findCfg: %w", err)
	}

	cfg := findCfgInDir(dir)

	for cfg == "" && dir != filepath.Dir(dir) {
		dir = filepath.Dir(dir)
		cfg = findCfgInDir(dir)
	}

	if cfg == "" {
		return "", os.ErrNotExist
	}

	return cfg, nil
}

// LoadConfig attempts to find and load a config file automatically.
// It searches for config files (.gqlschemagen.yml, gqlschemagen.yml, gqlschemagen.yaml)
// in the current directory and parent directories.
// Returns a config with defaults if no config file is found.
func LoadConfig() (*Config, error) {
	cfgPath, err := findCfg()
	if err != nil {
		// No config file found, return default config
		cfg := NewConfig()
		cfg.Normalize()
		return cfg, nil
	}

	return LoadConfigFromFile(cfgPath)
}

// LoadConfigFromFile loads a config from the specified file path.
// It stores the config file's directory in ConfigDir and resolves all relative paths
// in the config relative to that directory.
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Store the directory where config was found
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for config: %w", err)
	}
	cfg.ConfigDir = filepath.Dir(absPath)

	// Resolve all relative paths in the config relative to ConfigDir
	cfg.resolveRelativePaths()

	cfg.Normalize()
	return &cfg, nil
}

// resolveRelativePaths converts all relative paths in the config to be relative to ConfigDir.
// This ensures that paths work correctly when the config is loaded from a different directory
// than where the command is run (e.g., with //go:generate).
func (c *Config) resolveRelativePaths() {
	if c.ConfigDir == "" {
		return
	}

	// Resolve package paths
	for i, pkg := range c.Packages {
		if !filepath.IsAbs(pkg) {
			c.Packages[i] = filepath.Join(c.ConfigDir, pkg)
		}
	}

	// Resolve output path
	if c.Output != "" && !filepath.IsAbs(c.Output) {
		c.Output = filepath.Join(c.ConfigDir, c.Output)
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate required fields
	if len(c.Packages) == 0 {
		return fmt.Errorf("packages is required (at least one package path must be specified)")
	}

	// Validate strategy
	if c.GenStrategy != "" && c.GenStrategy != GenStrategySingle && c.GenStrategy != GenStrategyMultiple && c.GenStrategy != GenStrategyPackage {
		return fmt.Errorf("invalid strategy: %s (must be 'single', 'multiple', or 'package')", c.GenStrategy)
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
