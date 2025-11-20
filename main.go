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
//	gqlschemagen generate --watch                              # Watch for changes and regenerate
//
// For more information and examples, visit: https://github.com/pablor21/gqlschemagen
package main

import (
	"embed"

	"github.com/pablor21/gqlschemagen/cmd"
)

//go:embed gqlschemagen.yml
var DefaultConfig embed.FS

func main() {
	cmd.Execute(DefaultConfig)
}
