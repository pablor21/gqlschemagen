package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// Parser collects type specs and related AST nodes across a root dir
type Parser struct {
	StructTypes  map[string]*ast.TypeSpec
	Structs      map[string]*ast.StructType
	PackageNames map[string]string
	TypeToDecl   map[string]*ast.GenDecl
	// ordered list of type names for deterministic output
	TypeNames []string
}

func NewParser() *Parser {
	return &Parser{
		StructTypes:  make(map[string]*ast.TypeSpec),
		Structs:      make(map[string]*ast.StructType),
		PackageNames: make(map[string]string),
		TypeToDecl:   make(map[string]*ast.GenDecl),
	}
}

func (p *Parser) Walk(root string) error {
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
