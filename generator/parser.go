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

	"golang.org/x/tools/go/packages"
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
	PackageNames map[string]string
	PackagePaths map[string]string // Full import path for each type
	SourceFiles  map[string]string // Source file path for each type (absolute OS path)
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
	// Source files for enums
	EnumSourceFiles map[string]string // enum name -> source file path
	// Type parameters for generic types
	TypeParameters map[string][]string // type name -> parameter names (e.g., "Result" -> ["T"], "Map" -> ["K", "V"])
	// Scanned types registry - tracks all types we've scanned with their GQL annotations
	// Key: Go type name, Value: metadata about the scanned type
	ScannedTypes map[string]*ScannedTypeInfo
	// Package cache for import path resolution
	pkgCache map[string]*packages.Package // dir path -> package info
	// External types loaded on-demand (not from scanned packages)
	ExternalTypes map[string]bool // type name -> true if loaded on-demand from external package
}

// ScannedTypeInfo stores metadata about a scanned type
type ScannedTypeInfo struct {
	TypeName            string   // Go type name
	HasTypeDirective    bool     // Has @gqlType annotation
	HasInputDirective   bool     // Has @gqlInput annotation
	HasIncludeDirective bool     // Has @gqlInclude annotation
	IsExternal          bool     // Loaded on-demand from external package (not in scanned packages)
	GeneratedTypes      []string // List of GraphQL type names generated from this struct (from @gqlType)
	GeneratedInputs     []string // List of GraphQL input names generated from this struct (from @gqlInput)
}

func NewParser() *Parser {
	return &Parser{
		StructTypes:     make(map[string]*ast.TypeSpec),
		PackageNames:    make(map[string]string),
		PackagePaths:    make(map[string]string),
		SourceFiles:     make(map[string]string),
		TypeToDecl:      make(map[string]*ast.GenDecl),
		EnumTypes:       make(map[string]*EnumType),
		enumCandidates:  make(map[string]*enumCandidate),
		constBlocks:     make([]*constBlockInfo, 0),
		TypeNamespaces:  make(map[string]string),
		EnumNamespaces:  make(map[string]string),
		EnumSourceFiles: make(map[string]string),
		TypeParameters:  make(map[string][]string),
		ScannedTypes:    make(map[string]*ScannedTypeInfo),
		pkgCache:        make(map[string]*packages.Package),
		ExternalTypes:   make(map[string]bool),
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

	// Get the package import path for this file
	pkgImportPath := p.GetPackageImportPathFromFile(path, pkgName, "")

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
				if _, ok := t.Type.(*ast.StructType); ok {
					name := t.Name.Name
					p.StructTypes[name] = t
					p.PackageNames[name] = pkgName
					p.PackagePaths[name] = pkgImportPath // Store import path, not file path
					p.SourceFiles[name] = path           // Store source file path
					p.TypeToDecl[name] = genDecl
					p.TypeNames = appendIfMissing(p.TypeNames, name)

					// Extract and store type parameters if this is a generic type
					if t.TypeParams != nil {
						var params []string
						for _, field := range t.TypeParams.List {
							for _, paramName := range field.Names {
								params = append(params, paramName.Name)
							}
						}
						if len(params) > 0 {
							p.TypeParameters[name] = params
						}
					}

					// Store namespace (file-level or type-level override)
					if fileNamespace != "" {
						p.TypeNamespaces[name] = fileNamespace
					}
					continue
				}

				// Check if it's a type alias to a generic instantiation
				if _, ok := t.Type.(*ast.IndexListExpr); ok {
					name := t.Name.Name
					// Store it as if it were a struct so it can be processed
					p.StructTypes[name] = t
					p.PackageNames[name] = pkgName
					p.PackagePaths[name] = pkgImportPath
					p.SourceFiles[name] = path
					p.TypeToDecl[name] = genDecl
					p.TypeNames = appendIfMissing(p.TypeNames, name)

					// Store namespace if present
					if fileNamespace != "" {
						p.TypeNamespaces[name] = fileNamespace
					}
					continue
				}

				// Also handle single index expression (Go 1.18 style)
				if _, ok := t.Type.(*ast.IndexExpr); ok {
					name := t.Name.Name
					p.StructTypes[name] = t
					p.PackageNames[name] = pkgName
					p.PackagePaths[name] = pkgImportPath
					p.SourceFiles[name] = path
					p.TypeToDecl[name] = genDecl
					p.TypeNames = appendIfMissing(p.TypeNames, name)

					if fileNamespace != "" {
						p.TypeNamespaces[name] = fileNamespace
					}
					continue
				} // Check if it's a potential enum (type with @gqlEnum directive)
				if hasGqlEnumDirective(genDecl) {
					baseType := getBaseTypeName(t.Type)
					if baseType == "string" || baseType == "int" {
						// Store enum candidate for later matching
						p.enumCandidates[t.Name.Name] = &enumCandidate{
							TypeSpec:   t,
							GenDecl:    genDecl,
							BaseType:   baseType,
							PkgName:    pkgName,
							FilePath:   path,
							ImportPath: pkgImportPath, // Store import path too
							Namespace:  fileNamespace,
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
	TypeSpec   *ast.TypeSpec
	GenDecl    *ast.GenDecl
	BaseType   string // "string" or "int"
	PkgName    string
	FilePath   string
	ImportPath string // Package import path
	Namespace  string // File-level namespace
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
		// Normalize single-line comments
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)
		// Normalize block comments
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimPrefix(text, "/**")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)

			// Look for @gqlNamespace or @GqlNamespace(name:"value")
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "@gqlnamespace") {
				if idx := strings.Index(lowerLine, "@gqlnamespace"); idx != -1 {
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

// GetPackageImportPath returns the full import path for a type using go/packages
func (p *Parser) GetPackageImportPath(typeName string, modelPath string) string {
	pkgName := p.PackageNames[typeName]

	pkgPath, ok := p.PackagePaths[typeName]
	if !ok {
		// No package path info, use modelPath if provided (for local types only), otherwise package name
		if modelPath != "" && !p.ExternalTypes[typeName] {
			return modelPath
		}
		return pkgName
	}

	// If modelPath is provided and this is NOT an external type, use it instead of the detected package path
	// External types (loaded on-demand with @GqlInclude) should always use their real import path
	if modelPath != "" && !p.ExternalTypes[typeName] {
		return modelPath
	}

	// If we have the package path, return it directly
	if pkgPath != "" {
		return pkgPath
	}

	// Fallback: try to load from source file
	filePath, hasFile := p.SourceFiles[typeName]
	if !hasFile {
		if modelPath != "" {
			return modelPath
		}
		return pkgName
	}

	// Get the directory containing the file
	dir := filepath.Dir(filePath)

	// Check cache first
	if pkg, cached := p.pkgCache[dir]; cached {
		if pkg != nil && pkg.PkgPath != "" {
			return pkg.PkgPath
		}
	}

	// Load package info using go/packages
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil || len(pkgs) == 0 || pkgs[0].PkgPath == "" {
		// Fallback to modelPath if package loading fails, or package name if no modelPath
		p.pkgCache[dir] = nil // cache the failure
		if modelPath != "" {
			return modelPath
		}
		return pkgName
	}

	pkg := pkgs[0]
	p.pkgCache[dir] = pkg
	return pkg.PkgPath
}

// GetPackageImportPathFromFile builds the import path from a file path using go/packages
func (p *Parser) GetPackageImportPathFromFile(filePath string, pkgName string, modelPath string) string {
	// Get the directory containing the file
	dir := filepath.Dir(filePath)

	// Check cache first
	if pkg, cached := p.pkgCache[dir]; cached {
		if pkg != nil && pkg.PkgPath != "" {
			return pkg.PkgPath
		}
	}

	// Load package info using go/packages
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil || len(pkgs) == 0 || pkgs[0].PkgPath == "" {
		// Fallback to modelPath if package loading fails, or package name if no modelPath
		p.pkgCache[dir] = nil // cache the failure
		if modelPath != "" {
			return modelPath
		}
		return pkgName
	}

	pkg := pkgs[0]
	p.pkgCache[dir] = pkg
	return pkg.PkgPath
}

// hasGqlEnumDirective checks if a GenDecl has @gqlEnum or @GqlEnum directive in its doc comments
func hasGqlEnumDirective(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}
	for _, comment := range decl.Doc.List {
		text := strings.ToLower(comment.Text)
		if strings.Contains(text, "@gqlenum") {
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
		p.PackagePaths[enumTypeName] = candidate.ImportPath  // Use import path instead of file path
		p.EnumSourceFiles[enumTypeName] = candidate.FilePath // Store source file path

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

// parseValueDirective extracts @gqlEnumValue or @GqlEnumValue directive from comment
func (p *Parser) parseValueDirective(commentGroup *ast.CommentGroup, goName string, enumTypeName string) (graphQLName, description, deprecated string) {
	// Default: auto-generate GraphQL name by stripping enum type prefix
	graphQLName = stripEnumPrefix(goName, enumTypeName)

	if commentGroup == nil {
		return
	}

	hasDirective := false

	for _, comment := range commentGroup.List {
		text := comment.Text

		// Extract @gqlEnumValue or @GqlEnumValue directive
		lowerText := strings.ToLower(text)
		if strings.Contains(lowerText, "@gqlenumvalue") {
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

// parseEnumDirective extracts custom name, description, and namespace from @gqlEnum or @GqlEnum directive
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
		// Normalize single-line comments
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)
		// Normalize block comments
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimPrefix(text, "/**")
		text = strings.TrimSuffix(text, "*/")

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "//")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)

			// Check for @gqlEnum or @GqlEnum directive
			lowerLine := strings.ToLower(line)
			if strings.HasPrefix(lowerLine, "@gqlenum") {
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

// HasGQLAnnotations checks if a Go type (from field expression) has @gqlType or @gqlInput annotations
// This loads and parses the type on-demand if it's not already scanned
// parentTypeName is the name of the type that contains the field (used to find the source file for import resolution)
func (p *Parser) HasGQLAnnotations(fieldExpr ast.Expr, parentTypeName string) bool {
	// Extract package selector and type name from the field expression
	var pkgPath string
	var typeIdent *ast.Ident

	// Unwrap pointers, arrays, slices
	expr := fieldExpr
	for {
		switch t := expr.(type) {
		case *ast.StarExpr:
			expr = t.X
		case *ast.ArrayType:
			expr = t.Elt
		case *ast.SelectorExpr:
			// pkg.TypeName
			if ident, ok := t.X.(*ast.Ident); ok {
				// Get the package path from the source file of the parent type
				if parentFilePath, exists := p.SourceFiles[parentTypeName]; exists {
					// Parse the file to get imports
					fset := token.NewFileSet()
					f, err := parser.ParseFile(fset, parentFilePath, nil, parser.ParseComments)
					if err == nil {
						pkgAlias := ident.Name
						// Find the import path for this package alias
						for _, imp := range f.Imports {
							impPath := strings.Trim(imp.Path.Value, `"`)
							impName := filepath.Base(impPath)
							if imp.Name != nil {
								impName = imp.Name.Name
							}
							if impName == pkgAlias {
								pkgPath = impPath
								break
							}
						}
					}
				}
				typeIdent = t.Sel
			}
			goto done
		case *ast.Ident:
			// Just TypeName (same package)
			typeIdent = t
			goto done
		case *ast.IndexExpr, *ast.IndexListExpr:
			// Generic instantiation - extract base type
			if idx, ok := t.(*ast.IndexExpr); ok {
				expr = idx.X
			} else if idxList, ok := t.(*ast.IndexListExpr); ok {
				expr = idxList.X
			}
		default:
			return false
		}
	}
done:

	if typeIdent == nil {
		return false
	}

	typeName := typeIdent.Name

	// If no package path (same package or already scanned), check ScannedTypes
	if pkgPath == "" {
		if info, exists := p.ScannedTypes[typeName]; exists {
			return info.HasTypeDirective || info.HasInputDirective
		}
		return false
	}

	// Check if we already know about this type
	qualifiedName := pkgPath + "." + typeName
	if info, exists := p.ScannedTypes[qualifiedName]; exists {
		return info.HasTypeDirective || info.HasInputDirective
	}
	if info, exists := p.ScannedTypes[typeName]; exists {
		return info.HasTypeDirective || info.HasInputDirective
	}

	// Load the package on-demand to check for annotations
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil || len(pkgs) == 0 {
		return false
	}

	pkg := pkgs[0]

	// When loading a package on-demand, add ALL struct types from it to the parser
	// This ensures that non-annotated types (like Connection[T]) are available for embedded field expansion
	var foundRequestedType bool
	var requestedTypeHasAnnotations bool

	for _, file := range pkg.Syntax {
		// First pass: process type declarations
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				currentTypeName := typeSpec.Name.Name

				// Check if this is a struct type or an enum type
				_, isStruct := typeSpec.Type.(*ast.StructType)
				isEnum := hasGqlEnumDirective(genDecl)

				if !isStruct && !isEnum {
					continue
				}

				// Handle enum types
				if isEnum {
					baseType := getBaseTypeName(typeSpec.Type)
					if baseType == "string" || baseType == "int" {
						// Add to enum candidates
						p.enumCandidates[currentTypeName] = &enumCandidate{
							TypeSpec:   typeSpec,
							GenDecl:    genDecl,
							BaseType:   baseType,
							PkgName:    pkg.Name,
							FilePath:   pkg.Fset.File(file.Pos()).Name(),
							ImportPath: pkg.PkgPath,
							Namespace:  "", // Could extract from file-level directives if needed
						}

						// Track if this is the requested type
						if currentTypeName == typeName {
							foundRequestedType = true
							requestedTypeHasAnnotations = true
						}
					}
					continue
				}

				// Parse directives for struct types
				directives := ParseDirectives(typeSpec, genDecl)

				// Add to StructTypes (always, even if not annotated - needed for embedded expansion)
				p.StructTypes[currentTypeName] = typeSpec
				p.TypeToDecl[currentTypeName] = genDecl
				p.PackageNames[currentTypeName] = pkg.Name
				p.PackagePaths[currentTypeName] = pkg.PkgPath

				// Find source file
				for _, f := range pkg.Syntax {
					for _, d := range f.Decls {
						if gd, ok := d.(*ast.GenDecl); ok && gd == genDecl {
							p.SourceFiles[currentTypeName] = pkg.Fset.File(f.Pos()).Name()
							break
						}
					}
				}

				// Store type parameters if it's a generic type
				if typeSpec.TypeParams != nil && typeSpec.TypeParams.NumFields() > 0 {
					params := make([]string, 0, typeSpec.TypeParams.NumFields())
					for _, field := range typeSpec.TypeParams.List {
						for _, name := range field.Names {
							params = append(params, name.Name)
						}
					}
					p.TypeParameters[currentTypeName] = params
				}

				// If this type has annotations, add to TypeNames (for generation) and ScannedTypes
				if directives.HasTypeDirective || directives.HasInputDirective || directives.HasIncludeDirective {
					// Add to TypeNames for generation (if not already there)
					found := false
					for _, name := range p.TypeNames {
						if name == currentTypeName {
							found = true
							break
						}
					}
					if !found {
						p.TypeNames = append(p.TypeNames, currentTypeName)
					}

					// Mark as external type (loaded on-demand from external package)
					p.ExternalTypes[currentTypeName] = true

					// Cache in ScannedTypes
					info := &ScannedTypeInfo{
						TypeName:            currentTypeName,
						HasTypeDirective:    directives.HasTypeDirective,
						HasInputDirective:   directives.HasInputDirective,
						HasIncludeDirective: directives.HasIncludeDirective,
						IsExternal:          true, // This type was loaded on-demand
						GeneratedTypes:      make([]string, 0),
						GeneratedInputs:     make([]string, 0),
					}

					for _, typeDef := range directives.Types {
						if typeDef.Name != "" {
							info.GeneratedTypes = append(info.GeneratedTypes, typeDef.Name)
						} else {
							info.GeneratedTypes = append(info.GeneratedTypes, currentTypeName)
						}
					}

					for _, inputDef := range directives.Inputs {
						if inputDef.Name != "" {
							info.GeneratedInputs = append(info.GeneratedInputs, inputDef.Name)
						} else {
							info.GeneratedInputs = append(info.GeneratedInputs, currentTypeName+"Input")
						}
					}

					p.ScannedTypes[currentTypeName] = info
				}

				// Track if this is the requested type and if it has annotations
				if currentTypeName == typeName {
					foundRequestedType = true
					requestedTypeHasAnnotations = directives.HasTypeDirective || directives.HasInputDirective || directives.HasIncludeDirective
				}
			}
		}

		// Second pass: collect const blocks for enum matching
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}

			// Store const block info for later matching
			constBlock := &constBlockInfo{
				GenDecl:  genDecl,
				FilePath: pkg.Fset.File(file.Pos()).Name(),
				PkgName:  pkg.Name,
			}
			p.constBlocks = append(p.constBlocks, constBlock)

			// Immediately match with enum candidates (in case this is called during generation)
			p.parseConstBlock(constBlock, p.enumCandidates)
		}
	}

	return foundRequestedType && requestedTypeHasAnnotations
}
