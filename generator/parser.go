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
	PackagePath string      // Full import path where this const is defined
	PackageName string      // Package name where this const is defined
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
	// Enum candidates collected across all files before matching
	enumCandidates map[string]*enumCandidate
	// Const blocks collected for later matching to enum types
	constBlocks []*constBlockInfo
	// Namespace support - maps type/enum name to namespace
	TypeNamespaces map[string]string // type name -> namespace
	EnumNamespaces map[string]string // enum name -> namespace
}

func NewParser() *Parser {
	return &Parser{
		StructTypes:    make(map[string]*ast.TypeSpec),
		Structs:        make(map[string]*ast.StructType),
		PackageNames:   make(map[string]string),
		PackagePaths:   make(map[string]string),
		TypeToDecl:     make(map[string]*ast.GenDecl),
		EnumTypes:      make(map[string]*EnumType),
		enumCandidates: make(map[string]*enumCandidate),
		constBlocks:    make([]*constBlockInfo, 0),
		TypeNamespaces: make(map[string]string),
		EnumNamespaces: make(map[string]string),
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
			if err := p.parseFile(root); err != nil {
				return err
			}
		}
	} else {
		// If it's a directory, recursively scan all Go files
		if err := p.walkDir(root); err != nil {
			return err
		}
	}

	return nil
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

	// Extract file-level namespace from comments after package declaration
	fileNamespace := extractFileNamespace(f)

	// First pass: collect type declarations (structs and potential enums)
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

					// Store namespace (file-level or type-level override)
					if fileNamespace != "" {
						p.TypeNamespaces[name] = fileNamespace
					}
					continue
				}

				// Check if it's a potential enum (type with @gqlEnum directive)
				if hasGqlEnumDirective(genDecl) {
					baseType := getBaseTypeName(t.Type)
					if baseType == "string" || baseType == "int" {
						// Store enum candidate for later matching
						p.enumCandidates[t.Name.Name] = &enumCandidate{
							TypeSpec:  t,
							GenDecl:   genDecl,
							BaseType:  baseType,
							PkgName:   pkgName,
							FilePath:  path,
							Namespace: fileNamespace,
						}
					}
				}
			}
		}
	}

	// Second pass: collect const blocks for later matching
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		// Store const block info for later matching
		p.constBlocks = append(p.constBlocks, &constBlockInfo{
			GenDecl:  genDecl,
			PkgName:  pkgName,
			FilePath: path,
		})
	}

	return nil
}

// enumCandidate holds info about a potential enum type before we find its const values
type enumCandidate struct {
	TypeSpec  *ast.TypeSpec
	GenDecl   *ast.GenDecl
	BaseType  string // "string" or "int"
	PkgName   string
	FilePath  string
	Namespace string // File-level namespace
}

// constBlockInfo holds info about a const block for later matching to enum types
type constBlockInfo struct {
	GenDecl  *ast.GenDecl
	PkgName  string
	FilePath string // File path where this const block is defined
}

func appendIfMissing(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// extractFileNamespace extracts namespace from file-level @gqlNamespace directive
// Looks for @gqlNamespace directive in comments anywhere in the file header
func extractFileNamespace(f *ast.File) string {
	// Check all comment groups in the file
	for _, commentGroup := range f.Comments {
		if ns := findNamespaceInComments(commentGroup); ns != "" {
			// Check if this comment appears before any type declarations
			// by comparing positions
			for _, decl := range f.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				if genDecl.Tok == token.TYPE {
					// If the comment is before the first type declaration, it's file-level
					if commentGroup.Pos() < genDecl.Pos() {
						return ns
					}
					break
				}
			}
		}
	}

	return ""
} // findNamespaceInComments looks for @gqlNamespace directive in comment group
func findNamespaceInComments(cg *ast.CommentGroup) string {
	for _, comment := range cg.List {
		text := comment.Text
		// Normalize block comments
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimPrefix(text, "/**")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)

			// Look for @gqlNamespace(name:"value")
			if strings.Contains(line, "@gqlNamespace") {
				if idx := strings.Index(line, "@gqlNamespace"); idx != -1 {
					rest := line[idx:]
					// Find name parameter
					if nameIdx := strings.Index(rest, "name:"); nameIdx != -1 {
						nameStart := rest[nameIdx+5:]
						// Extract quoted value
						nameStart = strings.TrimSpace(nameStart)
						if strings.HasPrefix(nameStart, `"`) {
							if endIdx := strings.Index(nameStart[1:], `"`); endIdx != -1 {
								return nameStart[1 : endIdx+1]
							}
						}
					}
				}
			}
		}
	}
	return ""
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

// GetPackageImportPathFromFile builds the import path from a file path and package name
func (p *Parser) GetPackageImportPathFromFile(filePath string, pkgName string, modelPath string) string {
	if modelPath == "" {
		// Just return package name if no model path configured
		return pkgName
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
func (p *Parser) parseConstBlock(constBlock *constBlockInfo, enumCandidates map[string]*enumCandidate) {
	genDecl := constBlock.GenDecl

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
			// Handle both qualified (pkg.Type) and unqualified (Type) names
			var typeName string
			switch t := valueSpec.Type.(type) {
			case *ast.Ident:
				// Unqualified type: Status
				typeName = t.Name
			case *ast.SelectorExpr:
				// Qualified type: types.Status or pkg.Status
				typeName = t.Sel.Name
			}

			if typeName != "" {
				if candidate, exists := enumCandidates[typeName]; exists {
					enumTypeName = typeName
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
				PackagePath: constBlock.FilePath,
				PackageName: constBlock.PkgName,
			})

			iotaValue++
		}
	}

	// Create the EnumType and store it
	if len(values) > 0 {
		// Parse @gqlEnum directive for custom name, description, and namespace
		enumName, enumDesc, enumNamespace := parseEnumDirective(candidate.GenDecl.Doc, enumTypeName)

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

		// Store namespace: enum-level override takes precedence over file-level
		if enumNamespace != "" {
			p.EnumNamespaces[enumTypeName] = enumNamespace
		} else if candidate.Namespace != "" {
			p.EnumNamespaces[enumTypeName] = candidate.Namespace
		}
	}
}

// MatchEnumConstants matches all collected const blocks to enum candidates
// This should be called after all packages have been parsed to support cross-file and cross-package enums
func (p *Parser) MatchEnumConstants() {
	// Process all collected const blocks
	for _, constBlock := range p.constBlocks {
		p.parseConstBlock(constBlock, p.enumCandidates)
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

// parseEnumDirective extracts custom name, description, and namespace from @gqlEnum directive
// Returns (customName or defaultName, description, namespace)
func parseEnumDirective(commentGroup *ast.CommentGroup, defaultName string) (string, string, string) {
	name := defaultName
	var description string
	var namespace string

	if commentGroup == nil {
		return name, description, namespace
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
				// Extract namespace parameter if present
				if ns := extractDirectiveParam(line, "namespace"); ns != "" {
					namespace = ns
				}
			} else if !strings.HasPrefix(line, "@") && line != "" && description == "" {
				// Use regular comment as description if no @gqlEnum description
				description = line
			}
		}
	}

	return name, description, namespace
}
