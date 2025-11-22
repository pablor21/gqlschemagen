[![GoDoc](https://godoc.org/github.com/pablor21/gqlschemagen?status.svg)](https://godoc.org/github.com/pablor21/gqlschemagen)
[![GitHub release](https://img.shields.io/github/release/pablor21/gqlschemagen.svg)](https://github.com/pablor21/gqlschemagen/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/pablor21/gqlschemagen/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/pablor21/gqlschemagen)](https://goreportcard.com/report/github.com/pablor21/gqlschemagen)

# GQLSchemaGen

[![Logo](art/logo.svg)](./art/logo.svg)

**Generate GraphQL schemas from Go structs with annotations, generics support, and advanced field control.**

> 📖 **Full Documentation:** [https://pablor21.github.io/gqlschemagen](https://pablor21.github.io/gqlschemagen)

> 💡 **VS Code Extension:** [gqlschemagen-vscode](https://marketplace.visualstudio.com/items?itemName=pablor21.gqlschemagen-vscode)

## What is GQLSchemaGen?

GQLSchemaGen generates GraphQL schema files (`.graphqls`) from your Go code using simple annotations. Designed to work seamlessly with [gqlgen](https://gqlgen.com), it turns your Go structs into GraphQL types, inputs, and enums—keeping your schema in sync with your codebase.

### Key Features

✨ **Code-First Schema Generation** - Annotate Go structs to generate types, inputs, and enums  
🎯 **Advanced Field Control** - Fine-grained visibility with `ro`/`wo`/`rw` tags and field filtering  
🔄 **Auto-Discovery** - Automatically generate schemas for referenced types  
🧬 **Full Generics Support** - Works with Go 1.18+ generic types (`Response[T]`, `Connection[T]`)  
🗂️ **Namespace Organization** - Organize schemas into folders/namespaces  
🔧 **Scalar Mappings** - Map Go types to GraphQL scalars globally (UUID → ID, time.Time → DateTime)  
🛡️ **Schema Preservation** - Keep manual edits with `@GqlKeepBegin`/`@GqlKeepEnd` markers  
⚙️ **gqlgen Integration** - Generates `@goModel`, `@goField` directives automatically

## Quick Start

### Installation

```bash
go install github.com/pablor21/gqlschemagen/cmd/gqlschemagen@latest
```

### Basic Usage

**1. Annotate your Go structs:**

```go
package models

// @gqlType
type User struct {
    ID        string    `gql:"id,type:ID"`
    Name      string    `gql:"name"`
    Email     string    `gql:"email"`
    CreatedAt time.Time `gql:"createdAt,ro"` // Read-only (excluded from inputs)
}

// @gqlInput
type CreateUserInput struct {
    Name     string `gql:"name"`
    Email    string `gql:"email"`
    Password string `gql:"password,wo"` // Write-only (excluded from types)
}

// @gqlEnum
type UserRole string

const (
    UserRoleAdmin  UserRole = "admin"  // @gqlEnumValue(name:"ADMIN")
    UserRoleViewer UserRole = "viewer" // @gqlEnumValue(name:"VIEWER")
)
```

**2. Create config file `gqlschemagen.yml`:**

```yaml
packages:
  - ./models

output: schema.graphqls
```

**3. Generate schema:**

```bash
gqlschemagen generate
```

**4. Generated `schema.graphqls`:**

```graphql
type User @goModel(model: "your-module/models.User") {
  id: ID!
  name: String!
  email: String!
  createdAt: String!
}

input CreateUserInput {
  name: String!
  email: String!
  password: String!
}

enum UserRole {
  ADMIN
  VIEWER
}
```

## Advanced Features

### Multiple Types from One Struct

```go
// @gqlType(name:"User")
// @gqlType(name:"PublicUser", ignoreAll:"true")
// @gqlInput(name:"CreateUserInput")
// @gqlInput(name:"UpdateUserInput")
type User struct {
    ID    string `gql:"id,type:ID,ro"`
    Name  string `gql:"name,include:*"`      // Include in all
    Email string `gql:"email,include:User"`  // Only in User type
}
```

### Scalar Mappings

```yaml
scalars:
  ID:
    model:
      - github.com/google/uuid.UUID
  DateTime:
    model:
      - time.Time
```

### Auto-Generation

```yaml
auto_generate:
  strategy: referenced  # none | referenced | all | patterns
  max_depth: 3
```

## Documentation

- [Getting Started](https://pablor21.github.io/gqlschemagen/docs/getting-started)
- [Configuration](https://pablor21.github.io/gqlschemagen/docs/configuration)
- [Features](https://pablor21.github.io/gqlschemagen/docs/features/gql-types)
  - [Types & Inputs](https://pablor21.github.io/gqlschemagen/docs/features/gql-types)
  - [Field Filtering](https://pablor21.github.io/gqlschemagen/docs/features/field-filtering)
  - [Scalar Mappings](https://pablor21.github.io/gqlschemagen/docs/features/scalar-mappings)
  - [Auto-Generation](https://pablor21.github.io/gqlschemagen/docs/features/auto-generation)
  - [Generics Support](https://pablor21.github.io/gqlschemagen/docs/features/generics)
- [CLI Reference](https://pablor21.github.io/gqlschemagen/docs/cli-reference)

## Contributing

We welcome contributions! To add new features or improvements, please follow these steps:

1. **Update configuration**
   Modify the `Config` struct in `config.go` to include any new settings required by your feature.

2. **Add parsing logic**
   Implement the necessary parsing in `directives.go` to handle any new directives or annotations.

3. **Implement code generation**
   Extend `generator.go` with the generation logic for your feature.

4. **Update documentation**
   Make sure to update this README or relevant docs to reflect your changes, including a brief explanation of the feature.

5. **Add examples**
   Provide usage examples or sample code demonstrating how to use your feature.

Following these steps ensures that your contribution is clear, well-documented, and easy for others to understand.

## License

MIT License. See [LICENSE](LICENSE.md) for details.

---

Made with ❤️ by Pablo Ramirez <pablo@pramirez.dev> | [Website](https://pramirez.dev) | [GitHub](https://github.com/pablor21)
