package generator

import (
	"go/ast"
	"reflect"
	"strings"
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
}

// InputDefinition represents a single @gqlInput annotation
type InputDefinition struct {
	Name        string // Custom input name
	Description string // Input description
	IgnoreAll   bool   // ignoreAll property
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

				// @gqlType(name:"TypeName",description:"desc",ignoreAll:true)
				if strings.HasPrefix(line, "@gqlType(") || line == "@gqlType" {
					res.HasTypeDirective = true
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
					res.Types = append(res.Types, typeDef)
				}

				// @gqlInput(name:"InputName",description:"desc",ignoreAll:true)
				if strings.HasPrefix(line, "@gqlInput(") || line == "@gqlInput" {
					res.HasInputDirective = true
					res.GenInput = true // Enable input generation
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
					res.Inputs = append(res.Inputs, inputDef)
				} // @gqlIgnoreAll
				if strings.HasPrefix(line, "@gqlIgnoreAll") || strings.HasPrefix(line, "@ignoreall") {
					res.IgnoreAll = true
				}

				// @gqlUseModelDirective
				if strings.HasPrefix(line, "@gqlUseModelDirective") {
					res.UseModelDirective = true
				}

				// @gqlTypeExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Type1,Type2")
				if strings.HasPrefix(line, "@gqlTypeExtraField") {
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
							ef.On = strings.Split(on, ",")
							for i := range ef.On {
								ef.On[i] = strings.TrimSpace(ef.On[i])
							}
						} else {
							ef.On = []string{"*"}
						}
						if ef.Type != "" {
							res.TypeExtraFields = append(res.TypeExtraFields, ef)
						}
					}
				}

				// @gqlInputExtraField(name:"fieldName",type:"FieldType",description:"desc",on:"Input1,Input2")
				if strings.HasPrefix(line, "@gqlInputExtraField") {
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
							ef.On = strings.Split(on, ",")
							for i := range ef.On {
								ef.On[i] = strings.TrimSpace(ef.On[i])
							}
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

// splitParams splits parameters by comma, respecting quoted strings
func splitParams(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '"' {
			inQuotes = !inQuotes
			current.WriteByte(c)
		} else if c == ',' && !inQuotes {
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

// FieldOptions describes parsed options from gql struct tag
type FieldOptions struct {
	Name             string
	Ignore           bool
	Include          bool
	Omit             bool
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

	// Parse gql tag using splitParams to handle quoted values with commas
	parts := splitParams(g)

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

		// Handle key:value pairs (type:, description:, deprecated:)
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
			}
			continue
		}

		// Handle flags (no colon, just the flag name)
		switch p {
		case "ignore":
			res.Ignore = true
		case "omit":
			res.Omit = true
		case "include":
			res.Include = true
		case "optional":
			res.Optional = true
		case "required":
			res.Required = true
		case "forceResolver":
			res.ForceResolver = true
		case "deprecated":
			// deprecated - mark as deprecated without reason
			res.Deprecated = true
		}
	}

	return res
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
	case "ignore", "omit", "include", "optional", "required", "forceResolver", "deprecated":
		return true
	}
	return false
}

// StripPrefixSuffix removes specified prefixes and suffixes from a type name
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
