# CLI Example

This example demonstrates how to use `gqlschemagen` as a standalone CLI tool.

## Setup

1. Install gqlschemagen:
```bash
go install github.com/pablor21/gqlschemagen@latest
```

## Usage

### Using Configuration File

The `gqlschemagen.yml` file contains all the configuration:

```bash
gqlschemagen
```

This will:
- Read models from `./models`
- Generate GraphQL schema to `./schema`
- Use single file strategy (all types in one `schema.graphql`)
- Use camel case for naming
- Strip suffixes like "DTO", "Entity", "Model"

### Using CLI Flags

Override configuration with flags:

```bash
gqlschemagen -i ./models -o ./schema -s single -c camel
```

### Available Flags

- `-i, --input`: Input directory (default: "./models")
- `-o, --output`: Output directory (default: "./schema")
- `-s, --strategy`: Generation strategy: single or separate (default: "single")
- `-c, --case`: Name case: camel, pascal, snake (default: "camel")
- `--strip-prefix`: Comma-separated prefixes to strip
- `--strip-suffix`: Comma-separated suffixes to strip
- `-f, --config`: Config file path (default: "gqlschemagen.yml")

## Models

The `models/` directory contains example Go structs with gql annotations:

- `user.go` - User, UserRole, UserProfile
- `post.go` - Post, PostStatus, Comment

### Annotation Examples

**Type Definition:**
```go
/**
 * @gqlType(name:"User",description:"Represents a user")
 */
type User struct {
    ID string `gql:"id,type:ID,required"`
}
```

**Generate Input Types:**
```go
/**
 * @gqlInput(name:"CreateUserInput")
 * @gqlInput(name:"UpdateUserInput")
 */
type User struct {
    // ...
}
```

**Enum Types:**
```go
/**
 * @gqlEnum(description:"User role")
 */
type UserRole string
```

**Field Options:**
- `type`: GraphQL type (ID, String, Int, DateTime, etc.)
- `required`: Mark field as non-nullable
- `optional`: Mark field as nullable (pointer fields are auto-optional)
- `description`: Field description
- `forceResolver`: Exclude from input types (for computed fields)
- `include`: Include field even in @gqlIgnoreAll types

## Output

Running `gqlschemagen` generates `schema/schema.graphql` with:

```graphql
type User {
  id: ID!
  email: String!
  username: String!
  firstName: String
  lastName: String
  bio: String
  avatar: String
  isActive: Boolean!
  role: UserRole!
  createdAt: DateTime!
  updatedAt: DateTime!
}

input CreateUserInput {
  email: String!
  username: String!
  firstName: String
  lastName: String
  bio: String
  avatar: String
  isActive: Boolean!
  role: UserRole!
}

# ... more types
```
