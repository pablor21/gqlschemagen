[![GoDoc](https://godoc.org/github.com/pablor21/gqlschemagen?status.svg)](https://godoc.org/github.com/pablor21/gqlschemagen)
[![GitHub release](https://img.shields.io/github/release/pablor21/gqlschemagen.svg?v0.1.0)](https://img.shields.io/github/release/pablor21/gqlschemagen.svg?v0.1.0)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/pablor21/gqlschemagen/master/LICENSE)

# GQLSchemaGen

[![Logo](art/logo.svg)](./art/logo.svg)

## GraphQL Schema Generator

Generate GraphQL schemas directly from your Go models and DTOs.

## Overview

> Are you a vscode user? Check out the [gqlschemagen-vscode extension](https://marketplace.visualstudio.com/items?itemName=pablor21.gqlschemagen-vscode) for seamless integration!

**GQLSchemaGen** simplifies the process of creating GraphQL schemas from your existing Go code. Instead of manually writing schemas, you can generate them directly from your structs, saving time and reducing errors.

### Features

- Automatically generate GraphQL types from Go structs.
- Supports input and output types.
- Works with standard Go structs.
- **Full support for Go generics** - embed generic types seamlessly.
- Simple CLI for quick schema generation.

> For advanced features like integration with GQLGen, annotations, struct tags, and configuration, see the [full documentation](https://pablor21.github.io/gqlschemagen).

## Installation

```bash
go install github.com/pablor21/gqlschemagen/cmd/gqlschemagen@latest
```

## Usage

Generate a schema from your project:

```bash
# In your project directory
go run github.com/pablor21/gqlschemagen/cmd/gqlschemagen generate
```

This will scan your structs and produce the corresponding GraphQL schema files.

## Quick Start Example

Given a Go struct:

```go
type User struct {
    ID    string
    Name  string
    Email string
}
```

Running the generator will produce a GraphQL type:

```graphql
type User {
  id: String!
  name: String!
  email: String!
}
```

You can now use this schema directly in your GraphQL server.

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
