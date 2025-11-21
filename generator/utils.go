package generator

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// var GQLKEEP_REGEX = regexp.MustCompile(`(?s)# @gqlKeepBegin(.*?)# @gqlKeepEnd(?s)`)

// EnsureDir makes sure a directory exists
func EnsureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// FieldTypeName returns the bare type name used for nested type lookup
func FieldTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return FieldTypeName(t.X)
	case *ast.ArrayType:
		return FieldTypeName(t.Elt)
	case *ast.SelectorExpr:
		// package.Type -> Type
		return t.Sel.Name
	default:
		return ""
	}
}

// ExprToGraphQLType converts an ast.Expr to a GraphQL type string (with ! for required)
// This is a convenience wrapper that calls ExprToGraphQLTypeWithContext with nil context
func ExprToGraphQLType(expr ast.Expr) string {
	return ExprToGraphQLTypeWithContext(expr, nil, nil, nil)
}

// ExprToGraphQLTypeWithContext converts an ast.Expr to a GraphQL type string with context support
// The context provides type substitutions for generic type parameters and config for unresolved types
func ExprToGraphQLTypeWithContext(expr ast.Expr, config *Config, ctx *GenerationContext, gen *Generator) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// FIRST: Check if this identifier has a substitution in context (for generic type parameters)
		if ctx != nil && ctx.TypeSubstitutions != nil {
			if substitutedExpr, exists := ctx.TypeSubstitutions[t.Name]; exists {
				// Recursively convert the substituted expression
				// This handles Result[*User] where T maps to *User expression
				return ExprToGraphQLTypeWithContext(substitutedExpr, config, ctx, gen)
			}
		}

		// SECOND: Check built-in types
		switch t.Name {
		case "string":
			return "String!"
		case "int", "int32":
			return "Int!"
		case "int64":
			return "Int64!"
		case "float32", "float64":
			return "Float!"
		case "bool":
			return "Boolean!"
		case "interface{}":
			return "JSON!"
		case "Time", "time.Time":
			return "DateTime!"
		default:
			// THIRD: Check if it's an unresolved type parameter and config specifies replacement
			if config != nil && config.AutoGenerate.UnresolvedGenericType != "" {
				// Use simple heuristic for type parameters
				if isLikelyTypeParameter(t.Name) {
					return config.AutoGenerate.UnresolvedGenericType + "!"
				}
			}
			return t.Name + "!"
		}
	case *ast.StarExpr:
		// Pass context through
		return ExprToGraphQLTypeWithContext(t.X, config, ctx, gen)
	case *ast.ArrayType:
		return "[" + ExprToGraphQLTypeWithContext(t.Elt, config, ctx, gen) + "]!"
	case *ast.SelectorExpr:
		return t.Sel.Name + "!"
	case *ast.IndexExpr:
		// Handle generic instantiation like Repository[Post] or Edge[Comment]
		// First, resolve the type argument through the context (for nested generics like Edge[T] where T=*Comment)
		typeArg := t.Index
		if ctx != nil && ctx.TypeSubstitutions != nil {
			if ident, ok := typeArg.(*ast.Ident); ok {
				if substitutedExpr, exists := ctx.TypeSubstitutions[ident.Name]; exists {
					typeArg = substitutedExpr
				}
			}
		}

		// Track this instantiation and generate a concrete type name
		baseName := extractBaseTypeName(t.X)
		if gen != nil && baseName != "" {
			concreteTypeName := gen.trackGenericInstantiation(baseName, []ast.Expr{typeArg}, ctx)
			return concreteTypeName + "!"
		}
		return ExprToGraphQLTypeWithContext(t.X, config, ctx, gen)
	case *ast.IndexListExpr:
		// Handle generic instantiation with multiple params like Map[K, V]
		// Resolve each type argument through the context
		typeArgs := make([]ast.Expr, len(t.Indices))
		for i, arg := range t.Indices {
			typeArgs[i] = arg
			if ctx != nil && ctx.TypeSubstitutions != nil {
				if ident, ok := arg.(*ast.Ident); ok {
					if substitutedExpr, exists := ctx.TypeSubstitutions[ident.Name]; exists {
						typeArgs[i] = substitutedExpr
					}
				}
			}
		}

		baseName := extractBaseTypeName(t.X)
		if gen != nil && baseName != "" {
			concreteTypeName := gen.trackGenericInstantiation(baseName, typeArgs, ctx)
			return concreteTypeName + "!"
		}
		return ExprToGraphQLTypeWithContext(t.X, config, ctx, gen)
	default:
		return "String!"
	}
}

// extractBaseTypeName extracts the base type name from an expression
func extractBaseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractBaseTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// isLikelyTypeParameter checks if a name looks like a generic type parameter
// This is a simple heuristic used when we don't have access to the Generator
func isLikelyTypeParameter(name string) bool {
	// Common single-letter type parameters
	commonParams := map[string]bool{
		"T": true, "K": true, "V": true, "E": true,
		"U": true, "R": true, "S": true,
	}
	if commonParams[name] {
		return true
	}

	// Single uppercase letter (A-Z)
	if len(name) == 1 && name[0] >= 'A' && name[0] <= 'Z' {
		return true
	}

	return false
}

// ExprToGraphQLTypeForInput converts an ast.Expr to a GraphQL type string for input types
// It transforms custom type references to input references (e.g., Address -> AddressInput)
// Built-in types and enums remain unchanged
// This is a convenience wrapper that calls ExprToGraphQLTypeForInputWithContext with nil context
func ExprToGraphQLTypeForInput(expr ast.Expr, knownScalars []string, enumTypes map[string]*EnumType) string {
	return ExprToGraphQLTypeForInputWithContext(expr, knownScalars, enumTypes, nil, nil, nil)
}

// ExprToGraphQLTypeForInputWithContext converts with context support for type substitutions
func ExprToGraphQLTypeForInputWithContext(expr ast.Expr, knownScalars []string, enumTypes map[string]*EnumType, config *Config, ctx *GenerationContext, gen *Generator) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// FIRST: Check if this identifier has a substitution in context (for generic type parameters)
		if ctx != nil && ctx.TypeSubstitutions != nil {
			if substitutedExpr, exists := ctx.TypeSubstitutions[t.Name]; exists {
				// Recursively convert the substituted expression
				return ExprToGraphQLTypeForInputWithContext(substitutedExpr, knownScalars, enumTypes, config, ctx, gen)
			}
		}

		// SECOND: Check built-in types
		switch t.Name {
		case "string":
			return "String!"
		case "int", "int32":
			return "Int!"
		case "int64":
			return "Int64!"
		case "float32", "float64":
			return "Float!"
		case "bool":
			return "Boolean!"
		case "interface{}":
			return "JSON!"
		case "Time", "time.Time":
			return "DateTime!"
		default:
			// THIRD: Check if it's an unresolved type parameter
			if config != nil && config.AutoGenerate.UnresolvedGenericType != "" && isLikelyTypeParameter(t.Name) {
				return config.AutoGenerate.UnresolvedGenericType + "!"
			}

			// Check if it's an enum first (enums don't need Input suffix)
			if enumTypes != nil {
				if _, isEnum := enumTypes[t.Name]; isEnum {
					return t.Name + "!"
				}
			}
			// Check if it's a known scalar
			for _, scalar := range knownScalars {
				if scalar == t.Name {
					return t.Name + "!"
				}
			}
			// It's a custom type, convert to input
			return t.Name + "Input!"
		}
	case *ast.StarExpr:
		// Pass context through
		return ExprToGraphQLTypeForInputWithContext(t.X, knownScalars, enumTypes, config, ctx, gen)
	case *ast.ArrayType:
		return "[" + ExprToGraphQLTypeForInputWithContext(t.Elt, knownScalars, enumTypes, config, ctx, gen) + "]!"
	case *ast.SelectorExpr:
		// Check if it's an enum
		if enumTypes != nil {
			if _, isEnum := enumTypes[t.Sel.Name]; isEnum {
				return t.Sel.Name + "!"
			}
		}
		return t.Sel.Name + "Input!"
	case *ast.IndexExpr:
		// Handle generic instantiation - track and generate concrete type with Input suffix
		// First, resolve the type argument through the context (for nested generics)
		typeArg := t.Index
		if ctx != nil && ctx.TypeSubstitutions != nil {
			if ident, ok := typeArg.(*ast.Ident); ok {
				if substitutedExpr, exists := ctx.TypeSubstitutions[ident.Name]; exists {
					typeArg = substitutedExpr
				}
			}
		}

		baseName := extractBaseTypeName(t.X)
		if gen != nil && baseName != "" {
			concreteTypeName := gen.trackGenericInstantiation(baseName, []ast.Expr{typeArg}, ctx)
			return concreteTypeName + "Input!"
		}
		return ExprToGraphQLTypeForInputWithContext(t.X, knownScalars, enumTypes, config, ctx, gen)
	case *ast.IndexListExpr:
		// Handle generic instantiation with multiple params
		// Resolve each type argument through the context
		typeArgs := make([]ast.Expr, len(t.Indices))
		for i, arg := range t.Indices {
			typeArgs[i] = arg
			if ctx != nil && ctx.TypeSubstitutions != nil {
				if ident, ok := arg.(*ast.Ident); ok {
					if substitutedExpr, exists := ctx.TypeSubstitutions[ident.Name]; exists {
						typeArgs[i] = substitutedExpr
					}
				}
			}
		}

		baseName := extractBaseTypeName(t.X)
		if gen != nil && baseName != "" {
			concreteTypeName := gen.trackGenericInstantiation(baseName, typeArgs, ctx)
			return concreteTypeName + "Input!"
		}
		return ExprToGraphQLTypeForInputWithContext(t.X, knownScalars, enumTypes, config, ctx, gen)
	default:
		return "String!"
	}
}

func WriteFile(path, content string, config *Config) error {
	// Ensure parent dir exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	// check for # @gqlKeepBegin and # @gqlKeepEnd markers to preserve content (can have multiple)
	if FileExists(path) {
		var preservedSections []string
		var gqlKeepRegex = regexp.MustCompile(`(?s)` + config.KeepBeginMarker + `(.*?)` + config.KeepEndMarker + `(?s)`)
		existingContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		existingStr := string(existingContent)
		matches := gqlKeepRegex.FindAllStringSubmatch(existingStr, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			innerContent := match[1]
			// Replace the corresponding section in content
			preservedSections = append(preservedSections, innerContent)
		}

		if len(preservedSections) > 0 {
			combinedSections := config.KeepBeginMarker + strings.Join(preservedSections, "\n") + config.KeepEndMarker

			// Depending on placement config, insert preserved sections
			if config.KeepSectionPlacement == "start" {
				content = combinedSections + "\n" + content
			} else {
				content = content + "\n" + combinedSections + "\n"
			}
		} else {
			placeholder := config.KeepBeginMarker + "\n# You can add custom types or comments here and they will be preserved during code generation.\n" + config.KeepEndMarker
			// Depending on placement config, insert placeholder markers
			if config.KeepSectionPlacement == "start" {
				content = placeholder + "\n\n" + content
			} else {
				content = content + "\n\n" + placeholder + "\n\n"
			}
		}
	}

	// add a notice at the top
	content = "# Code generated by https://github.com/pablor21/gqlschemagen " + GetVersion() + ".\r\n" +
		"# PUT YOUR CUSTOM CONTENT BETWEEN @gqlKeep(Begin|End) markers, see:  https://github.com/pablor21/gqlschemagen#keeping-schema-modifications \n" + content

	// Write file (atomic write could be added if desired)
	return os.WriteFile(path, []byte(content), 0o644)
}

// helper to normalize package dir path for go run usage
func PkgDir(in string) string {
	if strings.HasPrefix(in, "./") || strings.HasPrefix(in, "/") {
		return in
	}
	// fallback - treat as local path
	return in
}

// ExprToGoType extracts the Go type name from an ast.Expr
// This returns the type as it appears in Go code (e.g., "outofscope.AnotherOutOfScope", "*User", "[]string")
func ExprToGoType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + ExprToGoType(t.X)
	case *ast.ArrayType:
		return "[]" + ExprToGoType(t.Elt)
	case *ast.SelectorExpr:
		// For package-qualified types like "outofscope.AnotherOutOfScope"
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.MapType:
		return "map[" + ExprToGoType(t.Key) + "]" + ExprToGoType(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}
