package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EnumValue represents a single value in an enum
type EnumValue struct {
	GoName      string      // e.g., "PermissionRead"
	GraphQLName string      // e.g., "READ"
	Value       interface{} // The actual value (string literal or int)
	Description string      // From comment
	Deprecated  string      // Deprecation reason if any
}

// EnumType represents a Go enum type
type EnumType struct {
	Name        string // GraphQL enum name (custom or derived from Go type)
	GoTypeName  string // Original Go type name
	BaseType    string // "string" or "int"
	Description string
	Values      []EnumValue
	TypeSpec    *ast.TypeSpec
	GenDecl     *ast.GenDecl
}

// Parser collects type specs and related AST nodes across a root dir
type Parser struct {
	StructTypes  map[string]*ast.TypeSpec
	Structs      map[string]*ast.StructType
	PackageNames map[string]string
	PackagePaths map[string]string // Full import path for each type
	TypeToDecl   map[string]*ast.GenDecl
	// ordered list of type names for deterministic output
	TypeNames []string
	// Enum support
	EnumTypes map[string]*EnumType
	EnumNames []string // ordered list for deterministic output
}

func NewParser() *Parser {
	return &Parser{
		StructTypes:  make(map[string]*ast.TypeSpec),
		Structs:      make(map[string]*ast.StructType),
		PackageNames: make(map[string]string),
		PackagePaths: make(map[string]string),
		TypeToDecl:   make(map[string]*ast.GenDecl),
		EnumTypes:    make(map[string]*EnumType),
	}
}

func (p *Parser) Walk(root string) error {
	// Clean the path
	root = filepath.Clean(root)

	// Check if path exists
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", root)
		}
		return fmt.Errorf("failed to access path %s: %w", root, err)
	}

	// If it's a file, parse it directly
	if !info.IsDir() {
		if strings.HasSuffix(root, ".go") && !strings.HasSuffix(root, "_test.go") {
			return p.parseFile(root)
		}
		return nil
	}

	// If it's a directory, recursively scan all Go files
	return p.walkDir(root)
}

func (p *Parser) walkDir(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			if err := p.parseFile(path); err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
		}
		return nil
	})
}

func (p *Parser) parseFile(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	pkgName := f.Name.Name

	// First pass: collect type declarations (structs and potential enums)
	enumCandidates := make(map[string]*enumCandidate) // type name -> candidate info

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				t, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				// Check if it's a struct
				if s, ok := t.Type.(*ast.StructType); ok {
					name := t.Name.Name
					p.StructTypes[name] = t
					p.Structs[name] = s
					p.PackageNames[name] = pkgName
					p.PackagePaths[name] = path
					p.TypeToDecl[name] = genDecl
					p.TypeNames = appendIfMissing(p.TypeNames, name)
					continue
				}

				// Check if it's a potential enum (type with @gqlEnum directive)
				if hasGqlEnumDirective(genDecl) {
					baseType := getBaseTypeName(t.Type)
					if baseType == "string" || baseType == "int" {
						enumCandidates[t.Name.Name] = &enumCandidate{
							TypeSpec: t,
							GenDecl:  genDecl,
							BaseType: baseType,
							PkgName:  pkgName,
							FilePath: path,
						}
					}
				}
			}
		}
	}

	// Second pass: find const blocks that define enum values
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		p.parseConstBlock(genDecl, enumCandidates, fset)
	}

	return nil
}

// enumCandidate holds info about a potential enum type before we find its const values
type enumCandidate struct {
	TypeSpec *ast.TypeSpec
	GenDecl  *ast.GenDecl
	BaseType string // "string" or "int"
	PkgName  string
	FilePath string
}

func appendIfMissing(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// GetPackageImportPath returns the full import path for a type
// If modelPath is provided and looks like a complete path, use it directly
// Otherwise, try to build the path by analyzing the file structure
func (p *Parser) GetPackageImportPath(typeName string, modelPath string) string {
	pkgName := p.PackageNames[typeName]

	if modelPath == "" {
		// Just return package name if no model path configured
		return pkgName
	}

	filePath, ok := p.PackagePaths[typeName]
	if !ok {
		// No file path info, just use modelPath directly
		return modelPath
	}

	// Get the directory of the file
	dir := filepath.ToSlash(filepath.Dir(filePath))
	parts := strings.Split(dir, "/")

	// Find the index where the package name appears
	pkgIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == pkgName {
			pkgIndex = i
			break
		}
	}

	if pkgIndex == -1 {
		// Package directory not found in path, use modelPath as-is
		return modelPath
	}

	// Check if there are meaningful parent directories between module root and package
	// Look for structure like: internal/models, pkg/entities, api/v2/models, etc.
	var subPath []string

	// Collect directories from package backward until we hit a likely module boundary
	for i := pkgIndex; i >= 0; i-- {
		part := parts[i]

		// Skip empty parts
		if part == "" || part == "." || part == ".." {
			continue
		}

		subPath = append([]string{part}, subPath...)

		// Stop if we hit common module structure markers (but include them)
		if part == "internal" || part == "pkg" || part == "cmd" || part == "api" {
			break
		}
	}

	// If we only have the package name itself, return modelPath as-is
	// This handles cases where modelPath already points to the complete package location
	if len(subPath) == 1 && subPath[0] == pkgName {
		return modelPath
	}

	// Otherwise, append the sub-path to modelPath
	if len(subPath) > 0 {
		return modelPath + "/" + strings.Join(subPath, "/")
	}

	return modelPath
}

// hasGqlEnumDirective checks if a GenDecl has @gqlEnum directive in its doc comments
func hasGqlEnumDirective(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}
	for _, comment := range decl.Doc.List {
		if strings.Contains(comment.Text, "@gqlEnum") {
			return true
		}
	}
	return false
}

// getBaseTypeName extracts the base type name from a type expression
func getBaseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

// parseConstBlock parses a const block and matches values to enum candidates
func (p *Parser) parseConstBlock(genDecl *ast.GenDecl, enumCandidates map[string]*enumCandidate, fset *token.FileSet) {
	if len(genDecl.Specs) == 0 {
		return
	}

	// Determine which enum type this const block belongs to
	var enumTypeName string
	var enumType string

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Get the type from the first const that has an explicit type
		if valueSpec.Type != nil {
			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				enumTypeName = ident.Name
				if candidate, exists := enumCandidates[enumTypeName]; exists {
					enumType = candidate.BaseType
					break
				}
			}
		}
	}

	if enumTypeName == "" {
		return // Not related to any enum candidate
	}

	candidate := enumCandidates[enumTypeName]
	if candidate == nil {
		return
	}

	// Parse the const values
	var values []EnumValue
	iotaValue := 0

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			goName := name.Name

			// Extract value
			var value interface{}
			if i < len(valueSpec.Values) {
				value = p.extractConstValue(valueSpec.Values[i], iotaValue, enumType)
			} else {
				// No explicit value, use iota for int or empty for string
				if enumType == "int" {
					value = iotaValue
				}
			}

			// Extract GraphQL name and description from comment
			graphQLName, description, deprecated := p.parseValueDirective(valueSpec.Comment, goName, enumTypeName)

			values = append(values, EnumValue{
				GoName:      goName,
				GraphQLName: graphQLName,
				Value:       value,
				Description: description,
				Deprecated:  deprecated,
			})

			iotaValue++
		}
	}

	// Create the EnumType and store it
	if len(values) > 0 {
		// Parse @gqlEnum directive for custom name and description
		enumName, enumDesc := parseEnumDirective(candidate.GenDecl.Doc, enumTypeName)

		enumType := &EnumType{
			Name:        enumName,
			GoTypeName:  enumTypeName,
			BaseType:    candidate.BaseType,
			Description: enumDesc,
			Values:      values,
			TypeSpec:    candidate.TypeSpec,
			GenDecl:     candidate.GenDecl,
		}

		p.EnumTypes[enumTypeName] = enumType
		p.EnumNames = appendIfMissing(p.EnumNames, enumTypeName)
		p.PackageNames[enumTypeName] = candidate.PkgName
		p.PackagePaths[enumTypeName] = candidate.FilePath
	}
}

// extractConstValue extracts the value from a const value expression
func (p *Parser) extractConstValue(expr ast.Expr, iotaValue int, baseType string) interface{} {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			// Remove quotes from string literal
			return strings.Trim(v.Value, "\"")
		}
		if v.Kind == token.INT {
			// Parse int literal
			var intVal int
			_, _ = fmt.Sscanf(v.Value, "%d", &intVal)
			return intVal
		}
	case *ast.Ident:
		if v.Name == "iota" {
			return iotaValue
		}
	case *ast.BinaryExpr:
		// Handle expressions like: iota * 10, iota + 1, 1 << iota
		// For simplicity, we'll just use the iota value
		// More complex evaluation could be added if needed
		return iotaValue
	case *ast.UnaryExpr:
		// Handle expressions like: -1
		return iotaValue
	}

	// Default: return iota value for int, empty string for string
	if baseType == "int" {
		return iotaValue
	}
	return ""
}

// parseValueDirective extracts @gqlEnumValue directive from comment
func (p *Parser) parseValueDirective(commentGroup *ast.CommentGroup, goName string, enumTypeName string) (graphQLName, description, deprecated string) {
	// Default: auto-generate GraphQL name by stripping enum type prefix
	graphQLName = stripEnumPrefix(goName, enumTypeName)

	if commentGroup == nil {
		return
	}

	hasDirective := false

	for _, comment := range commentGroup.List {
		text := comment.Text

		// Extract @gqlEnumValue directive
		if strings.Contains(text, "@gqlEnumValue") {
			hasDirective = true
			// Parse @gqlEnumValue(name:"CUSTOM_NAME", description:"...", deprecated:"...")
			if name := extractDirectiveParam(text, "name"); name != "" {
				graphQLName = name
			}
			if desc := extractDirectiveParam(text, "description"); desc != "" {
				description = desc
			}
			if depr := extractDirectiveParam(text, "deprecated"); depr != "" {
				deprecated = depr
			}
		} else if !hasDirective {
			// Only use regular comment as description if no @gqlEnumValue directive was found
			desc := strings.TrimPrefix(text, "//")
			desc = strings.TrimSpace(desc)
			if desc != "" && description == "" {
				description = desc
			}
		}
	}

	return
}

// stripEnumPrefix removes the enum type name prefix from a const name
// e.g., PermissionRead -> READ, ColorRed -> RED
func stripEnumPrefix(constName, enumTypeName string) string {
	if strings.HasPrefix(constName, enumTypeName) {
		stripped := strings.TrimPrefix(constName, enumTypeName)
		if stripped != "" {
			// Convert to UPPER_SNAKE_CASE
			return toScreamingSnakeCase(stripped)
		}
	}
	return toScreamingSnakeCase(constName)
}

// toScreamingSnakeCase converts camelCase to SCREAMING_SNAKE_CASE
func toScreamingSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToUpper(string(result))
}

// parseEnumDirective extracts custom name and description from @gqlEnum directive
// Returns (customName or defaultName, description)
func parseEnumDirective(commentGroup *ast.CommentGroup, defaultName string) (string, string) {
	name := defaultName
	var description string

	if commentGroup == nil {
		return name, description
	}

	for _, comment := range commentGroup.List {
		text := comment.Text
		// Normalize block comments
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimPrefix(text, "/**")
		text = strings.TrimSuffix(text, "*/")

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "//")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)

			// Check for @gqlEnum directive
			if strings.HasPrefix(line, "@gqlEnum") {
				// Extract name parameter if present
				if customName := extractDirectiveParam(line, "name"); customName != "" {
					name = customName
				}
				// Extract description parameter if present
				if desc := extractDirectiveParam(line, "description"); desc != "" {
					description = desc
				}
			} else if !strings.HasPrefix(line, "@") && line != "" && description == "" {
				// Use regular comment as description if no @gqlEnum description
				description = line
			}
		}
	}

	return name, description
}
