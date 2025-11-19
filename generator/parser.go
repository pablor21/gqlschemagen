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
	// Check if root contains ** pattern (recursive glob)
	if strings.Contains(root, "**") {
		// Handle recursive glob patterns manually
		return p.walkGlobPattern(root)
	}

	// Check if root contains other glob patterns
	if strings.ContainsAny(root, "*?[{") {
		// Use filepath.Glob for simple patterns
		matches, err := filepath.Glob(root)
		if err != nil {
			return fmt.Errorf("glob pattern error for %s: %w", root, err)
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}

			if info.IsDir() {
				// If matched path is a directory, walk it
				if err := p.walkDir(match); err != nil {
					return err
				}
			} else if strings.HasSuffix(match, ".go") && !strings.HasSuffix(match, "_test.go") {
				// If matched path is a Go file, parse it
				if err := p.parseFile(match); err != nil {
					return fmt.Errorf("parse %s: %w", match, err)
				}
			}
		}
		return nil
	}

	// No glob pattern, use regular directory walk
	return p.walkDir(root)
}

// walkGlobPattern handles ** recursive glob patterns
func (p *Parser) walkGlobPattern(pattern string) error {
	// Split pattern into base path and match pattern
	// e.g., "examples/**/*.go" -> base: "examples", pattern: "*.go"
	// e.g., "examples/**/models" -> base: "examples", pattern: "models"
	// e.g., "examples/**/models/*.go" -> base: "examples", pattern: "models/*.go"
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return fmt.Errorf("invalid glob pattern: %s (only one ** allowed)", pattern)
	}

	basePath := strings.TrimSuffix(parts[0], "/")
	basePath = strings.TrimSuffix(basePath, string(filepath.Separator))
	if basePath == "" {
		basePath = "."
	}

	matchPattern := strings.TrimPrefix(parts[1], "/")
	matchPattern = strings.TrimPrefix(matchPattern, string(filepath.Separator))

	// Check if basePath exists
	if _, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, return without error (no matches)
			return nil
		}
		return err
	}

	// Walk the directory tree starting from basePath
	return filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories that can't be accessed
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from base
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		// Normalize path separators for matching
		relPath = filepath.ToSlash(relPath)
		normalizedPattern := filepath.ToSlash(matchPattern)

		// If pattern is just a directory name (no wildcards or file extension)
		if !strings.ContainsAny(matchPattern, "*?") && !strings.Contains(matchPattern, ".") {
			// Pattern is a directory name like "models"
			if d.IsDir() && filepath.Base(path) == matchPattern {
				// Found matching directory, walk it
				return p.walkDir(path)
			}
			return nil
		}

		// For file patterns
		if d.IsDir() {
			return nil
		}

		// Try to match the pattern
		var matched bool

		// If pattern contains path separators (e.g., "entities/*.go")
		if strings.Contains(normalizedPattern, "/") {
			// Try matching against full relative path
			matched, err = filepath.Match(normalizedPattern, relPath)
			if err != nil {
				return err
			}

			// If not matched, try matching against suffix parts
			// For pattern "entities/*.go", check if any suffix of the path matches
			if !matched {
				pathParts := strings.Split(relPath, "/")
				patternParts := strings.Split(normalizedPattern, "/")

				// Try matching from different starting points in the path
				for i := 0; i <= len(pathParts)-len(patternParts); i++ {
					suffix := strings.Join(pathParts[i:], "/")
					matched, err = filepath.Match(normalizedPattern, suffix)
					if err != nil {
						return err
					}
					if matched {
						break
					}
				}
			}
		} else {
			// Simple file pattern, match against basename
			matched, err = filepath.Match(normalizedPattern, filepath.Base(path))
			if err != nil {
				return err
			}
		}

		if matched && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			if err := p.parseFile(path); err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
		}

		return nil
	})
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
