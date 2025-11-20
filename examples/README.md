# Examples

This directory contains examples of how to use `gqlschemagen` as a CLI tool.

## Directory Structure

```
examples/
├── cli/                           # CLI usage example
│   ├── models/                   # Go structs with gql annotations
│   └── gqlschemagen.yml          # Configuration file
├── embedded-structs.go           # Example of embedded struct handling
├── extra-fields-demo.go          # Example of extra fields directives
├── generics-relay-connections.go # Example of Go generics support (Relay connections)
├── multiple-annotations.go       # Example of multiple annotations
└── field-filtering.go            # Example of field filtering
```

## Quick Start

### CLI Example

```bash
cd examples/cli
gqlschemagen
```

This will generate GraphQL schemas from the models in `models/` directory using the configuration in `gqlschemagen.yml`.

## Model Annotations

Examples use the following annotation format:

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

See individual example files for more details.
