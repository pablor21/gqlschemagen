# Examples

This directory contains examples of how to use `gqlschemagen` both as a CLI tool and as a gqlgen plugin.

## Directory Structure

```
examples/
├── cli/                    # CLI usage example
│   ├── models/            # Go structs with gql annotations
│   └── gqlschemagen.yml     # Configuration file
└── gqlgen-plugin/         # gqlgen plugin usage example
    ├── graph/
    │   └── models/        # Go structs with gql annotations
    ├── generate.go        # Custom gqlgen entrypoint with plugin
    ├── gqlgen.yml         # Standard gqlgen config (no plugins section)
    └── README.md          # Plugin setup instructions
```

## Quick Start

### CLI Example

```bash
cd examples/cli
gqlschemagen
```

This will generate GraphQL schemas from the models in `models/` directory using the configuration in `gqlschemagen.yml`.

### Plugin Example

```bash
cd examples/gqlgen-plugin
go run github.com/99designs/gqlgen generate
```

This will use gqlgen with the gqlschemagen plugin to generate schemas from the models.

## Model Annotations

Both examples use the same annotation format:

```go
/**
 * @gqlType(name:"User",description:"A user in the system")
 * @gqlInput(name:"CreateUserInput")
 */
type User struct {
    ID       string `gql:"id,type:ID,required,description:User ID"`
    Email    string `gql:"email,required,description:User email"`
    Username string `gql:"username,required,description:Username"`
}
```

See individual example READMEs for more details.
