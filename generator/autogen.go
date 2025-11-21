package generator

import (
	"go/ast"
	"path/filepath"
	"strings"
)

// TypeReference represents a reference from one type to another
type TypeReference struct {
	FromType string // Source type name
	ToType   string // Referenced type name
	Reason   string // "field", "embedded", "slice", "pointer"
	Depth    int    // Distance from annotated type
}

// TypeNode represents a type in the dependency graph
type TypeNode struct {
	Name              string
	PackagePath       string
	IsAnnotated       bool // Has @gqlType, @gqlInput, or @gqlEnum
	HasTypeDirective  bool // Has explicit @gqlType
	HasInputDirective bool // Has explicit @gqlInput
	ShouldGenType     bool // Should be generated as type
	ShouldGenInput    bool // Should be generated as input
	Depth             int  // Distance from nearest annotated type
	References        []string
	ReferencedBy      []string
}

// DependencyGraph tracks type relationships
type DependencyGraph struct {
	Nodes map[string]*TypeNode
	Edges map[string][]string // fromType -> []toTypes
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*TypeNode),
		Edges: make(map[string][]string),
	}
}

// AddNode adds a type to the graph
func (g *DependencyGraph) AddNode(typeName, packagePath string, isAnnotated, hasTypeDir, hasInputDir, hasIncludeDir bool) {
	if _, exists := g.Nodes[typeName]; !exists {
		g.Nodes[typeName] = &TypeNode{
			Name:              typeName,
			PackagePath:       packagePath,
			IsAnnotated:       isAnnotated,
			HasTypeDirective:  hasTypeDir,
			HasInputDirective: hasInputDir,
			ShouldGenType:     hasTypeDir || hasIncludeDir,  // @gqlType or @GqlInclude
			ShouldGenInput:    hasInputDir || hasIncludeDir, // @gqlInput or @GqlInclude
			Depth:             -1,                           // Will be set during traversal
			References:        []string{},
			ReferencedBy:      []string{},
		}
	}
}

// AddEdge adds a dependency edge (fromType references toType)
func (g *DependencyGraph) AddEdge(fromType, toType string) {
	if fromType == toType {
		return // Skip self-references
	}

	// Ensure nodes exist
	if _, exists := g.Nodes[fromType]; !exists {
		g.AddNode(fromType, "", false, false, false, false)
	}
	if _, exists := g.Nodes[toType]; !exists {
		g.AddNode(toType, "", false, false, false, false)
	}

	// Add edge if not already present
	if !contains(g.Edges[fromType], toType) {
		g.Edges[fromType] = append(g.Edges[fromType], toType)
		g.Nodes[fromType].References = append(g.Nodes[fromType].References, toType)
		g.Nodes[toType].ReferencedBy = append(g.Nodes[toType].ReferencedBy, fromType)
	}
}

// BuildDependencyGraph constructs a dependency graph from parsed structs
func (g *Generator) BuildDependencyGraph() *DependencyGraph {
	graph := NewDependencyGraph()

	// First pass: Add all types as nodes
	for typeName, typeSpec := range g.P.StructTypes {
		genDecl := g.P.TypeToDecl[typeName]
		packagePath := g.P.PackagePaths[typeName]

		// Parse directives to check if annotated
		directives := ParseDirectives(typeSpec, genDecl)

		// Check if this is a type alias to a generic instantiation (IndexExpr or IndexListExpr)
		// These should be automatically considered annotated for both type and input generation
		isGenericAlias := false
		if _, ok := typeSpec.Type.(*ast.IndexListExpr); ok {
			isGenericAlias = true
		} else if _, ok := typeSpec.Type.(*ast.IndexExpr); ok {
			isGenericAlias = true
		}

		isAnnotated := directives.HasTypeDirective || directives.HasInputDirective || directives.HasIncludeDirective || isGenericAlias
		hasType := directives.HasTypeDirective || isGenericAlias
		hasInput := directives.HasInputDirective || isGenericAlias

		graph.AddNode(typeName, packagePath, isAnnotated, hasType, hasInput, directives.HasIncludeDirective)
	}

	// Add enums as annotated nodes (enums are always types, never inputs)
	for enumName, enumType := range g.P.EnumTypes {
		// Get package path from the first value if available
		packagePath := ""
		if len(enumType.Values) > 0 {
			packagePath = enumType.Values[0].PackagePath
		}
		graph.AddNode(enumName, packagePath, true, true, false, false)
	}

	// Second pass: Build edges by analyzing struct fields
	for typeName, typeSpec := range g.P.StructTypes {
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok || structType.Fields == nil {
			continue
		}
		for _, field := range structType.Fields.List {
			// For embedded fields, only extract nested references, not the embedded type itself
			// Embedded types don't need to be generated separately - their fields are inlined
			if field.Names == nil { // Embedded field
				embeddedRefs := g.extractEmbeddedTypeReferences(field.Type)
				for _, refType := range embeddedRefs {
					if refType != "" && refType != typeName {
						graph.AddEdge(typeName, refType)
					}
				}
			} else {
				// For named fields, extract the field type (which SHOULD be generated)
				referencedTypes := g.extractTypeReferences(field.Type)
				for _, refType := range referencedTypes {
					if refType != "" && refType != typeName {
						graph.AddEdge(typeName, refType)
					}
				}
			}
		}
	}

	return graph
}

// extractTypeReferences extracts all type names referenced in an AST expression
func (g *Generator) extractTypeReferences(expr ast.Expr) []string {
	var types []string

	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type: User, string, int, etc.
		if !isBuiltinType(t.Name) {
			types = append(types, t.Name)
		}

	case *ast.StarExpr:
		// Pointer: *User
		types = append(types, g.extractTypeReferences(t.X)...)

	case *ast.ArrayType:
		// Slice/Array: []User, []*User
		types = append(types, g.extractTypeReferences(t.Elt)...)

	case *ast.SelectorExpr:
		// Package-qualified: pkg.User
		if ident, ok := t.X.(*ast.Ident); ok {
			// Try to find the type in our structs
			typeName := t.Sel.Name
			if ts, exists := g.P.StructTypes[typeName]; exists {
				if _, ok := ts.Type.(*ast.StructType); ok {
					types = append(types, typeName)
				}
			}
			// Also check with package prefix
			fullName := ident.Name + "." + typeName
			if ts, exists := g.P.StructTypes[fullName]; exists {
				if _, ok := ts.Type.(*ast.StructType); ok {
					types = append(types, fullName)
				}
			}
		}

	case *ast.MapType:
		// Map: map[string]User
		types = append(types, g.extractTypeReferences(t.Key)...)
		types = append(types, g.extractTypeReferences(t.Value)...)

	case *ast.IndexExpr:
		// Generic with single type param: Connection[T]
		baseTypes := g.extractTypeReferences(t.X)
		types = append(types, baseTypes...)
		argTypes := g.extractTypeReferences(t.Index)
		types = append(types, argTypes...)

	case *ast.IndexListExpr:
		// Generic with multiple type params: Map[K, V]
		baseTypes := g.extractTypeReferences(t.X)
		types = append(types, baseTypes...)
		for _, index := range t.Indices {
			argTypes := g.extractTypeReferences(index)
			types = append(types, argTypes...)
		}
	}

	return types
}

// extractEmbeddedTypeReferences recursively extracts type references from an embedded type's fields
// This is needed to catch dependencies in generic types like Connection[T] which embeds Edge[T] and references PageInfo
func (g *Generator) extractEmbeddedTypeReferences(expr ast.Expr) []string {
	var types []string

	// First, extract type arguments from generic embedded types (e.g., Connection[Comment])
	// These type arguments should be generated as types themselves
	switch t := expr.(type) {
	case *ast.IndexExpr:
		// Connection[T] or pkg.Connection[T]
		argTypes := g.extractTypeReferences(t.Index)
		types = append(types, argTypes...)
	case *ast.IndexListExpr:
		// Map[K, V] or pkg.Map[K, V]
		for _, index := range t.Indices {
			argTypes := g.extractTypeReferences(index)
			types = append(types, argTypes...)
		}
	case *ast.StarExpr:
		// *Connection[T] or *pkg.Connection[T]
		if idx, ok := t.X.(*ast.IndexExpr); ok {
			argTypes := g.extractTypeReferences(idx.Index)
			types = append(types, argTypes...)
		} else if idxList, ok := t.X.(*ast.IndexListExpr); ok {
			for _, index := range idxList.Indices {
				argTypes := g.extractTypeReferences(index)
				types = append(types, argTypes...)
			}
		}
	}

	// Then get the base type name from the expression
	var embeddedTypeName string

	switch t := expr.(type) {
	case *ast.Ident:
		embeddedTypeName = t.Name
	case *ast.StarExpr:
		// Handle pointer to embedded type
		if ident, ok := t.X.(*ast.Ident); ok {
			embeddedTypeName = ident.Name
		} else if idx, ok := t.X.(*ast.IndexExpr); ok {
			// *Connection[T] - extract base name
			if baseIdent, ok := idx.X.(*ast.Ident); ok {
				embeddedTypeName = baseIdent.Name
			}
		} else if idxList, ok := t.X.(*ast.IndexListExpr); ok {
			// *Map[K,V] - extract base name
			if baseIdent, ok := idxList.X.(*ast.Ident); ok {
				embeddedTypeName = baseIdent.Name
			}
		} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
			// *pkg.Type - extract selector name
			embeddedTypeName = sel.Sel.Name
		}
	case *ast.IndexExpr:
		// Connection[T] - extract base name
		if baseIdent, ok := t.X.(*ast.Ident); ok {
			embeddedTypeName = baseIdent.Name
		} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
			// pkg.Connection[T]
			embeddedTypeName = sel.Sel.Name
		}
	case *ast.IndexListExpr:
		// Map[K,V] - extract base name
		if baseIdent, ok := t.X.(*ast.Ident); ok {
			embeddedTypeName = baseIdent.Name
		} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
			// pkg.Map[K,V]
			embeddedTypeName = sel.Sel.Name
		}
	case *ast.SelectorExpr:
		// pkg.Type
		embeddedTypeName = t.Sel.Name
	}

	if embeddedTypeName == "" {
		return types
	}

	// Look up the embedded type
	typeSpec, exists := g.P.StructTypes[embeddedTypeName]
	if !exists {
		return types
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return types
	}

	// Extract references from all fields of the embedded type
	for _, field := range structType.Fields.List {
		fieldRefs := g.extractTypeReferences(field.Type)
		types = append(types, fieldRefs...)

		// Recursively handle nested embedded fields
		if field.Names == nil {
			nestedRefs := g.extractEmbeddedTypeReferences(field.Type)
			types = append(types, nestedRefs...)
		}
	}

	return types
}

// MarkTypesForGeneration traverses the graph and marks types for generation
func (g *DependencyGraph) MarkTypesForGeneration(config *Config) {
	if !config.AutoGenerate.Enabled {
		// Only generate explicitly annotated types and inputs
		// Nodes already have ShouldGenType and ShouldGenInput set from explicit directives
		return
	}

	switch config.AutoGenerate.Strategy {
	case AutoGenNone:
		// Only annotated - already set in AddNode
		return

	case AutoGenAll:
		// Generate everything as both types and inputs
		for _, node := range g.Nodes {
			if !g.isExcluded(node, config) {
				if !node.HasInputDirective {
					node.ShouldGenType = true
				}
				if !node.HasTypeDirective {
					node.ShouldGenInput = true
				}
			}
		}

	case AutoGenReferenced:
		// BFS from annotated types - context-aware
		g.markReferencedTypes(config)

	case AutoGenPatterns:
		// Pattern-based only
		g.markByPatterns(config)
	}
}

// markReferencedTypes marks types reachable from annotated types with context awareness
func (g *DependencyGraph) markReferencedTypes(config *Config) {
	// Find all annotated types as starting points
	// We track separately for types and inputs using BFS
	typeQueue := []string{}                     // Types that reference other types
	inputQueue := []string{}                    // Inputs that reference other types
	visited := make(map[string]map[string]bool) // typeName -> map["type"|"input"]bool

	for typeName, node := range g.Nodes {
		if node.IsAnnotated {
			node.Depth = 0
			visited[typeName] = make(map[string]bool)

			// If it's a type, its dependencies should be types
			if node.HasTypeDirective {
				typeQueue = append(typeQueue, typeName)
				visited[typeName]["type"] = true
			}

			// If it's an input, its dependencies should be inputs
			if node.HasInputDirective {
				inputQueue = append(inputQueue, typeName)
				visited[typeName]["input"] = true
			}
		}
	}

	// BFS traversal for type context
	for len(typeQueue) > 0 {
		current := typeQueue[0]
		typeQueue = typeQueue[1:]

		currentNode := g.Nodes[current]
		currentDepth := currentNode.Depth

		// Check depth limit
		maxDepth := config.AutoGenerate.MaxDepth
		if maxDepth > 0 && currentDepth >= maxDepth {
			continue
		}

		// Process references - they should also be types
		for _, refType := range g.Edges[current] {
			refNode, exists := g.Nodes[refType]
			if !exists {
				continue
			}

			// Skip if already visited in type context with lower or equal depth
			if visited[refType] != nil && visited[refType]["type"] && refNode.Depth <= currentDepth+1 {
				continue
			}

			// Check exclusion patterns
			if g.isExcluded(refNode, config) {
				continue
			}

			// Mark for generation as type
			refNode.ShouldGenType = true
			refNode.Depth = currentDepth + 1
			if visited[refType] == nil {
				visited[refType] = make(map[string]bool)
			}
			visited[refType]["type"] = true
			typeQueue = append(typeQueue, refType)
		}
	}

	// BFS traversal for input context
	for len(inputQueue) > 0 {
		current := inputQueue[0]
		inputQueue = inputQueue[1:]

		currentNode := g.Nodes[current]
		currentDepth := currentNode.Depth

		// Check depth limit
		maxDepth := config.AutoGenerate.MaxDepth
		if maxDepth > 0 && currentDepth >= maxDepth {
			continue
		}

		// Process references - they should also be inputs
		for _, refType := range g.Edges[current] {
			refNode, exists := g.Nodes[refType]
			if !exists {
				continue
			}

			// Skip if already visited in input context with lower or equal depth
			if visited[refType] != nil && visited[refType]["input"] && refNode.Depth <= currentDepth+1 {
				continue
			}

			// Check exclusion patterns
			if g.isExcluded(refNode, config) {
				continue
			}

			// Mark for generation as input
			refNode.ShouldGenInput = true
			refNode.Depth = currentDepth + 1
			if visited[refType] == nil {
				visited[refType] = make(map[string]bool)
			}
			visited[refType]["input"] = true
			inputQueue = append(inputQueue, refType)
		}
	}
}

// markByPatterns marks types matching inclusion patterns
func (g *DependencyGraph) markByPatterns(config *Config) {
	for _, node := range g.Nodes {
		// Annotated types already have their generation flags set
		if node.IsAnnotated {
			continue
		}

		if g.matchesPatterns(node, config.AutoGenerate.Patterns) &&
			!g.isExcluded(node, config) {
			// Generate as both type and input when matching patterns
			node.ShouldGenType = true
			node.ShouldGenInput = true
		}
	}
}

// isExcluded checks if a type should be excluded based on patterns
func (g *DependencyGraph) isExcluded(node *TypeNode, config *Config) bool {
	return g.matchesPatterns(node, config.AutoGenerate.ExcludePatterns)
}

// matchesPatterns checks if a type matches any of the given patterns
func (g *DependencyGraph) matchesPatterns(node *TypeNode, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	fullPath := filepath.Join(node.PackagePath, node.Name)

	for _, pattern := range patterns {
		// Simple glob matching
		matched, _ := filepath.Match(pattern, fullPath)
		if matched {
			return true
		}

		// Also try matching just the type name
		matched, _ = filepath.Match(pattern, node.Name)
		if matched {
			return true
		}

		// Try matching package path
		if strings.Contains(pattern, "/") {
			matched, _ = filepath.Match(pattern, node.PackagePath)
			if matched {
				return true
			}
		}
	}

	return false
}

// isBuiltinType checks if a type is a Go builtin
func isBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"bool":       true,
		"string":     true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"byte":       true,
		"rune":       true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
		"error":      true,
	}
	return builtins[typeName]
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// applyAutoGeneration updates the parser to include auto-generated types and inputs
func (g *Generator) applyAutoGeneration(graph *DependencyGraph) {
	// Mark which types should be auto-generated as types or inputs
	// Note: All struct types are already in g.P.TypeNames by the parser
	// We just need to mark which ones should actually generate schemas

	for typeName, node := range graph.Nodes {
		// Skip if type doesn't exist in parser's StructTypes
		typeSpec, exists := g.P.StructTypes[typeName]
		if !exists {
			continue
		}

		// Skip generic type definitions (types with type parameters like Edge[T])
		// Only concrete instantiations should be generated
		if typeSpec.TypeParams != nil && typeSpec.TypeParams.NumFields() > 0 {
			continue
		}

		// Check if type has explicit annotation or should be skipped
		genDecl := g.P.TypeToDecl[typeName]
		directives := ParseDirectives(typeSpec, genDecl)

		// Skip types with @gqlIgnore or @gqlskip
		if directives.SkipType {
			continue
		}

		// Auto-generate as type if needed and not explicitly annotated with @gqlType
		// Types with @GqlInclude are eligible for auto-generation
		if node.ShouldGenType && !directives.HasTypeDirective {
			g.AutoGeneratedTypes[typeName] = true
		}

		// Auto-generate as input if needed and not explicitly annotated with @gqlInput
		// Types with @GqlInclude are eligible for auto-generation
		if node.ShouldGenInput && !directives.HasInputDirective {
			g.AutoGeneratedInputs[typeName] = true
		}
	}
}
