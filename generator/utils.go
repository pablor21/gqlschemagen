package generator

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDir makes sure a directory exists
func EnsureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
func ExprToGraphQLType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
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
			return t.Name + "!"
		}
	case *ast.StarExpr:
		return ExprToGraphQLType(t.X)
	case *ast.ArrayType:
		return "[" + ExprToGraphQLType(t.Elt) + "]!"
	case *ast.SelectorExpr:
		return t.Sel.Name + "!"
	default:
		return "String!"
	}
}

func WriteFile(path, content string) error {
	// Ensure parent dir exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

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
