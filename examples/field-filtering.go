package examples

// Field Filtering Examples
// This file demonstrates the new field filtering capabilities using gql struct tags

// Example 1: Read-Only Fields (ro)
// Fields marked with 'ro' appear only in types, not in inputs
// @gqlType(name:"User")
// @gqlInput(name:"UserInput")
type UserWithReadOnly struct {
	ID        string `gql:"id,type:ID,ro"` // Only in User type, not in UserInput
	CreatedAt string `gql:"createdAt,ro"`  // Only in User type, not in UserInput
	Name      string `gql:"name"`          // In both User type and UserInput
	Email     string `gql:"email"`         // In both User type and UserInput
}

// Example 2: Write-Only Fields (wo)
// Fields marked with 'wo' appear only in inputs, not in types
// @gqlType(name:"User")
// @gqlInput(name:"CreateUserInput")
type UserWithPassword struct {
	ID       string `gql:"id,type:ID,ro"` // Only in User type
	Name     string `gql:"name"`          // In both
	Password string `gql:"password,wo"`   // Only in CreateUserInput, not in User type
}

// Example 3: Include Field in Specific Types
// Use 'include:' to specify which types should have this field
// Supports three syntax styles: "Type,Type", 'Type,Type', or [Type,Type]
// @gqlType(name:"PublicUser")
// @gqlType(name:"AdminUser")
// @gqlType(name:"SuperAdminUser")
type UserWithConditionalFields struct {
	ID        string `gql:"id,type:ID"`
	Email     string `gql:"email,include:\"AdminUser,SuperAdminUser\""` // Double quotes
	Phone     string `gql:"phone,include:'AdminUser,SuperAdminUser'"`   // Single quotes (alternative)
	Address   string `gql:"address,include:[AdminUser,SuperAdminUser]"` // Square brackets (alternative)
	SecretKey string `gql:"secretKey,include:SuperAdminUser"`           // Only in SuperAdminUser (single value)
}

// Example 4: Omit/Ignore Field from Specific Types
// Use 'omit:' or 'ignore:' to exclude field from specific types (they're aliases)
// @gqlType(name:"FullUser")
// @gqlType(name:"PartialUser")
type UserWithOmissions struct {
	ID    string `gql:"id,type:ID"`
	Name  string `gql:"name"`
	Email string `gql:"email,omit:PartialUser"`   // Excluded from PartialUser
	Phone string `gql:"phone,ignore:PartialUser"` // Also excluded from PartialUser (omit and ignore are aliases)
}

// Example 5: Read-Only for Specific Types
// Combine 'ro' with type list to make field read-only for specific types only
// @gqlType(name:"AdminView")
// @gqlType(name:"UserView")
// @gqlInput(name:"AdminInput")
// @gqlInput(name:"UserInput")
type AccountData struct {
	ID         string  `gql:"id,type:ID,ro"`                     // Read-only for all types
	Name       string  `gql:"name"`                              // In all types and inputs
	SecretData string  `gql:"secretData,ro:AdminView"`           // Only in AdminView type (single value)
	Balance    float64 `gql:"balance,ro:\"AdminView,UserView\""` // Read-only in both AdminView and UserView
}

// Example 6: Wildcard Usage
// Use '*' to apply to all types/inputs
// @gqlIgnoreAll  // Ignore all fields by default
// @gqlType(name:"User")
// @gqlInput(name:"UserInput")
type SelectiveUser struct {
	ID       string `gql:"id,type:ID,include:*"` // Included in all (overrides @gqlIgnoreAll)
	Name     string `gql:"name,include"`         // Included in all (shorthand for include:*)
	Email    string `gql:"email,rw:*"`           // Read-write in all (overrides @gqlIgnoreAll)
	Internal string // Ignored (due to @gqlIgnoreAll)
}

// Example 7: Multiple Types with Different Visibility
// @gqlType(name:"UserV1")
// @gqlType(name:"UserV2")
// @gqlType(name:"UserV3")
type EvolvingUser struct {
	ID       string `gql:"id,type:ID"`                      // In all versions
	Name     string `gql:"name"`                            // In all versions
	Email    string `gql:"email,include:\"UserV2,UserV3\""` // Added in V2, kept in V3
	Phone    string `gql:"phone,include:UserV3"`            // Added in V3 only (single value)
	OldField string `gql:"oldField,omit:\"UserV2,UserV3\""` // Only in V1, removed in V2+
}

// Example 8: Complex Scenario - Different Fields for Different Contexts
// @gqlType(name:"PublicProfile")
// @gqlType(name:"PrivateProfile")
// @gqlInput(name:"CreateProfileInput")
// @gqlInput(name:"UpdateProfileInput")
type ProfileData struct {
	ID           string `gql:"id,type:ID,ro"`                                // Read-only in types
	Username     string `gql:"username,ro:\"PublicProfile,PrivateProfile\""` // Read-only in types, editable in inputs
	DisplayName  string `gql:"displayName"`                                  // In all
	Email        string `gql:"email,omit:PublicProfile"`                     // Hidden from public view (single value)
	Bio          string `gql:"bio"`                                          // In all
	PrivateNotes string `gql:"privateNotes,include:PrivateProfile"`          // Only in private view (single value)
	Password     string `gql:"password,wo"`                                  // Write-only (inputs only)
}

// Summary of available tags:
//
// Basic flags (no type list):
//   - ro              : Read-only everywhere (types only, excluded from all inputs)
//   - wo              : Write-only everywhere (inputs only, excluded from all types)
//   - rw              : Read-write everywhere (shorthand for include:*)
//   - include         : Include everywhere (overrides @gqlIgnoreAll)
//   - omit            : Omit/ignore everywhere
//   - ignore          : Omit/ignore everywhere (alias for omit)
//
// With single type:
//   - ro:TypeName     : Read-only for TypeName only (no quotes needed)
//   - wo:InputName    : Write-only for InputName only (no quotes needed)
//   - include:TypeName: Include only in TypeName (no quotes needed)
//   - omit:TypeName   : Exclude from TypeName (no quotes needed)
//
// With multiple types (REQUIRES QUOTES):
//   - ro:"TypeA,TypeB"        : Read-only for TypeA and TypeB only
//   - wo:"InputA,InputB"      : Write-only for InputA and InputB only
//   - rw:"TypeA,TypeB"        : Include in TypeA and TypeB (both types and inputs)
//   - include:"TypeA,TypeB"   : Include only in TypeA and TypeB
//   - omit:"TypeA,TypeB"      : Exclude from TypeA and TypeB
//   - ignore:"TypeA,TypeB"    : Exclude from TypeA and TypeB (alias for omit)
//
// Multiple types also support single quotes and square brackets:
//   - include:'TypeA,TypeB'   : Single quotes (alternative syntax)
//   - include:[TypeA,TypeB]   : Square brackets (alternative syntax)
//   - You can mix styles: include:"A,B",omit:'C,D',ro:[E,F]
//
// Wildcard:
//   - include:* or include : Include in all types/inputs
//   - omit:* or omit      : Exclude from all types/inputs
//   - rw:*                : Include in all types and inputs
//
// Notes:
//   - omit and ignore are aliases (same behavior)
//   - Multiple types can use: "TypeA,TypeB" or 'TypeA,TypeB' or [TypeA,TypeB]
//   - Single types don't need quotes: include:TypeName
//   - Separate type names with commas (no spaces): "TypeA,TypeB,TypeC"
//   - Type names in lists match the name specified in @gqlType/@gqlInput directives
