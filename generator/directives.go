package generator

import (
	"go/ast"
	"reflect"
	"strings"
)

// ExtraField represents a @gqlExtraField annotation
type ExtraField struct {
	Name         string
	Type         string
	OverrideTags string
	Description  string
}

// StructDirectives holds parsed values from surrounding comments for a type
type StructDirectives struct {
	GQLName           string
	GQLInput          string       // @gqlInput(name:"InputName",description:"desc")
	GQLType           string       // @gqlType(name:"TypeName",description:"desc")
	TypeDescription   string       // Description from @gqlType
	InputDescription  string       // Description from @gqlInput
	IgnoreAll         bool         // @gqlIgnoreAll
	TypeIgnoreAll     bool         // ignoreAll property in @gqlType
	InputIgnoreAll    bool         // ignoreAll property in @gqlInput
	UseModelDirective bool         // @gqlUseModelDirective
	SkipType          bool         // @gqlskip
	GenInput          bool         // Generate input type
	HasTypeDirective  bool         // Has @gqlType directive
	HasInputDirective bool         // Has @gqlInput directive
	Partial           bool         // @partial
	ExtraFields       []ExtraField // @gqlExtraField (repeatable)
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
				if strings.HasPrefix(line, "@gqlType") {
					res.HasTypeDirective = true
					params := parseDirectiveParams(line, "@gqlType")
					if name, ok := params["name"]; ok && name != "" {
						res.GQLType = name
						res.GQLName = name
					}
					if desc, ok := params["description"]; ok {
						res.TypeDescription = desc
					}
					if ignoreAll, ok := params["ignoreAll"]; ok && (ignoreAll == "true" || ignoreAll == "1") {
						res.TypeIgnoreAll = true
					}
				}

				// @gqlInput(name:"InputName",description:"desc",ignoreAll:true)
				if strings.HasPrefix(line, "@gqlInput") {
					res.HasInputDirective = true
					res.GenInput = true // Enable input generation
					params := parseDirectiveParams(line, "@gqlInput")
					if name, ok := params["name"]; ok && name != "" {
						res.GQLInput = name
					}
					if desc, ok := params["description"]; ok {
						res.InputDescription = desc
					}
					if ignoreAll, ok := params["ignoreAll"]; ok && (ignoreAll == "true" || ignoreAll == "1") {
						res.InputIgnoreAll = true
					}
				} // @gqlIgnoreAll
				if strings.HasPrefix(line, "@gqlIgnoreAll") || strings.HasPrefix(line, "@ignoreall") {
					res.IgnoreAll = true
				}

				// @gqlUseModelDirective
				if strings.HasPrefix(line, "@gqlUseModelDirective") {
					res.UseModelDirective = true
				}

				// @gqlExtraField(name:"fieldName",type:"FieldType",overrideTags:"tags",description:"desc")
				if strings.HasPrefix(line, "@gqlExtraField") {
					params := parseDirectiveParams(line, "@gqlExtraField")
					if name, ok := params["name"]; ok && name != "" {
						ef := ExtraField{Name: name}
						if typ, ok := params["type"]; ok {
							ef.Type = typ
						}
						if desc, ok := params["description"]; ok {
							ef.Description = desc
						}
						if tags, ok := params["overrideTags"]; ok {
							ef.OverrideTags = tags
						}
						if ef.Type != "" {
							res.ExtraFields = append(res.ExtraFields, ef)
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
	Name          string
	Ignore        bool
	Include       bool
	Omit          bool
	Optional      bool
	Required      bool
	Type          string // Custom GraphQL type
	ForceResolver bool
	Description   string
}

// ParseFieldOptions parses `gql:"name,omit|include,optional|required,type:GqlType,forceResolver,description:\"desc\""`
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

	// Parse gql tag
	parts := strings.Split(g, ",")

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

		// Handle key:value pairs (type: and description:)
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
	case "ignore", "omit", "include", "optional", "required", "forceResolver":
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
