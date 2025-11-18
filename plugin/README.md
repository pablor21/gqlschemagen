# gqlschemagen Plugin for gqlgen

A gqlgen plugin that automatically generates GraphQL schemas from Go structs with special annotations.

## Installation

```bash
go get github.com/pablor21/gqlschemagen/plugin
```


## Opt-in Generation

**Structs must have `@gqlType()` or `@gqlInput()` directives to be generated:**
- Use `@gqlType()` to generate a GraphQL type
- Use `@gqlInput()` to generate a GraphQL input  
- Structs without these directives are skipped

This gives you precise control over what gets generated from your Go models.

## Usage

### 1. Create a Custom Code Generation Entrypoint

Create a file named `generate.go` in your project root (same directory as `gqlgen.yml`):

```go
//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pablor21/gqlschemagen/plugin"
)

func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}

	// Create and configure the gqlschemagen plugin
	p := plugin.New()
	p.Packages = []string{
		"./graph/models",  // Scan this directory for Go structs
	}

	// Generate code with the plugin
	err = api.Generate(cfg, api.AddPlugin(p))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
```

### 2. Add Go Generate Directive

Add this comment to one of your Go files (e.g., create a `tools.go` file or add to `server.go`):

```go
//go:generate go run generate.go
```

Example `tools.go`:

```go
//go:build tools
// +build tools

package main

//go:generate go run generate.go

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/pablor21/gqlschemagen/plugin"
)
```

### 3. Configure Your `gqlgen.yml`

Your `gqlgen.yml` contains standard gqlgen configuration (NO plugins section):

```yaml
# Where to find the schema files that will be generated
schema:
  - graph/schema/*.graphql

# Generated code output
exec:
  filename: graph/generated.go
  package: graph

model:
  filename: graph/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"

# DO NOT add a plugins: section here - it won't work!
```

### 4. Run Code Generation

```bash
go generate ./...
```

Or directly:

```bash
go run generate.go
```

This will:
1. Run your custom `generate.go` entrypoint
2. gqlschemagen plugin generates GraphQL schemas from your Go structs
3. gqlgen continues with normal code generation

## Configuration

You can configure the plugin in two ways:

### Option 1: YAML Configuration (Recommended)

Create a `gqlschemagen.yml` file in your project root (same directory as `gqlgen.yml`):

```yaml
# Packages to scan for Go structs
packages:
  - ./graph/models
  - ./internal/domain

# Generator configuration (all optional)
generator:
  strategy: single                           # "single" or "multiple"
  output: graph/schema/generated.graphql     # Output file or directory
  field_case: camel                          # camel, snake, pascal, original, none
  use_json_tag: true                         # Use json tags for field names
  use_gqlgen_directives: false               # Generate @goModel, @goField directives
  strip_suffix: DTO,Entity,Model             # Strip suffixes from type names
  add_input_suffix: Input                    # Add suffix to input type names
  gen_inputs: true                           # Generate input types automatically
```

Then load it in your `generate.go`:

```go
func main() {
    cfg, _ := config.LoadConfigFromDefaultLocations()
    
    // Load plugin configuration from gqlschemagen.yml
    p, err := plugin.LoadConfigFromFile("gqlschemagen.yml")
    if err != nil {
        panic(err)
    }
    
    api.Generate(cfg, api.AddPlugin(p))
}
```

See the [example gqlschemagen.yml](../examples/gqlgen-plugin/gqlschemagen.yml) for all available options.

### Option 2: Programmatic Configuration

Configure directly in `generate.go`:

```go
p := plugin.New()

// Required: Packages to scan for Go structs with gql annotations
p.Packages = []string{
	"./graph/models",
	"./internal/domain",
}
```

### Advanced Configuration

You can customize the scanner behavior through the plugin's internal config:

```go
package main

import (
	"fmt"
	"os"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pablor21/gqlschemagen/generator"
	"github.com/pablor21/gqlschemagen/plugin"
)

func main() {
	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}

	// Create plugin with custom configuration
	genCfg := generator.NewConfig()
	
	// Generation strategy: single file or multiple files
	genCfg.GenStrategy = generator.GenStrategySingle // or GenStrategyMultiple
	
	// Output directory for generated schemas
	genCfg.Output = "graph/schema/generated"
	
	// Field name transformation
	genCfg.FieldCase = "camel" // camel, snake, pascal, original
	
	// Use json tags for field names
	genCfg.UseJSONTag = true
	
	// Generate @goModel and @goField directives
	genCfg.UseGQLGenDirectives = true
	
	// Strip prefixes/suffixes from type names
	genCfg.StripPrefix = []string{"DB", "Pg"}
	genCfg.StripSuffix = []string{"DTO", "Entity", "Model"}
	
	// Create plugin
	p := plugin.New()
	p.Packages = []string{"./graph/models"}
	
	// Note: Currently the plugin uses its default config
	// To use custom config, you may need to modify the plugin
	
	err = api.Generate(cfg, api.AddPlugin(p))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Packages` | `[]string` | `[]` | **Required.** List of package paths to scan for Go structs |

The plugin internally creates a `generator.Config` with these defaults:
- `GenStrategy`: `single` (all types in one file)
- `Output`: `graph/schema/generated`
- `FieldCase`: `camel`
- `UseJSONTag`: `true`
- `UseGQLGenDirectives`: `true`

## Examples

### Example 1: Single Package

```go
// generate.go
p := plugin.New()
p.Packages = []string{"./graph/models"}
```

### Example 2: Multiple Packages

```go
// generate.go
p := plugin.New()
p.Packages = []string{
	"./graph/models",
	"./internal/domain/entities",
	"./internal/domain/aggregates",
}
```

### Example 3: Monorepo

```go
// generate.go
p := plugin.New()
p.Packages = []string{
	"./services/user/models",
	"./services/product/models",
	"./pkg/common/types",
}
```

## Workflow

Here's what happens when you run `go generate ./...`:

1. **generate.go executes**
   - Loads `gqlgen.yml` configuration
   - Creates gqlschemagen plugin instance
   - Configures packages to scan

2. **gqlschemagen plugin runs** (MutateConfig hook)
   - Scans specified packages for Go structs
   - Finds structs with gql annotations (`gql.type`, `gql.input`, etc.)
   - Generates `.graphql` schema files in the output directory

3. **gqlgen continues**
   - Reads the generated schema files
   - Generates resolvers, models, and other code
   - Creates the final GraphQL server code

## Annotating Your Structs

In your Go structs, use special comments to define GraphQL types:

```go
package models

// gql.type
// User represents a user in the system
type User struct {
	ID        string    `json:"id"`        // Becomes: id: ID!
	Email     string    `json:"email"`     // Becomes: email: String!
	Name      string    `json:"name"`      // Becomes: name: String!
	CreatedAt time.Time `json:"createdAt"` // Becomes: createdAt: Time!
}

// gql.input
// CreateUserInput is the input for creating a user
type CreateUserInput struct {
	Email string `json:"email"` // email: String!
	Name  string `json:"name"`  // name: String!
}
```

For more details on annotations, see the [main README](../README.md).

## Troubleshooting

### Error: "field plugins not found in type config.Config"

This means you tried to add a `plugins:` section to `gqlgen.yml`. **This doesn't work with gqlgen.**

**Solution**: Remove the `plugins:` section from `gqlgen.yml` and configure the plugin in `generate.go` instead.

### Plugin doesn't run

Make sure:
1. You created `generate.go` with the correct code
2. You added `//go:generate go run generate.go` to a `.go` file
3. You're running `go generate ./...` or `go run generate.go`

### No schemas generated

Check:
1. `p.Packages` contains the correct paths to your struct files
2. Your structs have the proper gql annotations (`// gql.type`, etc.)
3. The output directory exists or can be created

### Import errors in generate.go

The `//go:build ignore` comment prevents the file from being part of your normal build. The imports are resolved when you run `go run generate.go`.

Make sure you have installed the dependencies:
```bash
go get github.com/99designs/gqlgen
go get github.com/pablor21/gqlschemagen/plugin
go mod tidy
```

## Complete Example

See the [examples/gqlgen-plugin](../examples/gqlgen-plugin) directory for a complete working example.

## Why This Approach?

gqlgen's architecture requires plugins to be registered programmatically because:

1. Plugins can have complex configurations that don't fit in YAML
2. Plugins may need to modify the gqlgen config before code generation
3. This gives plugins full control over the generation process

While this requires an extra file (`generate.go`), it provides maximum flexibility and follows gqlgen's official plugin documentation.

## Related

- [Main CLI Tool](../README.md) - Use gqlschemagen as a standalone CLI tool
- [gqlgen Plugin Documentation](https://gqlgen.com/reference/plugins/) - Official gqlgen plugin guide
- [Examples](../examples/) - Complete working examples
