// Package generator provides automatic GraphQL schema generation from Go structs.
//
// This package scans Go source code for struct definitions and generates GraphQL
// type definitions, input types, and schema files based on struct tags and special
// directives. It supports field name transformations, custom type mappings, and
// automatic input generation for mutations.
//
// # Features
//
//   - Automatic GraphQL type generation from Go structs
//   - Field name transformations (camelCase, snake_case, PascalCase)
//   - Support for gql tags and directives (@gqlgen, @gqlField, etc.)
//   - Automatic input type generation from structs
//   - Custom prefix/suffix stripping from type names
//   - Support for embedded structs and referenced types
//   - Configurable output strategies (single file or per-struct)
//
// # Basic Usage
//
//	parser := generator.NewParser()
//	err := parser.ParsePackages([]string{"./models"})
//	if err != nil {
//		panic(err)
//	}
//
//	config := generator.NewConfig()
//	config.Output = "schema.graphql"
//
//	gen := generator.NewGenerator(parser, config)
//	err = gen.Run()
//	if err != nil {
//		panic(err)
//	}
//
// For detailed documentation and examples, see:
// https://github.com/pablor21/gqlschemagen
package generator

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"sort"
	"strings"
)

type Generator struct {
	P      *Parser
	Config *Config
}

func NewGenerator(p *Parser, config *Config) *Generator {
	if config == nil {
		config = NewConfig()
	}
	config.Normalize()
	return &Generator{P: p, Config: config}
}

func (g *Generator) Run() error {
	// Ensure output directory exists
	outputDir := g.Config.Output
	if g.Config.GenStrategy == GenStrategySingle {
		outputDir = filepath.Dir(g.Config.Output)
	}
	if err := EnsureDir(outputDir); err != nil {
		return err
	}

	// Build topological order
	orders := g.buildDependencyOrder()

	// Generate based on strategy
	if g.Config.GenStrategy == GenStrategySingle {
		return g.generateSingleFile(orders)
	}
	return g.generateMultipleFiles(orders)
}

func (g *Generator) buildDependencyOrder() []string {
	names := make([]string, 0, len(g.P.TypeNames))
	for _, n := range g.P.TypeNames {
		names = append(names, n)
	}
	sort.Strings(names)

	// Topological sort
	orders := []string{}
	visited := map[string]bool{}
	var dfs func(string)
	dfs = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		st := g.P.Structs[n]
		if st == nil {
			return
		}
		for _, f := range st.Fields.List {
			ft := FieldTypeName(f.Type)
			if _, ok := g.P.Structs[ft]; ok {
				dfs(ft)
			}
		}
		orders = append(orders, n)
	}
	for _, n := range names {
		dfs(n)
	}
	return orders
}

func (g *Generator) generateSingleFile(orders []string) error {
	outFile := g.Config.Output
	if outFile == "" {
		outFile = filepath.Join(g.Config.OutDir, "schema.graphql")
	}

	if g.Config.SkipExisting && FileExists(outFile) {
		fmt.Println("skip", outFile)
		return nil
	}

	buf := strings.Builder{}
	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil || g.P.TypeToDecl[typeName] == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Generate type only if @gqlType directive is present
		if d.HasTypeDirective {
			typeContent := g.generateType(typeSpec, g.P.Structs[typeName], d)
			if typeContent != "" {
				buf.WriteString(typeContent)
			}
		}

		// Generate input only if @gqlInput directive is present
		if d.HasInputDirective {
			inputContent := g.generateInput(typeSpec, g.P.Structs[typeName], d)
			if inputContent != "" {
				buf.WriteString(inputContent)
			}
		}
	}

	return WriteFile(outFile, buf.String())
}

func (g *Generator) generateMultipleFiles(orders []string) error {
	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Skip if no directives present (opt-in generation)
		if !d.HasTypeDirective && !d.HasInputDirective {
			continue
		}

		// Generate filename
		fileName := g.resolveFileName(d, typeSpec.Name.Name)
		outFile := filepath.Join(g.Config.Output, fileName)

		if g.Config.SkipExisting && FileExists(outFile) {
			fmt.Println("skip", outFile)
			continue
		}

		if err := EnsureDir(filepath.Dir(outFile)); err != nil {
			return err
		}

		buf := strings.Builder{}

		// Generate type only if @gqlType directive is present
		if d.HasTypeDirective {
			typeContent := g.generateType(typeSpec, g.P.Structs[typeName], d)
			if typeContent != "" {
				buf.WriteString(typeContent)
			}
		}

		// Generate input only if @gqlInput directive is present
		if d.HasInputDirective {
			inputContent := g.generateInput(typeSpec, g.P.Structs[typeName], d)
			if inputContent != "" {
				buf.WriteString(inputContent)
			}
		}

		if buf.Len() > 0 {
			if err := WriteFile(outFile, buf.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Generator) resolveFileName(d StructDirectives, typeName string) string {
	pattern := g.Config.SchemaFileName
	if pattern == "" {
		pattern = "{model_name}.graphqls"
	}

	modelName := strings.ToLower(d.GQLName)
	pattern = strings.ReplaceAll(pattern, "{model_name}", modelName)
	pattern = strings.ReplaceAll(pattern, "{type_name}", typeName)

	return pattern
}

func (g *Generator) generateType(typeSpec *ast.TypeSpec, st *ast.StructType, d StructDirectives) string {
	name := d.GQLName
	if d.GQLType != "" {
		// Use custom type name from @gqlType annotation
		name = d.GQLType
	} else {
		// Apply prefix/suffix stripping only when no custom name is specified
		name = StripPrefixSuffix(name, g.Config.StripPrefix, g.Config.StripSuffix)
		// Apply prefix/suffix addition
		if g.Config.AddTypePrefix != "" {
			name = g.Config.AddTypePrefix + name
		}
		if g.Config.AddTypeSuffix != "" {
			name = name + g.Config.AddTypeSuffix
		}
	}

	buf := strings.Builder{}

	// Add description if present
	if d.TypeDescription != "" {
		buf.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", d.TypeDescription))
	}

	// Type declaration
	buf.WriteString(fmt.Sprintf("type %s", name))

	// Add @goModel directive if enabled
	if g.Config.UseGqlGenDirectives || d.UseModelDirective {
		pkgName := g.P.PackageNames[typeSpec.Name.Name]
		if g.Config.ModelPath != "" {
			pkgName = g.Config.ModelPath
		}
		buf.WriteString(fmt.Sprintf(" @goModel(model: \"%s.%s\")", pkgName, typeSpec.Name.Name))
	}

	buf.WriteString(" {\n")

	// Generate fields from struct
	fields := g.generateFields(st, d, false) // false = for type, not input
	if len(fields) == 0 && len(d.ExtraFields) == 0 && !g.Config.IncludeEmptyTypes {
		return "" // Skip empty types
	}

	buf.WriteString(fields)

	// Add extra fields from @gqlExtraField annotations
	for _, ef := range d.ExtraFields {
		if ef.Description != "" {
			buf.WriteString(fmt.Sprintf("\t\"\"\"%s\"\"\"\n", ef.Description))
		}
		buf.WriteString(fmt.Sprintf("\t%s: %s", ef.Name, ef.Type))
		if g.Config.UseGqlGenDirectives {
			buf.WriteString(" @goField(forceResolver: true)")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("}\n\n")

	return buf.String()
}

func (g *Generator) generateInput(typeSpec *ast.TypeSpec, st *ast.StructType, d StructDirectives) string {
	inputName := d.GQLInput
	if inputName == "" {
		// Apply prefix/suffix stripping before adding "Input" suffix
		baseName := StripPrefixSuffix(d.GQLName, g.Config.StripPrefix, g.Config.StripSuffix)
		inputName = baseName + "Input"
		// Apply prefix/suffix addition
		if g.Config.AddInputPrefix != "" {
			inputName = g.Config.AddInputPrefix + inputName
		}
		if g.Config.AddInputSuffix != "" {
			inputName = inputName + g.Config.AddInputSuffix
		}
	} else {
		// Custom name provided, don't add prefix/suffix
	}

	buf := strings.Builder{}

	// Add description if present
	if d.InputDescription != "" {
		buf.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", d.InputDescription))
	}

	// Input declaration
	buf.WriteString(fmt.Sprintf("input %s", inputName))

	// Add @goModel directive if enabled
	if g.Config.UseGqlGenDirectives || d.UseModelDirective {
		pkgName := g.P.PackageNames[typeSpec.Name.Name]
		if g.Config.ModelPath != "" {
			pkgName = g.Config.ModelPath
		}
		buf.WriteString(fmt.Sprintf(" @goModel(model: \"%s.%s\")", pkgName, typeSpec.Name.Name))
	}

	buf.WriteString(" {\n")

	// Generate fields from struct
	fields := g.generateFields(st, d, true) // true = for input
	if len(fields) == 0 && len(d.ExtraFields) == 0 && !g.Config.IncludeEmptyTypes {
		return "" // Skip empty inputs
	}

	buf.WriteString(fields)

	// Add extra fields from @gqlExtraField annotations
	for _, ef := range d.ExtraFields {
		if ef.Description != "" {
			buf.WriteString(fmt.Sprintf("\t\"\"\"%s\"\"\"\n", ef.Description))
		}
		buf.WriteString(fmt.Sprintf("\t%s: %s\n", ef.Name, ef.Type))
	}

	buf.WriteString("}\n\n")

	return buf.String()
}

func (g *Generator) generateFields(st *ast.StructType, d StructDirectives, forInput bool) string {
	buf := strings.Builder{}

	// Determine which ignoreAll flag to use
	ignoreAll := d.IgnoreAll
	if forInput && d.InputIgnoreAll {
		ignoreAll = true
	} else if !forInput && d.TypeIgnoreAll {
		ignoreAll = true
	}

	for _, f := range st.Fields.List {
		if f.Names == nil {
			continue // Skip embedded fields
		}

		opt := ParseFieldOptions(f, g.Config)

		// Determine if field should be included
		include := (!ignoreAll && !opt.Ignore && !opt.Omit) || opt.Include
		if !include {
			continue
		}

		// Resolve field name
		fieldName := opt.Name
		if fieldName == "" {
			fieldName = ResolveFieldName(f, g.Config)
		}

		// Resolve field type
		fieldType := opt.Type
		if fieldType == "" {
			fieldType = ExprToGraphQLType(f.Type)
		}

		// Handle optional/required
		if opt.Optional {
			fieldType = strings.TrimSuffix(fieldType, "!")
		} else if opt.Required && !strings.HasSuffix(fieldType, "!") {
			fieldType = fieldType + "!"
		}

		// Add field with description if present
		if opt.Description != "" {
			buf.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n", opt.Description))
		}

		buf.WriteString(fmt.Sprintf("    %s: %s", fieldName, fieldType))

		// Add @goField directive if forceResolver is set
		if g.Config.UseGqlGenDirectives && opt.ForceResolver {
			buf.WriteString(" @goField(forceResolver: true)")
		}

		buf.WriteString("\n")
	}

	return buf.String()
}
