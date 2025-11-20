# Path Resolution with `//go:generate`

When using `gqlschemagen` with `//go:generate`, paths in your config file are automatically resolved relative to the **config file's location**, not the current working directory.

## Example

Given this directory structure:

```
myproject/
├── gqlschemagen.yml
├── models/
│   └── user.go
└── cmd/
    └── generate.go
```

**gqlschemagen.yml:**
```yaml
packages:
  - ./models  # Relative to config file location (myproject/)
output: ./schema
```

**cmd/generate.go:**
```go
package main

//go:generate go run github.com/pablor21/gqlschemagen

import "github.com/pablor21/gqlschemagen/generator"

func main() {
    // This will automatically find ../gqlschemagen.yml
    // and resolve ./models as myproject/models
    generator.GenerateFromDefaultConfig()
}
```

## How It Works

1. The library searches for config files (`.gqlschemagen.yml`, `gqlschemagen.yml`, `gqlschemagen.yaml`) starting from the current directory and walking up parent directories.

2. When a config file is found, the library stores its directory in `cfg.ConfigDir`.

3. All relative paths in the config (`packages`, `output`) are automatically resolved relative to `ConfigDir`, **not** the current working directory.

4. Absolute paths remain unchanged.

## Example: Generate from Different Directory

```bash
# Config is in /project/gqlschemagen.yml
# You run from /project/cmd/
cd /project/cmd
go run github.com/pablor21/gqlschemagen
```

The config's `./models` path will correctly resolve to `/project/models`, not `/project/cmd/models`.

## Programmatic Usage

When loading config programmatically, you can check where it was loaded from:

```go
cfg, err := generator.LoadConfig()
if err != nil {
    log.Fatal(err)
}

fmt.Println("Config loaded from:", cfg.ConfigDir)
fmt.Println("Resolved packages:", cfg.Packages)
```

## Absolute Paths

If you use absolute paths in your config, they will not be modified:

```yaml
packages:
  - /absolute/path/to/models  # Not changed
  - ./relative/models         # Resolved relative to config location
```
