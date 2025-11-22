# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-11-21

### Added
- GraphQL schema generation from Go structs with annotations
- Support for `@gqlType`, `@gqlInput`, `@gqlEnum` directives
- Advanced field filtering with `ro`, `wo`, `rw`, `include`, `omit` tags
- Scalar mappings for global Go type to GraphQL scalar conversion
- Auto-generation strategies (none, referenced, all, patterns)
- Full Go generics support (1.18+)
- Namespace organization for multi-file schemas
- `@GqlKeepBegin`/`@GqlKeepEnd` markers for preserving manual edits
- gqlgen integration with `@goModel` and `@goField` directives
- CLI commands: `init`, `generate`
- VS Code extension integration
- Multiple type generation from single struct
- Field-level visibility control
- Cross-package enum support
- Type-specific field filtering
- Generic type parameter substitution

### Documentation
- Complete documentation site
- Getting started guide
- Configuration reference
- Feature documentation for all major features
- CLI reference
- Integration guides (gqlgen, VS Code)

[1.0.0]: https://github.com/pablor21/gqlschemagen/releases/tag/v1.0.0