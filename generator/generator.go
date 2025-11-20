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
//	config.Output = "schema.graphqls"
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
	// Check if we have any namespaces defined
	hasNamespaces := len(g.P.TypeNamespaces) > 0 || len(g.P.EnumNamespaces) > 0

	// Ensure output directory exists
	outputDir := g.Config.Output
	if g.Config.GenStrategy == GenStrategySingle && !hasNamespaces {
		// For single strategy without namespaces, check if Output is a file path or directory
		// If it ends with an extension, it's a file path - extract the directory
		if strings.HasSuffix(g.Config.Output, ".graphqls") || strings.HasSuffix(g.Config.Output, ".graphql") || strings.HasSuffix(g.Config.Output, ".gql") {
			outputDir = filepath.Dir(g.Config.Output)
		} else {
			outputDir = g.Config.Output
		}
	} else if g.Config.GenStrategy == GenStrategySingle && hasNamespaces {
		// When using namespaces with single strategy, output path should be treated as directory
		outputDir = g.Config.Output
	}
	if err := EnsureDir(outputDir); err != nil {
		return err
	}

	// Build topological order
	orders := g.buildDependencyOrder()

	// If namespaces are defined AND we're using package strategy, merge both approaches
	// If namespaces are defined with single/multiple strategy, use namespace generation
	// Otherwise, use the configured strategy
	if hasNamespaces && g.Config.GenStrategy == GenStrategyPackage {
		return g.generateByNamespaceAndPackage(orders)
	} else if hasNamespaces {
		return g.generateByNamespace(orders)
	}

	// Generate based on strategy
	switch g.Config.GenStrategy {
	case GenStrategySingle:
		return g.generateSingleFile(orders)
	case GenStrategyPackage:
		return g.generatePackageFiles(orders)
	default: // GenStrategyMultiple
		return g.generateMultipleFiles(orders)
	}
}

func (g *Generator) buildDependencyOrder() []string {
	names := make([]string, 0, len(g.P.TypeNames))
	// for _, n := range g.P.TypeNames {
	// 	names = append(names, n)
	// }
	names = append(names, g.P.TypeNames...)
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

// generateByNamespace generates schema files organized by namespace
func (g *Generator) generateByNamespace(orders []string) error {
	// Group types, inputs, and enums by namespace
	type namespaceItems struct {
		types  map[string][]TypeDefinition  // typeName -> type definitions
		inputs map[string][]InputDefinition // typeName -> input definitions
		enums  []string                     // enum names
	}

	namespaces := make(map[string]*namespaceItems)

	// Helper to get or create namespace group
	getNamespace := func(ns string) *namespaceItems {
		if ns == "" {
			ns = "_default"
		}
		if namespaces[ns] == nil {
			namespaces[ns] = &namespaceItems{
				types:  make(map[string][]TypeDefinition),
				inputs: make(map[string][]InputDefinition),
				enums:  []string{},
			}
		}
		return namespaces[ns]
	}

	// Group types and inputs by namespace (file-level or directive-level)
	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil || g.P.TypeToDecl[typeName] == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Get file-level namespace for this type
		fileNamespace := g.P.TypeNamespaces[typeName]

		// Group types by their namespace (directive-level overrides file-level)
		if d.HasTypeDirective {
			for _, typeDef := range d.Types {
				ns := typeDef.Namespace
				if ns == "" {
					ns = fileNamespace
				}
				nsItems := getNamespace(ns)
				nsItems.types[typeName] = append(nsItems.types[typeName], typeDef)
			}
		}

		// Group inputs by their namespace (directive-level overrides file-level)
		if d.HasInputDirective {
			for _, inputDef := range d.Inputs {
				ns := inputDef.Namespace
				if ns == "" {
					ns = fileNamespace
				}
				nsItems := getNamespace(ns)
				nsItems.inputs[typeName] = append(nsItems.inputs[typeName], inputDef)
			}
		}
	}

	// Group enums by namespace
	for _, enumName := range g.P.EnumNames {
		ns := g.P.EnumNamespaces[enumName]
		nsItems := getNamespace(ns)
		nsItems.enums = append(nsItems.enums, enumName)
	}

	// Collect all content in memory first (map of file path -> content)
	fileContents := make(map[string]*strings.Builder)

	// Generate content for each namespace
	for namespace, items := range namespaces {
		var outFile string

		if namespace == "_default" {
			// Types without namespace go to default location
			outFile = filepath.Join(g.Config.Output, g.Config.OutputFileName)
		} else {
			// Convert namespace to file path using configured separator
			// e.g., "user/auth" with separator "/" becomes "user/auth.graphqls"
			namespacePath := namespace
			if g.Config.NamespaceSeparator != "/" {
				namespacePath = strings.ReplaceAll(namespace, g.Config.NamespaceSeparator, string(filepath.Separator))
			}
			outFile = filepath.Join(g.Config.Output, namespacePath+g.Config.OutputFileExtension)
		}

		if g.Config.SkipExisting && FileExists(outFile) {
			fmt.Println("skip", outFile)
			continue
		}

		// Get or create buffer for this file
		buf, exists := fileContents[outFile]
		if !exists {
			buf = &strings.Builder{}
			fileContents[outFile] = buf
		}

		// Generate enums for this namespace
		for _, enumName := range items.enums {
			enumType := g.P.EnumTypes[enumName]
			if enumType == nil {
				continue
			}
			enumContent := g.generateEnum(enumType)
			if enumContent != "" {
				buf.WriteString(enumContent)
				buf.WriteString("\n")
			}
		}

		// Generate types for this namespace
		for typeName, typeDefs := range items.types {
			typeSpec := g.P.StructTypes[typeName]
			structType := g.P.Structs[typeName]
			d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])

			for _, typeDef := range typeDefs {
				typeContent := g.generateTypeFromDef(typeSpec, structType, d, typeDef)
				if typeContent != "" {
					buf.WriteString(typeContent)
				}
			}
		}

		// Generate inputs for this namespace
		for typeName, inputDefs := range items.inputs {
			typeSpec := g.P.StructTypes[typeName]
			structType := g.P.Structs[typeName]
			d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])

			for _, inputDef := range inputDefs {
				inputContent := g.generateInputFromDef(typeSpec, structType, d, inputDef)
				if inputContent != "" {
					buf.WriteString(inputContent)
				}
			}
		}
	}

	// Write all files at once
	for outFile, buf := range fileContents {
		if buf.Len() > 0 {
			// Ensure directory exists
			if err := EnsureDir(filepath.Dir(outFile)); err != nil {
				return err
			}
			if err := WriteFile(outFile, buf.String(), g.Config); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateByNamespaceAndPackage combines namespace and package strategies
// Types with namespaces use namespace, types without use package directory
func (g *Generator) generateByNamespaceAndPackage(orders []string) error {
	// Collect all content in memory first (map of file path -> content)
	fileContents := make(map[string]*strings.Builder)

	// Helper to get or create buffer for a file
	getBuffer := func(filePath string) *strings.Builder {
		buf, exists := fileContents[filePath]
		if !exists {
			buf = &strings.Builder{}
			fileContents[filePath] = buf
		}
		return buf
	}

	// Process enums
	for _, enumName := range g.P.EnumNames {
		enumType := g.P.EnumTypes[enumName]
		if enumType == nil {
			continue
		}

		var outFile string
		ns := g.P.EnumNamespaces[enumName]

		if ns != "" {
			// Use namespace
			namespacePath := ns
			if g.Config.NamespaceSeparator != "/" {
				namespacePath = strings.ReplaceAll(ns, g.Config.NamespaceSeparator, string(filepath.Separator))
			}
			outFile = filepath.Join(g.Config.Output, namespacePath+g.Config.OutputFileExtension)
		} else {
			// Use package directory
			filePath := g.P.PackagePaths[enumName]
			pkgDir := filepath.Dir(filePath)
			pkgName := filepath.Base(pkgDir)
			outFile = filepath.Join(g.Config.Output, pkgName+g.Config.OutputFileExtension)
		}

		if g.Config.SkipExisting && FileExists(outFile) {
			continue
		}

		buf := getBuffer(outFile)
		enumContent := g.generateEnum(enumType)
		if enumContent != "" {
			buf.WriteString(enumContent)
			buf.WriteString("\n")
		}
	}

	// Process types and inputs
	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Skip if no directives present
		if !d.HasTypeDirective && !d.HasInputDirective {
			continue
		}

		// Get file-level namespace for this type
		fileNamespace := g.P.TypeNamespaces[typeName]

		// Process types
		if d.HasTypeDirective {
			for _, typeDef := range d.Types {
				var outFile string
				ns := typeDef.Namespace
				if ns == "" {
					ns = fileNamespace
				}

				if ns != "" {
					// Use namespace
					namespacePath := ns
					if g.Config.NamespaceSeparator != "/" {
						namespacePath = strings.ReplaceAll(ns, g.Config.NamespaceSeparator, string(filepath.Separator))
					}
					outFile = filepath.Join(g.Config.Output, namespacePath+g.Config.OutputFileExtension)
				} else {
					// Use package directory
					filePath := g.P.PackagePaths[typeName]
					pkgDir := filepath.Dir(filePath)
					pkgName := filepath.Base(pkgDir)
					outFile = filepath.Join(g.Config.Output, pkgName+g.Config.OutputFileExtension)
				}

				if g.Config.SkipExisting && FileExists(outFile) {
					continue
				}

				buf := getBuffer(outFile)
				typeContent := g.generateTypeFromDef(typeSpec, g.P.Structs[typeName], d, typeDef)
				if typeContent != "" {
					buf.WriteString(typeContent)
				}
			}
		}

		// Process inputs
		if d.HasInputDirective {
			for _, inputDef := range d.Inputs {
				var outFile string
				ns := inputDef.Namespace
				if ns == "" {
					ns = fileNamespace
				}

				if ns != "" {
					// Use namespace
					namespacePath := ns
					if g.Config.NamespaceSeparator != "/" {
						namespacePath = strings.ReplaceAll(ns, g.Config.NamespaceSeparator, string(filepath.Separator))
					}
					outFile = filepath.Join(g.Config.Output, namespacePath+g.Config.OutputFileExtension)
				} else {
					// Use package directory
					filePath := g.P.PackagePaths[typeName]
					pkgDir := filepath.Dir(filePath)
					pkgName := filepath.Base(pkgDir)
					outFile = filepath.Join(g.Config.Output, pkgName+g.Config.OutputFileExtension)
				}

				if g.Config.SkipExisting && FileExists(outFile) {
					continue
				}

				buf := getBuffer(outFile)
				inputContent := g.generateInputFromDef(typeSpec, g.P.Structs[typeName], d, inputDef)
				if inputContent != "" {
					buf.WriteString(inputContent)
				}
			}
		}
	}

	// Write all files at once
	for outFile, buf := range fileContents {
		if buf.Len() > 0 {
			// Ensure directory exists
			if err := EnsureDir(filepath.Dir(outFile)); err != nil {
				return err
			}
			if err := WriteFile(outFile, buf.String(), g.Config); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) generateSingleFile(orders []string) error {
	// Determine output file path
	// If Output ends with an extension (old style), use it directly
	// If Output is a directory (new style), join with OutputFileName
	var outFile string
	if strings.HasSuffix(g.Config.Output, ".graphqls") || strings.HasSuffix(g.Config.Output, ".graphql") || strings.HasSuffix(g.Config.Output, ".gql") {
		// Old style: Output is the full file path
		outFile = g.Config.Output
	} else {
		// New style: Output is directory, use OutputFileName
		outFile = filepath.Join(g.Config.Output, g.Config.OutputFileName)
	}

	if g.Config.SkipExisting && FileExists(outFile) {
		fmt.Println("skip", outFile)
		return nil
	}

	buf := strings.Builder{}
	// Add code generation notice
	//buf.WriteString("# Code generated by https://github.com/pablor21/gqlschemagen, DO NOT EDIT.\n\n")

	// Generate enums first
	for _, enumName := range g.P.EnumNames {
		enumType := g.P.EnumTypes[enumName]
		if enumType == nil {
			continue
		}
		enumContent := g.generateEnum(enumType)
		if enumContent != "" {
			buf.WriteString(enumContent)
			buf.WriteString("\n")
		}
	}

	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil || g.P.TypeToDecl[typeName] == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Generate all types from @gqlType directives
		if d.HasTypeDirective {
			for _, typeDef := range d.Types {
				typeContent := g.generateTypeFromDef(typeSpec, g.P.Structs[typeName], d, typeDef)
				if typeContent != "" {
					buf.WriteString(typeContent)
				}
			}
		}

		// Generate all inputs from @gqlInput directives
		if d.HasInputDirective {
			for _, inputDef := range d.Inputs {
				inputContent := g.generateInputFromDef(typeSpec, g.P.Structs[typeName], d, inputDef)
				if inputContent != "" {
					buf.WriteString(inputContent)
				}
			}
		}
	}

	return WriteFile(outFile, buf.String(), g.Config)
}

func (g *Generator) generatePackageFiles(orders []string) error {
	// Group types, inputs, and enums by their Go package path
	type packageItems struct {
		types  map[string][]TypeDefinition  // typeName -> type definitions
		inputs map[string][]InputDefinition // typeName -> input definitions
		enums  []string                     // enum names
	}

	packages := make(map[string]*packageItems)

	// Helper to get or create package group
	getPackage := func(pkgPath string) *packageItems {
		if pkgPath == "" {
			pkgPath = "_default"
		}
		if packages[pkgPath] == nil {
			packages[pkgPath] = &packageItems{
				types:  make(map[string][]TypeDefinition),
				inputs: make(map[string][]InputDefinition),
				enums:  []string{},
			}
		}
		return packages[pkgPath]
	}

	// Group enums by package
	for _, enumName := range g.P.EnumNames {
		enumType := g.P.EnumTypes[enumName]
		if enumType == nil {
			continue
		}
		// Get package directory from file path in PackagePaths
		filePath := g.P.PackagePaths[enumName]
		pkgDir := filepath.Dir(filePath)
		pkg := getPackage(pkgDir)
		pkg.enums = append(pkg.enums, enumName)
	}

	// Group types and inputs by package
	for _, typeName := range orders {
		typeSpec := g.P.StructTypes[typeName]
		if typeSpec == nil {
			continue
		}

		d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])
		if d.SkipType {
			continue
		}

		// Get package directory from file path in PackagePaths
		filePath := g.P.PackagePaths[typeName]
		pkgDir := filepath.Dir(filePath)
		pkg := getPackage(pkgDir)

		// Add types
		if d.HasTypeDirective {
			pkg.types[typeName] = append(pkg.types[typeName], d.Types...)
		}

		// Add inputs
		if d.HasInputDirective {
			pkg.inputs[typeName] = append(pkg.inputs[typeName], d.Inputs...)
		}
	}

	// Sort package paths for deterministic output
	pkgPaths := make([]string, 0, len(packages))
	for pkgPath := range packages {
		pkgPaths = append(pkgPaths, pkgPath)
	}
	sort.Strings(pkgPaths)

	// Collect all content in memory first (map of file path -> content)
	fileContents := make(map[string]*strings.Builder)

	// Generate content for each package
	for _, pkgPath := range pkgPaths {
		items := packages[pkgPath]

		// Determine output file name from package path
		var outFile string
		if pkgPath == "_default" {
			// Types without package go to default location
			outFile = filepath.Join(g.Config.Output, g.Config.OutputFileName)
		} else {
			// Use the last segment of the package path as the file name
			// e.g., "/path/to/models" -> "models.graphqls"
			pkgName := filepath.Base(pkgPath)
			// Remove .go extension if present
			pkgName = strings.TrimSuffix(pkgName, ".go")
			outFile = filepath.Join(g.Config.Output, pkgName+g.Config.OutputFileExtension)
		}

		if g.Config.SkipExisting && FileExists(outFile) {
			fmt.Println("skip", outFile)
			continue
		}

		// Get or create buffer for this file
		buf, exists := fileContents[outFile]
		if !exists {
			buf = &strings.Builder{}
			fileContents[outFile] = buf
		}

		// Generate enums for this package
		for _, enumName := range items.enums {
			enumType := g.P.EnumTypes[enumName]
			if enumType == nil {
				continue
			}
			enumContent := g.generateEnum(enumType)
			if enumContent != "" {
				buf.WriteString(enumContent)
				buf.WriteString("\n")
			}
		}

		// Generate types for this package
		for typeName, typeDefs := range items.types {
			typeSpec := g.P.StructTypes[typeName]
			structType := g.P.Structs[typeName]
			d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])

			for _, typeDef := range typeDefs {
				typeContent := g.generateTypeFromDef(typeSpec, structType, d, typeDef)
				if typeContent != "" {
					buf.WriteString(typeContent)
				}
			}
		}

		// Generate inputs for this package
		for typeName, inputDefs := range items.inputs {
			typeSpec := g.P.StructTypes[typeName]
			structType := g.P.Structs[typeName]
			d := ParseDirectives(typeSpec, g.P.TypeToDecl[typeName])

			for _, inputDef := range inputDefs {
				inputContent := g.generateInputFromDef(typeSpec, structType, d, inputDef)
				if inputContent != "" {
					buf.WriteString(inputContent)
				}
			}
		}
	}

	// Write all files at once
	for outFile, buf := range fileContents {
		if buf.Len() > 0 {
			// Ensure directory exists
			if err := EnsureDir(filepath.Dir(outFile)); err != nil {
				return err
			}
			if err := WriteFile(outFile, buf.String(), g.Config); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) generateMultipleFiles(orders []string) error {
	// Collect all content in memory first (map of file path -> content)
	fileContents := make(map[string]*strings.Builder)

	// Generate enums first
	for _, enumName := range g.P.EnumNames {
		enumType := g.P.EnumTypes[enumName]
		if enumType == nil {
			continue
		}

		fileName := strings.ToLower(enumName) + g.Config.OutputFileExtension
		outFile := filepath.Join(g.Config.Output, fileName)

		if g.Config.SkipExisting && FileExists(outFile) {
			fmt.Println("skip", outFile)
			continue
		}

		// Get or create buffer for this file
		buf, exists := fileContents[outFile]
		if !exists {
			buf = &strings.Builder{}
			fileContents[outFile] = buf
		}

		enumContent := g.generateEnum(enumType)
		if enumContent != "" {
			buf.WriteString(enumContent)
		}
	}

	// Generate types and inputs
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

		// Get or create buffer for this file
		buf, exists := fileContents[outFile]
		if !exists {
			buf = &strings.Builder{}
			fileContents[outFile] = buf
		}

		// Generate all types from @gqlType directives
		if d.HasTypeDirective {
			for _, typeDef := range d.Types {
				typeContent := g.generateTypeFromDef(typeSpec, g.P.Structs[typeName], d, typeDef)
				if typeContent != "" {
					buf.WriteString(typeContent)
				}
			}
		}

		// Generate all inputs from @gqlInput directives
		if d.HasInputDirective {
			for _, inputDef := range d.Inputs {
				inputContent := g.generateInputFromDef(typeSpec, g.P.Structs[typeName], d, inputDef)
				if inputContent != "" {
					buf.WriteString(inputContent)
				}
			}
		}
	}

	// Write all files at once
	for outFile, buf := range fileContents {
		if buf.Len() > 0 {
			// Ensure directory exists
			if err := EnsureDir(filepath.Dir(outFile)); err != nil {
				return err
			}
			if err := WriteFile(outFile, buf.String(), g.Config); err != nil {
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

// generateTypeFromDef generates a GraphQL type from a specific TypeDefinition
func (g *Generator) generateTypeFromDef(typeSpec *ast.TypeSpec, st *ast.StructType, d StructDirectives, typeDef TypeDefinition) string {
	name := d.GQLName
	if typeDef.Name != "" {
		// Use custom type name from @gqlType annotation
		name = typeDef.Name
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
	if typeDef.Description != "" {
		buf.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", typeDef.Description))
	}

	// Type declaration
	buf.WriteString(fmt.Sprintf("type %s", name))

	// Add @goModel directive if enabled
	if g.Config.UseGqlGenDirectives || d.UseModelDirective {
		pkgPath := g.P.GetPackageImportPath(typeSpec.Name.Name, g.Config.ModelPath)
		buf.WriteString(fmt.Sprintf(" @goModel(model: \"%s.%s\")", pkgPath, typeSpec.Name.Name))
	}

	buf.WriteString(" {\n")

	// Generate fields from struct (use typeDef.IgnoreAll instead of d.TypeIgnoreAll)
	fields := g.generateFieldsForTypeNamed(st, d, typeDef.IgnoreAll, false, name)

	// Count applicable extra fields for this type
	applicableExtraFields := 0
	for _, ef := range d.TypeExtraFields {
		if shouldApplyExtraField(ef, name) {
			applicableExtraFields++
		}
	}

	if len(fields) == 0 && applicableExtraFields == 0 && !g.Config.IncludeEmptyTypes {
		return "" // Skip empty types
	}

	buf.WriteString(fields)

	// Add extra fields from @gqlTypeExtraField annotations
	for _, ef := range d.TypeExtraFields {
		// Check if this field applies to this type
		if !shouldApplyExtraField(ef, name) {
			continue
		}
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

// generateInputFromDef generates a GraphQL input from a specific InputDefinition
func (g *Generator) generateInputFromDef(typeSpec *ast.TypeSpec, st *ast.StructType, d StructDirectives, inputDef InputDefinition) string {
	inputName := inputDef.Name
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
	}

	buf := strings.Builder{}

	// Add description if present
	if inputDef.Description != "" {
		buf.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", inputDef.Description))
	}

	// Input declaration
	buf.WriteString(fmt.Sprintf("input %s", inputName))

	// Add @goModel directive if enabled
	if g.Config.UseGqlGenDirectives || d.UseModelDirective {
		pkgPath := g.P.GetPackageImportPath(typeSpec.Name.Name, g.Config.ModelPath)
		buf.WriteString(fmt.Sprintf(" @goModel(model: \"%s.%s\")", pkgPath, typeSpec.Name.Name))
	}

	buf.WriteString(" {\n")

	// Generate fields from struct (use inputDef.IgnoreAll instead of d.InputIgnoreAll)
	fields := g.generateFieldsForTypeNamed(st, d, inputDef.IgnoreAll, true, inputName)

	// Count applicable extra fields for this input
	applicableExtraFields := 0
	for _, ef := range d.InputExtraFields {
		if shouldApplyExtraField(ef, inputName) {
			applicableExtraFields++
		}
	}

	if len(fields) == 0 && applicableExtraFields == 0 && !g.Config.IncludeEmptyTypes {
		return "" // Skip empty inputs
	}

	buf.WriteString(fields)

	// Add extra fields from @gqlInputExtraField annotations
	for _, ef := range d.InputExtraFields {
		// Check if this field applies to this input
		if !shouldApplyExtraField(ef, inputName) {
			continue
		}
		if ef.Description != "" {
			buf.WriteString(fmt.Sprintf("\t\"\"\"%s\"\"\"\n", ef.Description))
		}
		buf.WriteString(fmt.Sprintf("\t%s: %s\n", ef.Name, ef.Type))
	}

	buf.WriteString("}\n\n")

	return buf.String()
}

// generateFieldsForType generates fields with specific ignoreAll setting
// func (g *Generator) generateFieldsForType(st *ast.StructType, d StructDirectives, typeIgnoreAll bool, forInput bool) string {
// 	return g.generateFieldsForTypeNamed(st, d, typeIgnoreAll, forInput, "")
// }

// generateFieldsForTypeNamed generates fields with specific ignoreAll setting and type/input name for filtering
func (g *Generator) generateFieldsForTypeNamed(st *ast.StructType, d StructDirectives, typeIgnoreAll bool, forInput bool, typeName string) string {
	buf := strings.Builder{}

	// Determine which ignoreAll flag to use
	ignoreAll := d.IgnoreAll || typeIgnoreAll

	for _, f := range st.Fields.List {
		// Handle embedded fields
		if f.Names == nil {
			// This is an embedded field - expand its fields
			embeddedFields := g.expandEmbeddedFieldNamed(f, d, ignoreAll, forInput, typeName)
			buf.WriteString(embeddedFields)
			continue
		}

		opt := ParseFieldOptions(f, g.Config)

		// Determine if field should be included using new type-specific logic
		include := shouldIncludeField(opt, ignoreAll, forInput, typeName)
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

		// Add @deprecated directive if field is deprecated
		if opt.Deprecated {
			if opt.DeprecatedReason != "" {
				// Escape quotes in the reason
				escapedReason := strings.ReplaceAll(opt.DeprecatedReason, `"`, `\"`)
				buf.WriteString(fmt.Sprintf(` @deprecated(reason: "%s")`, escapedReason))
			} else {
				buf.WriteString(" @deprecated")
			}
		}

		buf.WriteString("\n")
	}

	return buf.String()
}

// shouldApplyExtraField checks if an extra field should be applied to a given type/input name
// based on the 'on' filter. Returns true if:
// - On is empty (no filter specified, defaults to all)
// - On contains "*" (explicitly apply to all)
// - On contains the targetName
func shouldApplyExtraField(ef ExtraField, targetName string) bool {
	if len(ef.On) == 0 {
		return true
	}
	for _, name := range ef.On {
		if name == "*" || name == targetName {
			return true
		}
	}
	return false
}

// shouldIncludeField determines if a field should be included in a type/input
// based on the FieldOptions and context (forInput, typeName)
func shouldIncludeField(opt FieldOptions, ignoreAll bool, forInput bool, typeName string) bool {
	// Handle read-only (ro): include only in types, ignore in inputs
	if len(opt.ReadOnly) > 0 {
		if forInput {
			return false // Exclude from inputs
		}
		// Include in types only if typeName matches or is *
		return matchesTypeList(opt.ReadOnly, typeName)
	}

	// Handle write-only (wo): include only in inputs, ignore in types
	if len(opt.WriteOnly) > 0 {
		if !forInput {
			return false // Exclude from types
		}
		// Include in inputs only if typeName matches or is *
		return matchesTypeList(opt.WriteOnly, typeName)
	}

	// Handle read-write (rw): include in both types and inputs
	if len(opt.ReadWrite) > 0 {
		return matchesTypeList(opt.ReadWrite, typeName)
	}

	// Handle ignore/omit list (omit is alias for ignore)
	if len(opt.IgnoreList) > 0 && matchesTypeList(opt.IgnoreList, typeName) {
		return false
	}

	// Handle include list
	if len(opt.IncludeList) > 0 {
		return matchesTypeList(opt.IncludeList, typeName)
	}

	// Legacy behavior: check old boolean flags
	// Include if: (not ignored by ignoreAll AND not explicitly ignored/omitted) OR explicitly included
	return (!ignoreAll && !opt.Ignore && !opt.Omit) || opt.Include
}

// matchesTypeList checks if a typeName matches any entry in the list
// Returns true if list contains "*" or the exact typeName
func matchesTypeList(list []string, typeName string) bool {
	if len(list) == 0 {
		return false
	}
	for _, name := range list {
		if name == "*" || name == typeName {
			return true
		}
	}
	return false
}

// expandEmbeddedField recursively expands an embedded struct field into GraphQL fields
// func (g *Generator) expandEmbeddedField(f *ast.Field, d StructDirectives, ignoreAll bool, forInput bool) string {
// 	return g.expandEmbeddedFieldNamed(f, d, ignoreAll, forInput, "")
// }

// expandEmbeddedFieldNamed recursively expands an embedded struct field into GraphQL fields with type name
func (g *Generator) expandEmbeddedFieldNamed(f *ast.Field, d StructDirectives, ignoreAll bool, forInput bool, typeName string) string {
	// Get the type name of the embedded field
	var embeddedTypeName string
	switch t := f.Type.(type) {
	case *ast.Ident:
		embeddedTypeName = t.Name
	case *ast.StarExpr:
		// Handle pointer to embedded struct
		if ident, ok := t.X.(*ast.Ident); ok {
			embeddedTypeName = ident.Name
		}
	case *ast.SelectorExpr:
		// Handle embedded struct from another package (e.g., pkg.Type)
		embeddedTypeName = t.Sel.Name
	case *ast.IndexExpr:
		// Handle generic type with single type parameter (e.g., Connection[T])
		embeddedTypeName = g.extractGenericBaseName(t.X)
	case *ast.IndexListExpr:
		// Handle generic type with multiple type parameters (e.g., Map[K, V])
		embeddedTypeName = g.extractGenericBaseName(t.X)
	}

	if embeddedTypeName == "" {
		return "" // Unable to determine type name
	}

	// Look up the embedded struct in the parser
	embeddedStruct, exists := g.P.Structs[embeddedTypeName]
	if !exists {
		return "" // Embedded struct not found in parsed types
	}

	// Recursively generate fields for the embedded struct
	// Create a minimal StructDirectives for the embedded type (no special directives)
	embeddedDirectives := StructDirectives{
		IgnoreAll: ignoreAll, // Inherit ignoreAll setting
	}

	return g.generateFieldsForTypeNamed(embeddedStruct, embeddedDirectives, false, forInput, typeName)
}

// extractGenericBaseName extracts the base type name from a generic type expression
// For example: Connection[T] -> "Connection", pkg.Edge[T] -> "Edge"
func (g *Generator) extractGenericBaseName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		// Handle package-qualified generic types (e.g., pkg.Type[T])
		return t.Sel.Name
	case *ast.StarExpr:
		// Handle pointer to generic type (e.g., *Connection[T])
		return g.extractGenericBaseName(t.X)
	}
	return ""
}

// generateEnum generates a GraphQL enum definition from an EnumType
func (g *Generator) generateEnum(enumType *EnumType) string {
	buf := strings.Builder{}

	// Add description if present
	if enumType.Description != "" {
		buf.WriteString(fmt.Sprintf("\"\"\"\n%s\n\"\"\"\n", enumType.Description))
	}

	buf.WriteString(fmt.Sprintf("enum %s", enumType.Name))

	// Add @goModel directive if gqlgen directives are enabled
	if g.Config.UseGqlGenDirectives {
		pkgPath := g.P.GetPackageImportPath(enumType.GoTypeName, g.Config.ModelPath)
		buf.WriteString(fmt.Sprintf(" @goModel(model: \"%s.%s\")", pkgPath, enumType.GoTypeName))
	}

	buf.WriteString(" {\n")

	// Generate enum values
	for _, value := range enumType.Values {
		// Add description if present
		if value.Description != "" {
			buf.WriteString(fmt.Sprintf("  \"\"\"\n  %s\n  \"\"\"\n", value.Description))
		}

		// Add the enum value
		buf.WriteString(fmt.Sprintf("  %s", value.GraphQLName))

		// Add @goEnum directive if gqlgen directives are enabled
		if g.Config.UseGqlGenDirectives {
			// Use the package path where the const value is defined, not where the type is defined
			var valuePkgPath string
			if value.PackagePath != "" {
				// Use the stored package info for this specific const value
				valuePkgPath = g.P.GetPackageImportPathFromFile(value.PackagePath, value.PackageName, g.Config.ModelPath)
			} else {
				// Fallback to using the enum type's package (for backwards compatibility)
				valuePkgPath = g.P.GetPackageImportPath(enumType.GoTypeName, g.Config.ModelPath)
			}
			buf.WriteString(fmt.Sprintf(" @goEnum(value: \"%s.%s\")", valuePkgPath, value.GoName))
		}

		// Add deprecated directive if present
		if value.Deprecated != "" {
			buf.WriteString(fmt.Sprintf(" @deprecated(reason: \"%s\")", value.Deprecated))
		}

		buf.WriteString("\n")
	}

	buf.WriteString("}\n")

	return buf.String()
}
