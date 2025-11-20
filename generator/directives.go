package generator

import (
	"go/ast"
	"reflect"
	"strings"
	"unicode"
)

// ExtraField represents a @gqlTypeExtraField or @gqlInputExtraField annotation
type ExtraField struct {
	Name         string
	Type         string
	OverrideTags string
	Description  string
	ForType      bool     // true if this is a @gqlTypeExtraField
	ForInput     bool     // true if this is a @gqlInputExtraField
	On           []string // list of type/input names this applies to, empty or ["*"] means all
}

// TypeDefinition represents a single @gqlType annotation
type TypeDefinition struct {
	Name        string // Custom type name
	Description string // Type description
	IgnoreAll   bool   // ignoreAll property
	Namespace   string // Custom namespace override
}

// InputDefinition represents a single @gqlInput annotation
type InputDefinition struct {
	Name        string // Custom input name
	Description string // Input description
	IgnoreAll   bool   // ignoreAll property
	Namespace   string // Custom namespace override
}

// StructDirectives holds parsed values from surrounding comments for a type
type StructDirectives struct {
	GQLName           string            // Default struct name
	Types             []TypeDefinition  // All @gqlType annotations
	Inputs            []InputDefinition // All @gqlInput annotations
	IgnoreAll         bool              // @gqlIgnoreAll
	UseModelDirective bool              // @gqlUseModelDirective
	SkipType          bool              // @gqlskip
	GenInput          bool              // Generate input type
	HasTypeDirective  bool              // Has @gqlType directive
	HasInputDirective bool              // Has @gqlInput directive
	Partial           bool              // @partial
	TypeExtraFields   []ExtraField      // @gqlTypeExtraField (repeatable)
	InputExtraFields  []ExtraField      // @gqlInputExtraField (repeatable)
}

// ParseDirectives collects directives from GenDecl.Doc, TypeSpec.Doc and TypeSpec.Comment
func ParseDirectives(typeSpec *ast.TypeSpec, genDecl *ast.GenDecl) StructDirectives {
	res := StructDirectives{GQLName: typeSpec.Name.Name}
	var comments []*ast.CommentGroup
	if genDecl != nil && genDecl.Doc != nil {
		comments = append(comments, genDecl.Doc)
	}
	if typeSpec.Doc != nil {
		comments = append(comments, typeSpec.Doc)
	}
	if typeSpec.Comment != nil {
		comments = append(comments, typeSpec.Comment)
	}

	for _, cg := range comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			// normalize single-line comments starting with //
			text = strings.TrimPrefix(text, "//")
			text = strings.TrimSpace(text)
			// normalize block comments starting with /* or /**
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimPrefix(text, "/**")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)

			for _, line := range strings.Split(text, "\n") {
				// Trim whitespace, then remove leading *, then trim again
				line = strings.TrimSpace(line)
				line = strings.TrimPrefix(line, "*")
				line = strings.TrimSpace(line)

				// @gqlType(name:"TypeName",description:"desc",ignoreAll:true,namespace:"api/v1")
				if hasDirectivePrefix(line, "Type(") || hasDirectiveName(line, "Type") {
					res.HasTypeDirective = true
					line = normalizeDirective(line)
					params := parseDirectiveParams(line, "@gqlType")

					typeDef := TypeDefinition{}
					if name, ok := params["name"]; ok && name != "" {
						typeDef.Name = name
						if len(res.Types) == 0 {
							res.GQLName = name
						}
					}
					if desc, ok := params["description"]; ok {
						typeDef.Description = desc
					}
					if ignoreAll, ok := params["ignoreAll"]; ok && (ignoreAll == "true" || ignoreAll == "1") {
						typeDef.IgnoreAll = true
					}
					if namespace, ok := params["namespace"]; ok {
						typeDef.Namespace = namespace
					}
					res.Types = append(res.Types, typeDef)
				}

				// @gqlInput(name:"InputName",description:"desc",ignoreAll:true,namespace:"api/v1")
				if hasDirectivePrefix(line, "Input(") || hasDirectiveName(line, "Input") {
					res.HasInputDirective = true
					res.GenInput = true // Enable input generation
					line = normalizeDirective(line)
					params := parseDirectiveParams(line, "@gqlInput")

					inputDef := InputDefinition{}
					if name, ok := params["name"]; ok && name != "" {
						inputDef.Name = name
					}
					if desc, ok := params["description"]; ok {
						inputDef.Description = desc
					}
					if ignoreAll, ok := params["ignoreAll"]; ok && (ignoreAll == "true" || ignoreAll == "1") {
						inputDef.IgnoreAll = true
					}
					if namespace, ok := params["namespace"]; ok {
						inputDef.Namespace = namespace
					}
					res.Inputs = append(res.Inputs, inputDef)
				}

				// @gqlIgnoreAll or @GqlIgnoreAll
				if hasDirectivePrefix(line, "IgnoreAll") || strings.HasPrefix(line, "@ignoreall") {
					res.IgnoreAll = true
				}

				// @gqlUseModelDirective or @GqlUseModelDirective
				if hasDirectivePrefix(line, "UseModelDirective") {
					res.UseModelDirective = true
				}

				// @gqlExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Input1,Input2")
				if hasDirectivePrefix(line, "ExtraField") {
					line = normalizeDirective(line)
					params := parseDirectiveParams(line, "@gqlExtraField")
					if name, ok := params["name"]; ok && name != "" {
						ef := ExtraField{Name: name, ForType: true, ForInput: true}
						if typ, ok := params["type"]; ok {
							ef.Type = typ
						}
						if desc, ok := params["description"]; ok {
							ef.Description = desc
						}
						if tags, ok := params["overrideTags"]; ok {
							ef.OverrideTags = tags
						}
						if on, ok := params["on"]; ok {
							ef.On = parseListValue(on)
						} else {
							ef.On = []string{"*"}
						}
						if ef.Type != "" {
							res.InputExtraFields = append(res.InputExtraFields, ef)
							res.TypeExtraFields = append(res.TypeExtraFields, ef)
						}
					}
				}

				// @gqlTypeExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Type1,Type2")
				if hasDirectivePrefix(line, "TypeExtraField") {
					line = normalizeDirective(line)
					params := parseDirectiveParams(line, "@gqlTypeExtraField")
					if name, ok := params["name"]; ok && name != "" {
						ef := ExtraField{Name: name, ForType: true, ForInput: false}
						if typ, ok := params["type"]; ok {
							ef.Type = typ
						}
						if desc, ok := params["description"]; ok {
							ef.Description = desc
						}
						if tags, ok := params["overrideTags"]; ok {
							ef.OverrideTags = tags
						}
						if on, ok := params["on"]; ok {
							ef.On = parseListValue(on)
						} else {
							ef.On = []string{"*"}
						}
						if ef.Type != "" {
							res.TypeExtraFields = append(res.TypeExtraFields, ef)
						}
					}
				}

				// @gqlInputExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Input1,Input2")
				if hasDirectivePrefix(line, "InputExtraField") {
					line = normalizeDirective(line)
					params := parseDirectiveParams(line, "@gqlInputExtraField")
					if name, ok := params["name"]; ok && name != "" {
						ef := ExtraField{Name: name, ForType: false, ForInput: true}
						if typ, ok := params["type"]; ok {
							ef.Type = typ
						}
						if desc, ok := params["description"]; ok {
							ef.Description = desc
						}
						if tags, ok := params["overrideTags"]; ok {
							ef.OverrideTags = tags
						}
						if on, ok := params["on"]; ok {
							ef.On = parseListValue(on)
						} else {
							ef.On = []string{"*"}
						}
						if ef.Type != "" {
							res.InputExtraFields = append(res.InputExtraFields, ef)
						}
					}
				}
			}
		}
	}
	return res
}

// hasDirectivePrefix checks if line starts with @gql or @Gql followed by the given suffix
func hasDirectivePrefix(line, suffix string) bool {
	return strings.HasPrefix(line, "@gql"+suffix) || strings.HasPrefix(line, "@Gql"+suffix)
}

// hasDirectiveName checks if line exactly matches @gql or @Gql followed by the given name
func hasDirectiveName(line, name string) bool {
	return line == "@gql"+name || line == "@Gql"+name
}

// normalizeDirective converts @Gql to @gql for consistent parsing
func normalizeDirective(line string) string {
	if strings.HasPrefix(line, "@Gql") {
		return "@gql" + strings.TrimPrefix(line, "@Gql")
	}
	return line
}

// parseDirectiveParams parses directive parameters like @directive(name:"value",other:"value2")
func parseDirectiveParams(line, prefix string) map[string]string {
	result := make(map[string]string)

	// Remove directive prefix
	line = strings.TrimPrefix(line, prefix)
	line = strings.TrimSpace(line)

	// Remove parentheses if present
	line = strings.TrimPrefix(line, "(")
	line = strings.TrimSuffix(line, ")")
	line = strings.TrimSpace(line)

	if line == "" {
		return result
	}

	// Parse key:value pairs
	parts := splitParams(line)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, ":") {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		// Remove quotes from value
		value = strings.Trim(value, `"`)

		result[key] = value
	}

	return result
}

// splitParams splits parameters by comma, respecting quoted strings and brackets
func splitParams(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	bracketDepth := 0

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '"' {
			inQuotes = !inQuotes
			current.WriteByte(c)
		} else if c == '[' && !inQuotes {
			bracketDepth++
			current.WriteByte(c)
		} else if c == ']' && !inQuotes {
			bracketDepth--
			current.WriteByte(c)
		} else if c == ',' && !inQuotes && bracketDepth == 0 {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseListValue parses a value that can be either:
// - A comma-separated string: "Type1,Type2"
// - An array with double quotes: ["Type1","Type2"]
// - An array with single quotes: ['Type1','Type2']
// - An empty string: ""
// - An empty array: [] or ”
// Returns a slice of trimmed strings, or nil for empty values
func parseListValue(value string) []string {
	value = strings.TrimSpace(value)

	// Handle empty string or empty array literals
	if value == "" || value == "[]" || value == "''" {
		return nil
	}

	// Handle array syntax with brackets
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		// Remove brackets
		value = strings.TrimPrefix(value, "[")
		value = strings.TrimSuffix(value, "]")
		value = strings.TrimSpace(value)

		if value == "" {
			return nil
		}

		// Parse comma-separated items respecting quotes
		var items []string
		var current strings.Builder
		inQuotes := false
		quoteChar := byte(0)

		for i := 0; i < len(value); i++ {
			c := value[i]

			if !inQuotes && (c == '"' || c == '\'') {
				inQuotes = true
				quoteChar = c
				// Don't include the quote character itself
			} else if inQuotes && c == quoteChar {
				inQuotes = false
				quoteChar = 0
				// Don't include the quote character itself
			} else if c == ',' && !inQuotes {
				item := strings.TrimSpace(current.String())
				if item != "" {
					items = append(items, item)
				}
				current.Reset()
			} else if inQuotes || !unicode.IsSpace(rune(c)) {
				// Skip spaces outside quotes
				current.WriteByte(c)
			}
		}

		// Add the last item
		if current.Len() > 0 {
			item := strings.TrimSpace(current.String())
			if item != "" {
				items = append(items, item)
			}
		}

		return items
	}

	// Handle comma-separated string (legacy format)
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// FieldOptions describes parsed options from gql struct tag
type FieldOptions struct {
	Name             string
	Ignore           bool     // Legacy: ignore everywhere
	Include          bool     // Legacy: include everywhere
	Omit             bool     // Legacy: omit everywhere (alias for Ignore)
	IgnoreList       []string // List of types to ignore/omit this field in (supports *)
	IncludeList      []string // List of types to include this field in (supports *)
	ReadWrite        []string // rw: include in both types and inputs (supports *)
	ReadOnly         []string // ro: include only in types, ignore in inputs (supports *)
	WriteOnly        []string // wo: include only in inputs, ignore in types (supports *)
	Optional         bool
	Required         bool
	Type             string // Custom GraphQL type
	ForceResolver    bool
	Description      string
	Deprecated       bool   // Field is deprecated (flag only)
	DeprecatedReason string // Deprecation reason (if provided)
}

// ParseFieldOptions parses `gql:"name,omit|include,optional|required,type:GqlType,forceResolver,description:\"desc\",deprecated,deprecated:\"reason\""`
func ParseFieldOptions(field *ast.Field, config *Config) FieldOptions {
	res := FieldOptions{}
	if field.Tag == nil {
		return res
	}
	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
	g := tag.Get("gql")

	// If no gql tag, try json tag if enabled
	if g == "" && config.UseJsonTag {
		jsonTag := tag.Get("json")
		if jsonTag != "" {
			jsonParts := strings.Split(jsonTag, ",")
			// Check for json:"-" which means ignore the field
			if len(jsonParts) > 0 && jsonParts[0] == "-" {
				res.Ignore = true
				return res
			}
			if len(jsonParts) > 0 && jsonParts[0] != "" {
				res.Name = jsonParts[0]
			}
		}
		return res
	}

	if g == "" {
		return res
	}

	// Parse gql tag - handle key:value,value,value specially for lists
	parts := splitParamsWithLists(g)

	// First part is the name (unless empty/omitted)
	if len(parts) > 0 {
		firstPart := strings.TrimSpace(parts[0])
		// If first part is not empty and not a flag/key:value, it's the name
		if firstPart != "" && !strings.Contains(firstPart, ":") && !isKnownFlag(firstPart) {
			res.Name = firstPart
			parts = parts[1:] // Skip first part for remaining processing
		}
	}

	// Process remaining parts (or all parts if name was omitted)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Handle key:value pairs (type:, description:, deprecated:, include:, omit:, ignore:, rw:, ro:, wo:)
		if strings.Contains(p, ":") {
			kv := strings.SplitN(p, ":", 2)
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "type":
				res.Type = value
			case "description":
				// Remove quotes if present
				res.Description = strings.Trim(value, "\"'")
			case "deprecated":
				// deprecated:"reason" - mark as deprecated with reason
				res.Deprecated = true
				res.DeprecatedReason = strings.Trim(value, "\"'")
			case "include":
				// include:TypeA,TypeB or include:* or include (backwards compat)
				res.IncludeList = parseTypeList(value)
			case "omit", "ignore":
				// omit/ignore:TypeA,TypeB or omit/ignore:* (omit is alias for ignore)
				res.IgnoreList = parseTypeList(value)
			case "rw":
				// rw:TypeA,TypeB or rw:* or rw (read-write, both types and inputs)
				res.ReadWrite = parseTypeList(value)
			case "ro":
				// ro:TypeA,TypeB or ro:* or ro (read-only, types only)
				res.ReadOnly = parseTypeList(value)
			case "wo":
				// wo:TypeA,TypeB or wo:* or wo (write-only, inputs only)
				res.WriteOnly = parseTypeList(value)
			}
			continue
		}

		// Handle flags (no colon, just the flag name)
		switch p {
		case "ignore", "omit":
			// Legacy: ignore/omit everywhere (omit is alias for ignore)
			res.Ignore = true
			res.Omit = true
			res.IgnoreList = []string{"*"}
		case "include":
			// Legacy: include everywhere
			res.Include = true
			res.IncludeList = []string{"*"}
		case "rw":
			// Read-write: include in both types and inputs (all)
			res.ReadWrite = []string{"*"}
		case "ro":
			// Read-only: include only in types, not in inputs (all)
			res.ReadOnly = []string{"*"}
		case "wo":
			// Write-only: include only in inputs, not in types (all)
			res.WriteOnly = []string{"*"}
		case "optional":
			res.Optional = true
		case "required":
			res.Required = true
		case "forceResolver",
			"force_resolver":
			res.ForceResolver = true
		case "deprecated":
			// deprecated - mark as deprecated without reason
			res.Deprecated = true
		}
	}

	return res
}

// splitParamsWithLists splits parameters by comma, respecting quoted strings and brackets
// Comma-separated type lists can use: include:'TypeA,TypeB', include:"TypeA,TypeB", or include:[TypeA,TypeB]
func splitParamsWithLists(s string) []string {
	var parts []string
	var current strings.Builder
	var quoteChar byte // Stores the opening quote character (' or ")
	inQuotes := false
	inBrackets := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if !inQuotes && !inBrackets {
			if c == '\'' || c == '"' {
				// Opening quote
				inQuotes = true
				quoteChar = c
				current.WriteByte(c)
			} else if c == '[' {
				// Opening bracket
				inBrackets = true
				current.WriteByte(c)
			} else if c == ',' {
				// Comma outside quotes/brackets separates parameters
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		} else if inQuotes {
			current.WriteByte(c)
			if c == quoteChar {
				// Closing quote (matching the opening quote)
				inQuotes = false
			}
		} else if inBrackets {
			current.WriteByte(c)
			if c == ']' {
				// Closing bracket
				inBrackets = false
			}
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// isListKey checks if a key:value string has a key that expects a list
func isListKey(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, ":") {
		return false
	}
	key := strings.SplitN(s, ":", 2)[0]
	key = strings.TrimSpace(key)
	switch key {
	case "include", "omit", "ignore", "rw", "ro", "wo":
		return true
	}
	return false
}

// ResolveFieldName resolves field name based on config and tags
// Priority: gql tag name > json tag > struct field name (case transformation only applies to struct field)
func ResolveFieldName(field *ast.Field, config *Config) string {
	// 1. Check gql tag name (highest priority, always used if present)
	if field.Tag != nil {
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		g := tag.Get("gql")
		if g != "" {
			parts := strings.Split(g, ",")
			if len(parts) > 0 {
				firstPart := strings.TrimSpace(parts[0])
				// If first part is not empty and not a flag/key:value, it's the name
				if firstPart != "" && !strings.Contains(firstPart, ":") && !isKnownFlag(firstPart) {
					return firstPart
				}
			}
		}

		// 2. Try json tag (second priority)
		if config.UseJsonTag {
			if j := tag.Get("json"); j != "" {
				jsonName := strings.Split(j, ",")[0]
				if jsonName != "" && jsonName != "-" {
					return jsonName
				}
			}
		}
	}

	// 3. Use struct field name with case transformation (lowest priority)
	if len(field.Names) > 0 {
		name := field.Names[0].Name
		return TransformFieldName(name, config.FieldCase)
	}
	return ""
}

// TransformFieldName transforms a field name based on the case setting
func TransformFieldName(name string, fieldCase FieldCase) string {
	switch fieldCase {
	case FieldCaseSnake:
		return ToSnakeCase(name)
	case FieldCasePascal:
		return name // Keep as-is (PascalCase)
	case FieldCaseOriginal:
		return name // Keep as-is
	case FieldCaseNone:
		return name // Keep struct field name untouched
	case FieldCaseCamel:
		fallthrough
	default:
		// Convert to camelCase
		if len(name) == 0 {
			return name
		}
		// Handle acronyms at the start (e.g., "ID" -> "id", "URL" -> "url")
		if len(name) <= 3 && isAllUpper(name) {
			return strings.ToLower(name)
		}
		// Standard camelCase conversion
		return strings.ToLower(name[:1]) + name[1:]
	}
}

// isAllUpper checks if all letters in a string are uppercase
func isAllUpper(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return false
		}
	}
	return true
}

// isKnownFlag checks if a string is a known gql tag flag
func isKnownFlag(s string) bool {
	switch s {
	case "ignore", "omit", "include", "optional", "required", "forceResolver", "force_resolver", "deprecated", "rw", "ro", "wo":
		return true
	}
	return false
}

// parseTypeList parses a comma-separated list of type names
// Supports: "TypeA,TypeB", "*", or TypeA (single value)
// Returns ["*"] for empty values (backwards compatibility)
func parseTypeList(value string) []string {
	value = strings.TrimSpace(value)

	// Remove surrounding quotes or brackets if present
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			// Remove quotes (single or double)
			value = value[1 : len(value)-1]
		} else if value[0] == '[' && value[len(value)-1] == ']' {
			// Remove square brackets
			value = value[1 : len(value)-1]
		}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return []string{"*"}
	}
	if value == "*" {
		return []string{"*"}
	}

	// Split by comma for lists
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{"*"}
	}
	return result
} // StripPrefixSuffix removes specified prefixes and suffixes from a type name
// prefixList and suffixList are comma-separated strings
// e.g. "DB,Pg" and "DTO,Entity,Model"
func StripPrefixSuffix(name, prefixList, suffixList string) string {
	if prefixList != "" {
		prefixes := strings.Split(prefixList, ",")
		for _, prefix := range prefixes {
			prefix = strings.TrimSpace(prefix)
			if prefix != "" && strings.HasPrefix(name, prefix) {
				name = strings.TrimPrefix(name, prefix)
				break // Only strip one prefix
			}
		}
	}

	if suffixList != "" {
		suffixes := strings.Split(suffixList, ",")
		for _, suffix := range suffixes {
			suffix = strings.TrimSpace(suffix)
			if suffix != "" && strings.HasSuffix(name, suffix) {
				name = strings.TrimSuffix(name, suffix)
				break // Only strip one suffix
			}
		}
	}

	return name
}

// ToSnakeCase converts PascalCase to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// // extractDescription extracts description from comment group
// func extractDescription(commentGroup *ast.CommentGroup) string {
// 	if commentGroup == nil {
// 		return ""
// 	}

// 	var description []string
// 	for _, comment := range commentGroup.List {
// 		text := comment.Text
// 		// Normalize block comments
// 		text = strings.TrimPrefix(text, "/*")
// 		text = strings.TrimPrefix(text, "/**")
// 		text = strings.TrimSuffix(text, "*/")

// 		// Process each line
// 		for _, line := range strings.Split(text, "\n") {
// 			line = strings.TrimSpace(line)
// 			line = strings.TrimPrefix(line, "//")
// 			line = strings.TrimPrefix(line, "*")
// 			line = strings.TrimSpace(line)

// 			// Skip directive lines
// 			if strings.HasPrefix(line, "@") {
// 				continue
// 			}

// 			if line != "" {
// 				description = append(description, line)
// 			}
// 		}
// 	}

// 	return strings.Join(description, " ")
// }

// extractDirectiveParam extracts a parameter value from a directive comment
// e.g., @gqlEnumValue(name:"CUSTOM") -> extractDirectiveParam(text, "name") returns "CUSTOM"
func extractDirectiveParam(text, paramName string) string {
	// Look for pattern: paramName:"value" or paramName:'value'
	pattern := paramName + `:`
	idx := strings.Index(text, pattern)
	if idx == -1 {
		return ""
	}

	// Skip past the parameter name and colon
	text = text[idx+len(pattern):]
	text = strings.TrimSpace(text)

	// Extract quoted value
	if strings.HasPrefix(text, `"`) {
		// Find closing quote
		end := strings.Index(text[1:], `"`)
		if end != -1 {
			return text[1 : end+1]
		}
	} else if strings.HasPrefix(text, `'`) {
		// Find closing quote
		end := strings.Index(text[1:], `'`)
		if end != -1 {
			return text[1 : end+1]
		}
	}

	return ""
}
