# GraphQL Schema Generator

A powerful, flexible GraphQL schema generator for Go projects that analyzes Go 
structs and generates GraphQL schema files with support for [gqlgen](https://github.com/99designs/gqlgen) directives, 
custom naming strategies, and extensive configuration options.


FOR THE GQLGEN PLUGIN PLEASE SEE THE [PLUGIN DOCUMENTATION](plugin/README.md)


## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [Install as a Library](#install-as-a-library)
  - [Install as a CLI Tool](#install-as-a-cli-tool)
  - [Install as a gqlgen Plugin](#install-as-a-gqlgen-plugin)
- [Quick Start](#quick-start)
  - [Standalone CLI](#standalone-cli)
  - [As a gqlgen Plugin](#as-a-gqlgen-plugin)
- [Annotations](#annotations)
  - [Type-level Annotations](#type-level-annotations)
  - [Field-level Struct Tags](#field-level-struct-tags)
- [Configuration](#configuration)
  - [Configuration File](#configuration-file)
  - [CLI Flags](#cli-flags)
  - [Field Case Transformations](#field-case-transformations)
  - [Using JSON Tags](#using-json-tags)
- [Examples](#examples)
- [Integration with gqlgen](#integration-with-gqlgen)
- [Advanced Usage](#advanced-usage)
  - [Conditional Generation](#conditional-generation)
  - [Type Composition](#type-composition)
  - [Stripping Prefixes and Suffixes](#stripping-prefixes-and-suffixes)
  - [Adding Prefixes and Suffixes](#adding-prefixes-and-suffixes)
  - [Combining Strip and Add Operations](#combining-strip-and-add-operations)
  - [Custom Scalar Types](#custom-scalar-types)
- [Troubleshooting](#troubleshooting) 
- [Contributing](#contributing)
- [License](#license)

## Features

- 🎯 **Annotation-based**: Use comments to control schema generation
- 📝 **Struct Tag Support**: Fine-grained control via struct tags
- 🔄 **Flexible Field Naming**: Multiple case transformations (camel, snake, pascal)
- 📦 **Generation Strategies**: Single file or multiple files
- 🎨 **[gqlgen](https://github.com/99designs/gqlgen) Integration**: Automatic @goModel and @goField directives
- 📋 **Input Type Generation**: Auto-generate GraphQL Input types
- 📚 **Field Descriptions**: Extract from struct tags or comments
- ⚙️ **Highly Configurable**: CLI flags and per-struct customization
- 🔌 **gqlgen Plugin**: Can be used as a standalone tool or [gqlgen](https://github.com/99designs/gqlgen) plugin
- 🧩 **Embedded Struct Support**: Automatically expand embedded struct fields into parent types
- ➕ **Extra Fields**: Add resolver-only fields to types and inputs with @gqlTypeExtraField and @gqlInputExtraField
- 🌐 **Glob Pattern Support**: Scan packages using glob patterns like `**/models` or `internal/**/entities`

## Installation

### Install as a Library

```bash
go get github.com/pablor21/gqlschemagen
```

### Install as a CLI Tool

To install the CLI tool globally:

```bash
go install github.com/pablor21/gqlschemagen@latest
```

Alternatively, you can install it as a Go tool in your project (requires Go 1.21+):

```bash
# Install as a project tool
go get -u github.com/pablor21/gqlschemagen
go get -tool github.com/pablor21/gqlschemagen

# Or add to tools.go and install
# tools.go:
# //go:build tools
# package tools
# import _ "github.com/pablor21/gqlschemagen"

# Then run:
go mod tidy
go install github.com/pablor21/gqlschemagen
```

After installation, you can run it directly:

```bash
# Using the installed binary
gqlschemagen --pkg ./internal/domain/entities --out ./graph/schema

# Or with short flags
gqlschemagen -p ./internal/domain/entities -o ./graph/schema
```

### Install as a gqlgen Plugin

The plugin is in a **separate module** that includes gqlgen as a dependency.

Add to your project's dependencies:

```bash
go get github.com/pablor21/gqlschemagen/plugin
```

**Important**: gqlgen plugins must be registered in a custom `generate.go` file, not in `gqlgen.yml`. See the [Plugin Documentation](plugin/README.md) for complete setup instructions.

## Quick Start

### Standalone CLI

The CLI supports multiple commands:

#### Initialize Configuration

Create a default configuration file:

```bash
# Create gqlschemagen.yml in current directory
gqlschemagen init

# Create with custom name
gqlschemagen init --output custom-config.yml
gqlschemagen init -o custom-config.yml

# Overwrite existing file
gqlschemagen init --force
gqlschemagen init -f
```

#### Generate Schema

You can run the generator in three ways:

1. **Using `go run`** (no installation required):
```bash
go run github.com/pablor21/gqlschemagen generate -p ./internal/domain/entities -o ./graph/schema
```

2. **Using the installed binary** (after `go install`):
```bash
gqlschemagen generate -p ./internal/domain/entities -o ./graph/schema
```

3. **Using a config file**:
```bash
# Create config first
gqlschemagen init

# Then generate (uses gqlschemagen.yml by default)
gqlschemagen generate
```

#### Get Help

View available commands and usage:

```bash
# Show all commands
gqlschemagen help
gqlschemagen --help
gqlschemagen -h

# Show help for specific command
gqlschemagen generate --help
gqlschemagen init --help
```

#### Using Command Line Flags

```bash
# Generate single schema file
gqlschemagen generate \
  --pkg ./internal/domain/entities \
  --out ./graph/schema/generated/schema.graphql

# Or using short flags
gqlschemagen generate \
  -p ./internal/domain/entities \
  -o ./graph/schema/generated/schema.graphql

# Generate multiple schema files (one per type)
gqlschemagen generate \
  --pkg ./internal/domain/entities \
  --out ./graph/schema/generated \
  --strategy multiple \
  --schema-file-name "{model_name}.graphqls"
  
# Or using short flags
gqlschemagen generate \
  -p ./internal/domain/entities \
  -o ./graph/schema/generated \
  -s multiple
```

#### Using YAML Configuration

Create a default configuration file:

```bash
# Generate default gqlschemagen.yml
gqlschemagen init
```

This creates a `gqlschemagen.yml` file with all available options:

```yaml
# gqlschemagen Plugin Configuration
# Packages to scan for Go structs with gql annotations
packages:
   - ./

# Generator configuration (optional - defaults shown)
generator:
  # Output strategy: "single" for one file, "multiple" for separate files per type
  strategy: single
  
  # Output path (file for single strategy, directory for multiple)
  output: graph/schema/generated.graphql
  
  # Field name transformation: camel, snake, pascal, original, none
  field_case: camel
  
  # Use json struct tags for field names when gql tag is not present
  use_json_tag: true
  
  # Generate @goModel and @goField directives for gqlgen
  use_gqlgen_directives: false
  
  # Base path for @goModel directive (e.g., 'github.com/user/project/models')
  model_path: ""
  
  # Strip prefixes/suffixes from type names (comma-separated)
  strip_prefix: ""
  strip_suffix: ""
  
  # Add prefixes/suffixes to type/input names
  add_type_prefix: ""
  add_type_suffix: ""
  add_input_prefix: ""
  add_input_suffix: ""
  
  # Schema file name pattern for multiple mode (default: {model_name}.graphqls)
  schema_file_name: "{model_name}.graphqls"
  
  # Include types with no fields
  include_empty_types: false
  
  # Skip generating files that already exist
  skip_existing: false
```

Then run:

```bash
# Uses gqlschemagen.yml by default
gqlschemagen generate

# Or specify a custom config file
gqlschemagen generate --config my-config.yml
gqlschemagen generate -f my-config.yml

# Can also use go run
go run github.com/pablor21/gqlschemagen generate
go run github.com/pablor21/gqlschemagen generate --config my-config.yml
```

**Note:** CLI flags override values from the config file when explicitly set.

### As a [gqlgen](https://github.com/99designs/gqlgen) Plugin

[See Plugin Documentation](./plugin/README.md) for detailed plugin usage, configuration, and integration with [gqlgen](https://github.com/99designs/gqlgen).

## Annotations

### Important: Opt-in Generation

**Types and inputs are only generated for structs that have the appropriate directives:**
- Use `@gqlType()` to generate a GraphQL type
- Use `@gqlInput()` to generate a GraphQL input
- Structs without these directives are **skipped**

This opt-in approach gives you precise control over what gets generated.

### Type-level Annotations

Add these as block comments (`/** */`) above your struct declaration:

#### `@gqlType(name:"TypeName",description:"desc",ignoreAll:true)`
Specify the GraphQL type name and optional description.

```go
/**
 * @gqlType(name:"UserProfile",description:"Represents a user in the system")
 */
type User struct {
    ID   string
    Name string
}
```

Generates:
```graphql
"""Represents a user in the system"""
type UserProfile @goModel(model: "your-package.User") {
    id: ID!
    name: String!
}
```

**Parameters:**
- `name` (optional): Custom GraphQL type name. If omitted, uses the struct name (with transformations applied)
- `description` (optional): Type description shown in GraphQL schema
- `ignoreAll` (optional): When `true`, excludes all fields unless explicitly included with `gql:"include"` tag

**Notes:**
- If you don't specify a name, the generator will apply prefix/suffix stripping and adding based on CLI flags
- Custom names bypass all transformations
- `ignoreAll:true` works like `@gqlIgnoreAll` but only for the type, not the input

#### `@gqlInput(name:"InputName",description:"desc",ignoreAll:true)`
Generate an Input type with optional custom name and description.

```go
/**
 * @gqlType()
 * @gqlInput(name:"CreateUserInput",description:"Input for creating a user")
 */
type User struct {
    Name  string
    Email string
}
```

Generates both:
```graphql
type User @goModel(model: "your-package.User") {
    name: String!
    email: String!
}

"""Input for creating a user"""
input CreateUserInput @goModel(model: "your-package.User") {
    name: String!
    email: String!
}
```

**Parameters:**
- `name` (optional): Custom input name. If omitted, generates `{TypeName}Input`
- `description` (optional): Input description

#### `@gqlIgnoreAll`
Ignore all fields by default (use with `include` tag to selectively include).

```go
/**
 * @gqlIgnoreAll
 */
type InternalUser struct {
    ID       string `gql:"include"`  // Only this field will be exported
    Internal string                 // Ignored
    Private  string                 // Ignored
}
```

#### `@gqlUseModelDirective`
Force @goModel directive for this type (even if UseGqlGenDirectives is false).

```go
/**
 * @gqlUseModelDirective
 */
type User struct {
    ID string
}
```

Generates:
```graphql
type User @goModel(model: "your-package.User") {
    id: ID!
}
```

#### `@gqlTypeExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Type1,Type2")`
Add extra fields only to GraphQL types (not inputs). Useful for resolver-only fields. Can be used multiple times.

```go
/**
 * @gqlType(name:"User")
 * @gqlInput(name:"UserInput")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",description:"User's posts")
 * @gqlTypeExtraField(name:"followers",type:"[User!]!",description:"Followers list")
 */
type User struct {
    ID       string
    Username string
    Email    string
}
```

Generates:
```graphql
type User @goModel(model: "your-package.User") {
    id: ID!
    username: String!
    email: String!
    """User's posts"""
    posts: [Post!]! @goField(forceResolver: true)
    """Followers list"""
    followers: [User!]! @goField(forceResolver: true)
}

input UserInput @goModel(model: "your-package.User") {
    id: ID!
    username: String!
    email: String!
    # Note: extra fields NOT included in input
}
```

**Parameters:**
- `name` (required): Field name in the GraphQL schema
- `type` (required): GraphQL type (e.g., `String!`, `[Post!]!`, `User`)
- `description` (optional): Field description
- `on` (optional): Comma-separated list of type names to apply this field to. Defaults to `*` (all types)

**Using the `on` parameter:**

```go
/**
 * @gqlType(name:"Article")
 * @gqlType(name:"BlogPost")
 * @gqlTypeExtraField(name:"author",type:"User!",description:"Article author",on:"Article")
 * @gqlTypeExtraField(name:"writer",type:"User!",description:"Blog writer",on:"BlogPost")
 * @gqlTypeExtraField(name:"comments",type:"[Comment!]!",description:"Comments")
 */
type Content struct {
    ID    string
    Title string
}
```

Generates:
```graphql
# Article type gets "author" and "comments" fields
type Article {
    id: ID!
    title: String!
    """Article author"""
    author: User! @goField(forceResolver: true)
    """Comments"""
    comments: [Comment!]! @goField(forceResolver: true)
}

# BlogPost type gets "writer" and "comments" fields
type BlogPost {
    id: ID!
    title: String!
    """Blog writer"""
    writer: User! @goField(forceResolver: true)
    """Comments"""
    comments: [Comment!]! @goField(forceResolver: true)
}
```

#### `@gqlInputExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Input1,Input2")`
Add extra fields only to GraphQL inputs (not types). Useful for input-specific fields like passwords.

```go
/**
 * @gqlType(name:"User")
 * @gqlInput(name:"CreateUserInput")
 * @gqlInput(name:"UpdateUserInput")
 * @gqlInputExtraField(name:"password",type:"String!",description:"User password",on:"CreateUserInput")
 */
type User struct {
    ID       string
    Username string
    Email    string
}
```

Generates:
```graphql
type User {
    id: ID!
    username: String!
    email: String!
    # Note: password NOT in type
}

input CreateUserInput {
    id: ID!
    username: String!
    email: String!
    """User password"""
    password: String!
}

input UpdateUserInput {
    id: ID!
    username: String!
    email: String!
    # Note: password NOT in UpdateUserInput (on:"CreateUserInput" only)
}
```

**Parameters:**
- `name` (required): Field name in the GraphQL schema
- `type` (required): GraphQL type (e.g., `String!`, `ID`, `[String!]`)
- `description` (optional): Field description
- `on` (optional): Comma-separated list of input names to apply this field to. Defaults to `*` (all inputs)

**Notes:**
- Extra fields automatically get `@goField(forceResolver: true)` for types when gqlgen directives are enabled
- The `on` parameter accepts `*` (all), specific type/input names, or comma-separated lists
- These fields must be implemented as resolvers (for types) or handled in your input processing (for inputs)

### Field-level Struct Tags

Control individual field behavior with `gql:` struct tag. The first part is always the field name (can be omitted with leading comma to use json tag or transformed struct field name).

#### Basic Options

| Tag | Description | Example |
|-----|-------------|---------|
| First value | Custom field name | `gql:"userId"` or `gql:"userId,type:ID"` |
| Omit name | Use json tag or transformed name | `gql:",type:ID"` |
| `type:value` | Custom GraphQL type | `gql:"createdAt,type:DateTime"` |
| `description:value` | Field description | `gql:"email,description:User's email"` |
| `ignore` | Skip this field | `gql:"ignore"` |
| `omit` | Alias for ignore | `gql:"omit"` |
| `include` | Include even if @gqlIgnoreAll | `gql:"include"` |
| `optional` | Make field nullable (remove !) | `gql:"age,optional"` |
| `required` | Force non-null (add !) | `gql:"email,required"` |
| `forceResolver` | Add @goField(forceResolver: true) | `gql:"author,forceResolver"` |

#### Examples

Checkout the [Examples](./examples) section for more detailed usage.

```go
type User struct {
    // Custom name and type
    ID string `gql:"userId,type:ID"`
    
    // Use json tag name (with -use-json-tag flag or leading comma)
    Email string `gql:",type:Email,required" json:"email_address"`
    
    // Multiple options
    CreatedAt time.Time `gql:"createdAt,type:DateTime,forceResolver,description:Account creation time"`
    
    // Optional field
    Age *int `gql:"age,optional"`
    
    // Ignored field
    Internal string `gql:"ignore"`
    
    // Include in @gqlIgnoreAll type
    PublicID string `gql:"include"`
}
```

#### Combining Multiple Options

```go
type User struct {
    ID        string    `gql:"userId,type:ID,required,description:Unique user identifier"`
    Email     string    `gql:"email,type:Email,required,description:User's email address"`
    Age       *int      `gql:"age,optional,description:User's age"`
    Internal  string    `gql:"ignore"`
    CreatedAt time.Time `gql:"createdAt,type:DateTime,forceResolver,description:Creation timestamp"`
}
```

Generates:
```graphql
type User @goModel(model: "your-package.User") {
    """Unique user identifier"""
    userId: ID!
    
    """User's email address"""
    email: Email!
    
    """User's age"""
    age: Int
    
    """Creation timestamp"""
    createdAt: DateTime @goField(forceResolver: true)
}
```

## Configuration

### Configuration File

You can use a YAML configuration file instead of CLI flags.

#### Create Configuration File

```bash
# Create default gqlschemagen.yml
gqlschemagen init

# Create with custom name
gqlschemagen init -o custom.yml
```

#### Use Configuration File

```bash
# Uses gqlschemagen.yml by default
gqlschemagen generate

# Specify a custom config file
gqlschemagen generate --config custom.yml
gqlschemagen generate -f custom.yml

# Also works with go run
go run github.com/pablor21/gqlschemagen generate
go run github.com/pablor21/gqlschemagen generate --config custom.yml
```

**Example gqlschemagen.yml:**

```yaml
# List of packages to scan (supports glob patterns)
packages: 
    - ./internal/domain/entities
    - ./internal/models
    # Glob patterns are supported:
    - ./internal/**/models     # All 'models' directories under 'internal'
    - ./**/entities/*.go       # All .go files in 'entities' directories
output: ./graph/schema/generated
strategy: single
field_case: camel
use_json_tag: true
use_gqlgen_directives: false
model_path: github.com/user/project/models
strip_prefix: DB,Pg
strip_suffix: DTO,Entity,Model
add_type_prefix: ""
add_type_suffix: ""
add_input_prefix: ""
add_input_suffix: ""
schema_file_name: "{model_name}.graphqls"
include_empty_types: false
skip_existing: false
```

**Package Path Patterns:**
- **Direct paths**: `./internal/models` - scans all .go files in the directory
- **Simple globs**: `./internal/*/models` - matches single-level wildcards
- **Recursive globs**: `./internal/**/models` - matches any depth (recommended)
- **File patterns**: `./**/entities/*.go` - matches specific files recursively

**Notes:**
- If the default `gqlschemagen.yml` doesn't exist, it's silently ignored
- If a custom config file is specified but not found, an error is raised
- CLI flags override values from the config file when explicitly set
- Glob patterns are processed recursively for `**` patterns

### CLI Flags

#### Generate Command Flags

All flags support both long format (`--flag`) and short format (`-f`):

```bash
gqlschemagen generate [flags]
```

**Available Flags:**

```bash
--config, -f string
    Path to config file (default: "gqlschemagen.yml")

--pkg, -p string
    Root package directory to scan (required)
    
--out, -o string
    Output directory or file path (default: "graph/schema/gqlschemagen.graphqls" for single, "graph/schema" for multiple)
    
--strategy, -s string
    Generation strategy: "single" or "multiple" (default: "single")
    
--field-case, -c string
    Field name case transformation: "camel", "snake", "pascal", "original", or "none" (default: "camel")
    
--use-json-tag bool
    Use json tag for field names instead of struct field name (default: true)
    
--gqlgen, --use-gqlgen-directives bool
    Generate @goModel and @goField directives for gqlgen (default: false)

--model-path, -m string
    Base path for @goModel directive (e.g., 'github.com/user/project/models')
    If specified, overrides the actual package path in @goModel directives

--strip-prefix string
    Comma-separated list of prefixes to strip from type names (e.g., 'DB,Pg')
    Only applies when @gqlType doesn't specify a custom name
    Example: DBUser -> User, PgProduct -> Product

--strip-suffix string
    Comma-separated list of suffixes to strip from type names (e.g., 'DTO,Entity,Model')
    Only applies when @gqlType doesn't specify a custom name
    Example: UserDTO -> User, PostEntity -> Post

--add-type-prefix string
    Prefix to add to GraphQL type names (unless @gqlType specifies custom name)
    Example: 'Gql' converts User -> GqlUser

--add-type-suffix string
    Suffix to add to GraphQL type names (unless @gqlType specifies custom name)
    Example: 'Type' converts User -> UserType

--add-input-prefix string
    Prefix to add to GraphQL input names (unless @gqlInput specifies custom name)
    Example: 'Input' converts CreateUserInput -> InputCreateUserInput

--add-input-suffix string
    Suffix to add to GraphQL input names (unless @gqlInput specifies custom name)
    Example: 'Payload' converts CreateUserInput -> CreateUserInputPayload
    
--schema-file-name string
    Schema file name pattern for multiple mode (default: "{model_name}.graphqls")
    Available placeholders: {model_name}, {type_name}
    
--include-empty-types bool
    Include types with no fields in the schema (default: false)
    
--skip-existing bool
    Skip generating files that already exist (default: false)
```

### Field Case Transformations

#### Camel Case (default)
```go
type User struct {
    FirstName string  // -> firstName
    LastName  string  // -> lastName
}
```

#### Snake Case
```bash
--field-case snake
```
```go
type User struct {
    FirstName string  // -> first_name
    LastName  string  // -> last_name
}
```

#### Pascal Case
```bash
--field-case pascal
```
```go
type User struct {
    FirstName string  // -> FirstName (unchanged)
    LastName  string  // -> LastName (unchanged)
}
```

#### Original (no transformation)
```bash
--field-case original
```

### Using JSON Tags

When `--use-json-tag` is enabled (default: `true`), the generator uses json tag names for field names and respects `json:"-"` to ignore fields.

```bash
--use-json-tag  # (default: true)
```

```go
type User struct {
    ID        string `json:"user_id"`     // -> user_id
    FirstName string `json:"firstName"`   // -> firstName
    LastName  string `json:"last_name"`   // -> last_name
    Password  string `json:"-"`           // -> ignored (json:"-" means skip)
    Internal  string `json:"-"`           // -> ignored
}
```

**Generated:**
```graphql
type User {
    user_id: String!
    firstName: String!
    last_name: String!
}
```

#### Override with `gql:"include"`

You can force inclusion of `json:"-"` fields using the `include` flag:

```go
type SecureUser struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    PasswordHash string `json:"-" gql:"include"`  // Included despite json:"-"
    ApiKey       string `json:"-"`                 // Ignored
}
```

**Generated:**
```graphql
type SecureUser {
    id: String!
    name: String!
    passwordHash: String!
}
```

**Notes:**
- `json:"-"` is only respected when `-use-json-tag=true`
- `gql:"include"` overrides `json:"-"` behavior
- Priority: `gql` tag > `json` tag > struct field name

## Examples

### Example 1: Basic Type

```go
package entities

type User struct {
    ID        string
    Email     string
    FirstName string
    LastName  string
}
```

**Command:**
```bash
gqlschemagen generate --pkg ./internal/domain/entities --out ./graph/schema
```

**Generated schema.graphql:**
```graphql
type User @goModel(model: "jobix.com/backend/internal/domain/entities.User") {
    id: ID!
    email: String!
    firstName: String!
    lastName: String!
}
```

### Example 2: Custom Type Name with Input

```go
// @gqlType:UserProfile
// @gqlInput:UpdateUserInput
// @gqlType(description:User profile information
type User struct {
    ID        string    `gql:"type:ID,description:Unique identifier"`
    Email     string    `gql:"type:Email,required:,description:Email address"`
    FirstName string    `gql:"description:User's first name"`
    LastName  string    `gql:"description:User's last name"`
    Age       *int      `gql:"optional:,description:User's age"`
}
```

**Generated:**
```graphql
"""User profile information"""
type UserProfile @goModel(model: "jobix.com/backend/internal/domain/entities.User") {
    """Unique identifier"""
    id: ID!
    
    """Email address"""
    email: Email!
    
    """User's first name"""
    firstName: String!
    
    """User's last name"""
    lastName: String!
    
    """User's age"""
    age: Int
}

"""User profile information (Input)"""
input UpdateUserInput @goModel(model: "jobix.com/backend/internal/domain/entities.User") {
    """Unique identifier"""
    id: ID!
    
    """Email address"""
    email: Email!
    
    """User's first name"""
    firstName: String!
    
    """User's last name"""
    lastName: String!
    
    """User's age"""
    age: Int
}
```

### Example 3: Selective Field Export

```go
// @gqlIgnoreAll
type SecureUser struct {
    ID           string `gql:"include:,type:ID"`
    Email        string `gql:"include:"`
    PasswordHash string // Ignored
    Salt         string // Ignored
    SessionToken string // Ignored
}
```

**Generated:**
```graphql
type SecureUser @goModel(model: "jobix.com/backend/internal/domain/entities.SecureUser") {
    id: ID!
    email: String!
}
```

### Example 4: Custom Types and Resolvers

```go
type Post struct {
    ID        string    `gql:"type:ID"`
    Title     string
    Content   string
    Author    *User     `gql:"forceResolver:,description:Post author"`
    CreatedAt time.Time `gql:"type:DateTime,forceResolver:"`
    UpdatedAt time.Time `gql:"type:DateTime,forceResolver:"`
}
```

**Generated:**
```graphql
type Post @goModel(model: "jobix.com/backend/internal/domain/entities.Post") {
    id: ID!
    title: String!
    content: String!
    
    """Post author"""
    author: User @goField(forceResolver: true)
    
    createdAt: DateTime @goField(forceResolver: true)
    updatedAt: DateTime @goField(forceResolver: true)
}
```

### Example 5: Multiple File Generation

```bash
gqlschemagen generate \
  --pkg ./internal/domain/entities \
  --out ./graph/schema/types \
  --strategy multiple \
  --schema-file-name "{model_name}.graphqls"
```

**File structure:**
```
graph/schema/types/
├── user.graphqls
├── post.graphqls
├── comment.graphqls
└── ...
```

## Integration with gqlgen

The generator is designed to work seamlessly with [gqlgen](https://github.com/99designs/gqlgen).

### gqlgen.yml Configuration

```yaml
schema:
  - graph/schema/generated/*.graphql
  - graph/schema/generated/*.graphqls

models:
  ID:
    model: github.com/99designs/gqlgen/graphql.ID
  DateTime:
    model: time.Time
  Email:
    model: string

autobind:
  - yourproject.com/backend/internal/domain/entities
```

### Workflow

1. Define your domain entities with annotations:
```go
/**
 * @gqlType()
 */
type User struct {
    ID        string    `gql:"type:ID"`
    Email     string    `gql:"type:Email"`
    CreatedAt time.Time `gql:"type:DateTime,forceResolver"`
}
```

2. Generate GraphQL schema:
```bash
gqlschemagen generate --pkg ./internal/domain/entities --out ./graph/schema/generated
```

3. Generate [gqlgen](https://github.com/99designs/gqlgen) code:
```bash
go run github.com/99designs/gqlgen generate
```

4. Implement resolvers for fields marked with `forceResolver`:
```go
func (r *userResolver) CreatedAt(ctx context.Context, obj *entities.User) (time.Time, error) {
    return obj.CreatedAt, nil
}
```

## Advanced Usage

### Conditional Generation

Use struct tags to create different schemas for different use cases:

```go
type User struct {
    ID           string `gql:"type:ID" json:"id"`
    Email        string `gql:"required:" json:"email"`
    PasswordHash string `gql:"ignore:" json:"-"`
    
    // Admin-only fields
    InternalNotes string `gql:"omit:" json:"internal_notes,omitempty"`
}
```

### Type Composition

```go
type BaseEntity struct {
    ID        string    `gql:"type:ID"`
    CreatedAt time.Time `gql:"type:DateTime,forceResolver:"`
    UpdatedAt time.Time `gql:"type:DateTime,forceResolver:"`
}

type User struct {
    BaseEntity           // Embedded fields are included
    Email      string    `gql:"type:Email"`
    Name       string
}
```

### Stripping Prefixes and Suffixes

Automatically remove common prefixes and suffixes from type names:

```bash
gqlschemagen generate \
  --pkg ./internal/domain \
  --out ./graph/schema/generated/schema.graphql \
  --strip-suffix "DTO,Entity,Model" \
  --strip-prefix "DB,Pg"
```

```go
package domain

/**
 * @gqlType
 * UserDTO becomes "User" in GraphQL
 */
type UserDTO struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

/**
 * @gqlType
 * PostEntity becomes "Post" in GraphQL
 */
type PostEntity struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

/**
 * @gqlType
 * DBProduct becomes "Product" in GraphQL
 */
type DBProduct struct {
    ID    string  `json:"id"`
    Price float64 `json:"price"`
}

/**
 * @gqlType(name:"CustomName")
 * Custom name overrides stripping
 */
type CategoryDTO struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

/**
 * @gqlType
 * @gqlInput
 * OrderDTO becomes "Order" type and "OrderInput"
 */
type OrderDTO struct {
    UserID    string `json:"user_id"`
    ProductID string `json:"product_id"`
}
```

Generates:
```graphql
type User @goModel(model: "domain.UserDTO") {
    id: String!
    name: String!
}

type Post @goModel(model: "domain.PostEntity") {
    id: String!
    title: String!
}

type Product @goModel(model: "domain.DBProduct") {
    id: String!
    price: Float!
}

type CustomName @goModel(model: "domain.CategoryDTO") {
    id: String!
    name: String!
}

type Order @goModel(model: "domain.OrderDTO") {
    user_id: String!
    product_id: String!
}

input OrderInput @goModel(model: "domain.OrderDTO") {
    user_id: String!
    product_id: String!
}
```

**Notes:**
- Stripping only applies when `@gqlType` or `@gqlInput` doesn't specify a custom name
- For input types, the base name is stripped, then "Input" is appended
- Only one prefix and one suffix are stripped per type (first match wins)
- Use commas to separate multiple prefixes/suffixes

### Adding Prefixes and Suffixes

Add prefixes or suffixes to GraphQL type and input names:

```bash
gqlschemagen generate \
  --pkg ./internal/domain \
  --out ./graph/schema/generated/schema.graphql \
  --add-type-prefix "Gql" \
  --add-input-suffix "Payload"
```

```go
package domain

/**
 * @gqlType
 */
type User struct {
    ID   string
    Name string
}

/**
 * @gqlType
 * @gqlInput
 */
type CreateOrder struct {
    UserID    string
    ProductID string
}

/**
 * @gqlType(name:"CustomTypeName")
 * Custom name overrides additions
 */
type Product struct {
    ID string
}
```

Generates:
```graphql
type GqlUser @goModel(model: "domain.User") {
    id: String!
    name: String!
}

type GqlCreateOrder @goModel(model: "domain.CreateOrder") {
    userID: String!
    productID: String!
}

input CreateOrderInputPayload @goModel(model: "domain.CreateOrder") {
    userID: String!
    productID: String!
}

type CustomTypeName @goModel(model: "domain.Product") {
    id: String!
}
```

**Notes:**
- Additions only apply when `@gqlType` or `@gqlInput` doesn't specify a custom name
- For inputs, prefix/suffix are added AFTER "Input" is appended
- Can combine with stripping: strip first, then add

### Combining Strip and Add Operations

You can combine stripping and adding for complete control:

```bash
gqlschemagen generate \
  --pkg ./internal/domain \
  --out ./graph/schema/generated/schema.graphql \
  --strip-suffix "DTO,Entity" \
  --add-type-prefix "Gql" \
  --add-input-suffix "Payload"
```

```go
/**
 * @gqlType
 * @gqlInput
 */
type UserDTO struct {
    ID   string
    Name string
}
```

Result:
- Struct: `UserDTO`
- Strip suffix "DTO": `UserDTO` → `User`
- Add type prefix "Gql": `User` → `GqlUser`
- Type name: **`GqlUser`**
- Input base: `User` (stripped)
- Add "Input": `User` → `UserInput`
- Add input suffix "Payload": `UserInput` → `UserInputPayload`
- Input name: **`UserInputPayload`**

Generates:
```graphql
type GqlUser @goModel(model: "domain.UserDTO") {
    id: String!
    name: String!
}

input UserInputPayload @goModel(model: "domain.UserDTO") {
    id: String!
    name: String!
}
```

**Notes:**
- Stripping only applies when `@gqlType` or `@gqlInput` doesn't specify a custom name
- For input types, the base name is stripped, then "Input" is appended
- Only one prefix and one suffix are stripped per type (first match wins)
- Use commas to separate multiple prefixes/suffixes

### Custom Scalar Types

Define custom scalars in your gqlgen.yml and use them in struct tags:

```go
type User struct {
    Email     string    `gql:"type:Email"`
    URL       string    `gql:"type:URL"`
    JSON      string    `gql:"type:JSONString"`
    Date      time.Time `gql:"type:Date"`
}
```

### Embedded Struct Support

The generator automatically expands embedded struct fields into the parent GraphQL type. This allows you to compose types from reusable components.

#### Basic Embedded Structs

```go
// Base contains common fields
type Base struct {
    ID        string `json:"id"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}

/**
 * @gqlType(name:"Article")
 */
type Article struct {
    Base           // Embedded struct - fields will be expanded
    Title   string `json:"title"`
    Content string `json:"content"`
    Author  string `json:"author"`
}
```

**Generated:**
```graphql
type Article {
    id: String!
    created_at: String!
    updated_at: String!
    title: String!
    content: String!
    author: String!
}
```

#### Multiple Embedded Structs

You can embed multiple structs in a single type:

```go
// Timestamped provides timestamp fields
type Timestamped struct {
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}

// Identifiable provides ID field
type Identifiable struct {
    ID string `json:"id"`
}

/**
 * @gqlType(name:"BlogPost")
 * @gqlInput(name:"BlogPostInput")
 */
type BlogPost struct {
    Identifiable // Embedded - ID field will be included
    Timestamped  // Embedded - CreatedAt and UpdatedAt will be included
    Title        string `json:"title"`
    Body         string `json:"body"`
    Published    bool   `json:"published"`
}
```

**Generated:**
```graphql
type BlogPost {
    id: String!
    created_at: String!
    updated_at: String!
    title: String!
    body: String!
    published: Boolean!
}

input BlogPostInput {
    id: String!
    created_at: String!
    updated_at: String!
    title: String!
    body: String!
    published: Boolean!
}
```

#### Embedded Structs with Field Annotations

Field annotations on embedded struct fields are preserved:

```go
// Metadata provides metadata fields
type Metadata struct {
    /**
     * @gqlField(description:"Tags for categorization")
     */
    Tags []string `json:"tags"`
    /**
     * @gqlField(name:"viewCount",description:"Number of views")
     */
    Views int `json:"views"`
}

/**
 * @gqlType(name:"ContentWithMetadata",description:"Content with embedded metadata")
 */
type ContentWithMetadata struct {
    Base
    Metadata
    Title       string `json:"title"`
    Description string `json:"description"`
}
```

**Generated:**
```graphql
"""Content with embedded metadata"""
type ContentWithMetadata {
    id: String!
    created_at: String!
    updated_at: String!
    """Tags for categorization"""
    tags: [String!]!
    """Number of views"""
    viewCount: Int!
    title: String!
    description: String!
}
```

#### Notes on Embedded Structs

- Embedded struct fields are recursively expanded into the parent type
- All field annotations (`@gqlField`, struct tags) on embedded fields are respected
- Field naming rules (case transformation, JSON tags) apply to embedded fields
- Embedded structs themselves don't need GraphQL annotations
- Only embedded structs found in the scanned packages are expanded
- Pointer embedded structs (`*Base`) are also supported

## Integration with gqlgen

### Plugin Integration

When used as a gqlgen plugin, the struct scanner runs automatically before gqlgen's code generation. This provides a seamless workflow:

1. **Schema Generation**: Scan Go structs → Generate `.graphqls` files
2. **Code Generation**: gqlgen reads schemas → Generate resolvers and models

**Benefits:**
- Single command to generate everything: `go run github.com/99designs/gqlgen generate`
- Schemas stay in sync with your domain models
- No manual schema writing for basic CRUD types
- Full type safety from Go structs to GraphQL

### Example Workflow

```bash
# 1. Configure gqlgen.yml (see examples/gqlgen.yml.example)
# 2. Add annotations to your Go structs
# 3. Run gqlgen
go run github.com/99designs/gqlgen generate

# The plugin will:
# - Scan your packages
# - Generate GraphQL schemas
# - Then gqlgen generates resolvers
```

### Hybrid Approach

You can combine auto-generated and hand-written schemas:

```yaml
# gqlgen.yml
schema:
  - graph/schema/generated/*.graphqls  # Auto-generated from structs
  - graph/schema/custom/*.graphql      # Hand-written queries/mutations
```

This lets you:
- Auto-generate types from domain models
- Manually write queries, mutations, and subscriptions
- Add custom types not backed by Go structs

### See Also

- `examples/gqlgen.yml.example` - Full gqlgen configuration example
- [gqlgen documentation](https://gqlgen.com) - Official gqlgen docs

## Troubleshooting

### Type Not Generated

- Check if struct is public (starts with uppercase)
- Verify package is in the scan path
- Check for `@gqlIgnoreAll` without `include:` tags
- Ensure `include-empty-types` is true if type has no fields

### Wrong Field Names

- Check `-field-case` flag or `field_case` in plugin config
- Verify struct tags (`name:` takes precedence)
- If using `-use-json-tag`, check json tags

### Missing @goModel Directive

- Ensure `-use-gqlgen-directives=true` (default)
- Or add `@gqlUseModelDirective` annotation to specific types

### Fields Not Showing Descriptions

- Descriptions come from `description:` struct tag
- Or from `@gqlType(description:` type-level annotation

## Contributing

When adding new features:

1. Update `Config` struct in `config.go`
2. Add parsing logic in `directives.go`
3. Implement generation in `generator.go`
4. Update this documentation
5. Add examples

## License

MIT License. See [LICENSE](LICENSE.md) for details.

--- 

Made with ❤️ by Pablo Ramirez <pablo@pramirez.dev> | [Website](https://pramirez.dev) | [GitHub](https://github.com/pablor21)
