package examples

// This file demonstrates the new array syntax support for the 'on' parameter
// in @GqlTypeExtraField, @GqlInputExtraField, and @GqlExtraField directives

// Example 1: Array syntax with double quotes
/**
 * @gqlType(name:"User")
 * @gqlType(name:"Admin")
 * @gqlTypeExtraField(name:"permissions",type:"[String!]!",description:"User permissions",on:["User","Admin"])
 */
type UserWithArraySyntax struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Example 2: Array syntax with single quotes
/**
 * @gqlType(name:"Article")
 * @gqlType(name:"BlogPost")
 * @gqlTypeExtraField(name:"tags",type:"[String!]!",description:"Content tags",on:['Article','BlogPost'])
 */
type ContentWithSingleQuotes struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Example 3: Empty array means apply to nothing (field won't be added)
/**
 * @gqlType(name:"Product")
 * @gqlTypeExtraField(name:"hidden",type:"String",description:"This won't appear",on:[])
 */
type ProductWithEmptyArray struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// Example 4: Empty string (alternative to empty array)
/**
 * @gqlInput(name:"CreateOrderInput")
 * @gqlInputExtraField(name:"specialField",type:"String",description:"This won't appear",on:"")
 */
type OrderWithEmptyString struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
}

// Example 5: Wildcard in array format
/**
 * @gqlType(name:"Comment")
 * @gqlType(name:"Reply")
 * @gqlTypeExtraField(name:"createdAt",type:"String!",description:"Creation timestamp",on:["*"])
 */
type CommentWithWildcard struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// Example 6: Mixed - GqlExtraField with array syntax (applies to both types and inputs)
/**
 * @gqlType(name:"Task")
 * @gqlInput(name:"TaskInput")
 * @gqlExtraField(name:"metadata",type:"String",description:"Task metadata",on:["Task","TaskInput"])
 */
type TaskWithMetadata struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// Example 7: Backward compatibility - comma-separated still works
/**
 * @gqlType(name:"Event")
 * @gqlType(name:"Notification")
 * @gqlTypeExtraField(name:"timestamp",type:"String!",description:"Event timestamp",on:"Event,Notification")
 */
type EventWithLegacySyntax struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// Example 8: Multiple extra fields with different array formats
/**
 * @gqlType(name:"Project")
 * @gqlInput(name:"CreateProjectInput")
 * @gqlInput(name:"UpdateProjectInput")
 * @gqlTypeExtraField(name:"contributors",type:"[User!]!",description:"Project contributors",on:["Project"])
 * @gqlInputExtraField(name:"ownerId",type:"ID!",description:"Owner ID",on:['CreateProjectInput'])
 * @gqlExtraField(name:"tags",type:"[String!]",description:"Project tags",on:"Project,CreateProjectInput")
 */
type ProjectWithMultipleFields struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Example 9: Single item in array
/**
 * @gqlType(name:"Document")
 * @gqlTypeExtraField(name:"version",type:"Int!",description:"Document version",on:["Document"])
 */
type DocumentWithSingleItemArray struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// Example 10: Array with spaces (should still work)
/**
 * @gqlType(name:"Report")
 * @gqlType(name:"Analysis")
 * @gqlTypeExtraField(name:"generatedAt",type:"String!",description:"Generation timestamp",on:[ "Report" , "Analysis" ])
 */
type ReportWithSpacedArray struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Data  string `json:"data"`
}
