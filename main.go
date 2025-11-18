// GQLSchemaGen is a tool that scans Go structs and generates GraphQL schema definitions.
//
// It parses Go struct types and generates corresponding GraphQL type and input definitions,
// with support for custom directives, field transformations, and flexible output strategies.
//
// Features:
//   - Opt-in generation with @gqlType and @gqlInput directives
//   - Field name transformations (camelCase, snake_case, PascalCase)
//   - Support for @gqlIgnore, @gqlIgnoreAll directives
//   - Optional @goModel and @goField directives for gqlgen integration
//   - Single or multiple file output strategies
//   - Type name transformations (strip/add prefixes/suffixes)
//   - YAML configuration file support
//
// Usage:
//
//	gqlschemagen init                                          # Create default configuration file
//	gqlschemagen generate --pkg ./models                       # Generate schema from Go structs
//
// For more information and examples, visit: https://github.com/pablor21/gqlschemagen
package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablor21/gqlschemagen/generator"
	"gopkg.in/yaml.v3"
)

//go:embed gqlschemagen.yml
var DefaultConfig embed.FS

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "init":
		initCommand(os.Args[2:])
	case "generate":
		generateCommand(os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "GQLSchemaGen - Generate GraphQL schemas from Go structs\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  gqlschemagen init [options]              Create default configuration file\n")
	fmt.Fprintf(os.Stderr, "  gqlschemagen generate [options]          Generate GraphQL schema from Go structs\n")
	fmt.Fprintf(os.Stderr, "  gqlschemagen help                        Show this help message\n\n")
	fmt.Fprintf(os.Stderr, "Run 'gqlschemagen <command> --help' for more information on a command.\n")
}

func preprocessArgs(args []string) []string {
	// Convert --flag to -flag for Go's flag package
	processed := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "--") {
			processed[i] = "-" + strings.TrimPrefix(arg, "--")
		} else {
			processed[i] = arg
		}
	}
	return processed
}

func initCommand(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gqlschemagen init [options]\n\n")
		fmt.Fprintf(os.Stderr, "Create a default gqlschemagen.yml configuration file in the current directory.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --output, -o <file>    Output file path (default: gqlschemagen.yml)\n")
		fmt.Fprintf(os.Stderr, "  --force, -f            Overwrite existing file\n")
	}

	// Preprocess args
	processedArgs := preprocessArgs(args)

	output := fs.String("output", "gqlschemagen.yml", "output file path")
	fs.StringVar(output, "o", "gqlschemagen.yml", "short for --output")
	force := fs.Bool("force", false, "overwrite existing file")
	fs.BoolVar(force, "f", false, "short for --force")

	fs.Parse(processedArgs)

	// Check if file already exists
	if _, err := os.Stat(*output); err == nil && !*force {
		log.Fatalf("File %s already exists. Use --force to overwrite.", *output)
	}

	// Read embedded default config
	configData, err := DefaultConfig.ReadFile("gqlschemagen.yml")
	if err != nil {
		log.Fatalf("Failed to read default config: %v", err)
	}

	// Write to output file
	if err := os.WriteFile(*output, configData, 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}

	fmt.Printf("Created configuration file: %s\n", *output)
	fmt.Println("Edit this file to customize your schema generation settings.")
}

func generateCommand(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gqlschemagen generate [options]\n\n")
		fmt.Fprintf(os.Stderr, "Generate GraphQL schema from Go structs.\n\n")
		fmt.Fprintf(os.Stderr, "Required flags:\n")
		fmt.Fprintf(os.Stderr, "  --pkg, -p <path>              Root package dir to scan\n\n")
		fmt.Fprintf(os.Stderr, "Optional flags:\n")
		fmt.Fprintf(os.Stderr, "  --config, -f <file>           Path to config file (default: gqlschemagen.yml)\n")
		fmt.Fprintf(os.Stderr, "  --out, -o <path>              Output directory or file path (default: graph/schema)\n")
		fmt.Fprintf(os.Stderr, "  --strategy, -s <strategy>     Generation strategy: single or multiple (default: single)\n")
		fmt.Fprintf(os.Stderr, "  --skip-existing               Skip generating files that already exist\n")
		fmt.Fprintf(os.Stderr, "  --field-case, -c <case>       Field name case: camel, snake, pascal, original, none (default: camel)\n")
		fmt.Fprintf(os.Stderr, "  --use-json-tag                Use json tag for field names (default: true)\n")
		fmt.Fprintf(os.Stderr, "  --gqlgen                      Generate @goModel and @goField directives (default: false)\n")
		fmt.Fprintf(os.Stderr, "  --model-path, -m <path>       Base path for @goModel directive\n")
		fmt.Fprintf(os.Stderr, "  --strip-prefix <prefixes>     Comma-separated prefixes to strip from type names\n")
		fmt.Fprintf(os.Stderr, "  --strip-suffix <suffixes>     Comma-separated suffixes to strip from type names\n")
		fmt.Fprintf(os.Stderr, "  --add-type-prefix <prefix>    Prefix to add to GraphQL type names\n")
		fmt.Fprintf(os.Stderr, "  --add-type-suffix <suffix>    Suffix to add to GraphQL type names\n")
		fmt.Fprintf(os.Stderr, "  --add-input-prefix <prefix>   Prefix to add to GraphQL input names\n")
		fmt.Fprintf(os.Stderr, "  --add-input-suffix <suffix>   Suffix to add to GraphQL input names\n")
		fmt.Fprintf(os.Stderr, "  --schema-file-name <pattern>  Schema file name pattern for multiple mode (default: {model_name}.graphqls)\n")
		fmt.Fprintf(os.Stderr, "  --include-empty-types         Include types with no fields\n")
	}

	// Preprocess args
	processedArgs := preprocessArgs(args)

	// Required flags
	pkg := fs.String("pkg", "", "root package dir to scan (required)")
	fs.StringVar(pkg, "p", "", "short for --pkg")

	// Optional flags
	configFile := fs.String("config", "gqlschemagen.yml", "path to config file")
	fs.StringVar(configFile, "f", "gqlschemagen.yml", "short for --config")

	out := fs.String("out", "", "output directory or file path")
	fs.StringVar(out, "o", "", "short for --out")

	strategy := fs.String("strategy", "single", "generation strategy: single or multiple")
	fs.StringVar(strategy, "s", "single", "short for --strategy")

	skipExisting := fs.Bool("skip-existing", false, "skip generating files that already exist")

	fieldCase := fs.String("field-case", "camel", "field name case: camel, snake, pascal, original, or none")
	fs.StringVar(fieldCase, "c", "camel", "short for --field-case")

	useJsonTag := fs.Bool("use-json-tag", true, "use json tag for field names (priority: gql tag > json tag > struct field)")

	useGqlGenDirectives := fs.Bool("gqlgen", false, "generate @goModel and @goField directives for gqlgen")
	fs.BoolVar(useGqlGenDirectives, "use-gqlgen-directives", false, "long form of --gqlgen")

	modelPath := fs.String("model-path", "", "base path for @goModel directive (e.g., 'github.com/user/project/models')")
	fs.StringVar(modelPath, "m", "", "short for --model-path")

	stripPrefix := fs.String("strip-prefix", "", "comma-separated list of prefixes to strip from type names (e.g., 'DB,Pg')")

	stripSuffix := fs.String("strip-suffix", "", "comma-separated list of suffixes to strip from type names (e.g., 'DTO,Entity,Model')")

	addTypePrefix := fs.String("add-type-prefix", "", "prefix to add to GraphQL type names (unless @gqlType specifies custom name)")

	addTypeSuffix := fs.String("add-type-suffix", "", "suffix to add to GraphQL type names (unless @gqlType specifies custom name)")

	addInputPrefix := fs.String("add-input-prefix", "", "prefix to add to GraphQL input names (unless @gqlInput specifies custom name)")

	addInputSuffix := fs.String("add-input-suffix", "", "suffix to add to GraphQL input names (unless @gqlInput specifies custom name)")

	schemaFileName := fs.String("schema-file-name", "{model_name}.graphqls", "schema file name pattern (multiple mode only)")

	includeEmptyTypes := fs.Bool("include-empty-types", false, "include types with no fields in the schema")

	fs.Parse(processedArgs)

	// Initialize config
	cfg := generator.NewConfig()

	// Load config from YAML file if it exists
	if *configFile != "" {
		if _, err := os.Stat(*configFile); err == nil {
			data, err := os.ReadFile(*configFile)
			if err != nil {
				log.Fatalf("failed to read config file %s: %v", *configFile, err)
			}
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Fatalf("failed to parse config file %s: %v", *configFile, err)
			}
			fmt.Printf("Loaded config from %s\n", *configFile)
		} else if *configFile != "gqlschemagen.yml" {
			// Only error if a non-default config file was specified but not found
			log.Fatalf("config file not found: %s", *configFile)
		}
	}

	// Override config with CLI flags (only if they were explicitly set)
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "pkg", "p":
			cfg.Input = *pkg
		case "out", "o":
			cfg.Output = *out
		case "strategy", "s":
			cfg.GenStrategy = generator.GenStrategy(*strategy)
		case "skip-existing":
			cfg.SkipExisting = *skipExisting
		case "field-case", "c":
			cfg.FieldCase = generator.FieldCase(*fieldCase)
		case "use-json-tag":
			cfg.UseJsonTag = *useJsonTag
		case "gqlgen", "use-gqlgen-directives":
			cfg.UseGqlGenDirectives = *useGqlGenDirectives
		case "model-path", "m":
			cfg.ModelPath = *modelPath
		case "strip-prefix":
			cfg.StripPrefix = *stripPrefix
		case "strip-suffix":
			cfg.StripSuffix = *stripSuffix
		case "add-type-prefix":
			cfg.AddTypePrefix = *addTypePrefix
		case "add-type-suffix":
			cfg.AddTypeSuffix = *addTypeSuffix
		case "add-input-prefix":
			cfg.AddInputPrefix = *addInputPrefix
		case "add-input-suffix":
			cfg.AddInputSuffix = *addInputSuffix
		case "schema-file-name":
			cfg.SchemaFileName = *schemaFileName
		case "include-empty-types":
			cfg.IncludeEmptyTypes = *includeEmptyTypes
		}
	})

	// Set default output based on strategy if not specified
	if cfg.Output == "" {
		if cfg.GenStrategy == generator.GenStrategySingle {
			cfg.Output = "graph/schema/gqlschemagen.graphqls"
		} else {
			cfg.Output = "graph/schema"
		}
	}

	// Clean output path
	cfg.Output = filepath.Clean(cfg.Output)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation error: %v", err)
	}

	parser := generator.NewParser()
	if err := parser.Walk(generator.PkgDir(cfg.Input)); err != nil {
		log.Fatalf("parse error: %v", err)
	}

	engine := generator.NewGenerator(parser, cfg)
	if err := engine.Run(); err != nil {
		log.Fatalf("generation error: %v", err)
	}

	fmt.Println("done")
}
