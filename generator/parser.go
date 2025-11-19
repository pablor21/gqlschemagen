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
// If modelPath is provided, it constructs the path relative to it
func (p *Parser) GetPackageImportPath(typeName string, modelPath string) string {
	if modelPath == "" {
		// Just return package name if no model path configured
		return p.PackageNames[typeName]
	}

	filePath, ok := p.PackagePaths[typeName]
	if !ok {
		return p.PackageNames[typeName]
	}

	// Get the directory of the file
	dir := filepath.ToSlash(filepath.Dir(filePath))

	// The file path is absolute or relative from where the tool was run
	// We need to extract the package path from the directory structure

	// Strategy: Look for the package directory and its parent directories
	// Start from the file's directory and work backwards to build the import path
	parts := strings.Split(dir, "/")

	// Find the index where the package name appears
	pkgName := p.PackageNames[typeName]
	pkgIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == pkgName {
			pkgIndex = i
			break
		}
	}

	if pkgIndex == -1 {
		// Fallback: just append package name
		return modelPath + "/" + pkgName
	}

	// Build the import path from the package directory and any parent directories
	// that are part of the module structure (not going beyond common paths like src, internal, pkg, etc.)
	var importParts []string

	// Work backwards from pkgIndex to collect relevant path components
	for i := pkgIndex; i >= 0; i-- {
		part := parts[i]

		// Skip common irrelevant directories
		if part == "." || part == ".." || part == "" {
			continue
		}

		// Stop at common workspace roots (but include them if they're meaningful)
		// Include: internal, pkg, cmd, api, etc.
		// Stop at: src (unless it's meaningful), workspace root indicators

		importParts = append([]string{part}, importParts...)

		// Stop if we hit a likely module root indicator
		// But continue collecting if we see internal, pkg, cmd, api, etc.
		if i < len(parts)-1 {
			// Check if this could be a module structure path
			if part != "internal" && part != "pkg" && part != "cmd" && part != "api" &&
				part != "models" && part != "entities" && part != "domain" {
				// Might be at module root, but keep the current directory
				break
			}
		}
	}

	// Join with model path
	if len(importParts) > 0 {
		return modelPath + "/" + strings.Join(importParts, "/")
	}

	return modelPath + "/" + pkgName
}
