# Auto-Loading Configuration Example

This example demonstrates how to use `gqlschemagen` as a library with automatic configuration loading.

## How It Works

The library will automatically search for configuration files in the following order:
1. `.gqlschemagen.yml` in the current directory
2. `gqlschemagen.yml` in the current directory
3. `gqlschemagen.yaml` in the current directory

If not found in the current directory, it will search parent directories recursively up to the root.

## Usage

```go
package main

import (
    "log"
    "github.com/pablor21/gqlschemagen/generator"
)

func main() {
    // Auto-load config and generate
    if err := generator.GenerateFromDefaultConfig(); err != nil {
        log.Fatalf("Generation failed: %v", err)
    }
}
```

## API Reference

### `GenerateFromDefaultConfig() error`
Automatically searches for a config file in the current directory and parent directories, then generates the schema. If no config file is found, it uses default configuration values.

### `LoadConfig() (*Config, error)`
Searches for and loads a config file. Returns a config with defaults if no file is found.

### `FindConfig() string`
Returns the path to the config file if found, or empty string if not found.

### `LoadConfigFromFile(path string) (*Config, error)`
Loads a specific config file by path.

### `Generate(cfg *Config) error`
Generates the schema using the provided configuration.

## Programmatic Usage

You can also load the config and modify it before generating:

```go
package main

import (
    "log"
    "github.com/pablor21/gqlschemagen/generator"
)

func main() {
    // Load config (auto-searches for config file)
    cfg, err := generator.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Customize config programmatically
    cfg.UseGqlGenDirectives = true
    cfg.FieldCase = generator.FieldCaseCamel
    cfg.Packages = append(cfg.Packages, "./additional/models")

    // Generate with modified config
    if err := generator.Generate(cfg); err != nil {
        log.Fatalf("Generation failed: %v", err)
    }
}
```

## Running the Example

```bash
# Create a config file first
cd examples/autoload
cp ../../gqlschemagen.yml .

# Run the example
go run main.go
```
