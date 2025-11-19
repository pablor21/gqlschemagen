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

// Parser collects type specs and related AST nodes across a root dir
type Parser struct {
	StructTypes  map[string]*ast.TypeSpec
	Structs      map[string]*ast.StructType
	PackageNames map[string]string
	PackagePaths map[string]string // Full import path for each type
	TypeToDecl   map[string]*ast.GenDecl
	// ordered list of type names for deterministic output
	TypeNames []string
}

func NewParser() *Parser {
	return &Parser{
		StructTypes:  make(map[string]*ast.TypeSpec),
		Structs:      make(map[string]*ast.StructType),
		PackageNames: make(map[string]string),
		PackagePaths: make(map[string]string),
		TypeToDecl:   make(map[string]*ast.GenDecl),
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

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			t, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			s, ok := t.Type.(*ast.StructType)
			if !ok {
				continue
			}
			name := t.Name.Name
			p.StructTypes[name] = t
			p.Structs[name] = s
			p.PackageNames[name] = pkgName
			// Store the file path to derive package import path later
			p.PackagePaths[name] = path
			p.TypeToDecl[name] = genDecl
			p.TypeNames = appendIfMissing(p.TypeNames, name)
		}
	}
	return nil
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
