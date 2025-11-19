package examples

// Example 1: Basic TypeExtraField - adds field only to types
/**
 * @gqlType(name:"User")
 * @gqlInput(name:"UserInput")
 * @gqlTypeExtraField(name:"posts",type:"[Post!]!",description:"User's posts (only in type)")
 * @gqlInputExtraField(name:"password",type:"String!",description:"Password (only in input)")
 */
type BasicUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Example 2: Using 'on' parameter to target specific types
/**
 * @gqlType(name:"Article")
 * @gqlType(name:"BlogPost")
 * @gqlInput(name:"ArticleInput")
 * @gqlTypeExtraField(name:"author",type:"User!",description:"Article author",on:"Article")
 * @gqlTypeExtraField(name:"writer",type:"User!",description:"Blog writer",on:"BlogPost")
 * @gqlTypeExtraField(name:"comments",type:"[Comment!]!",description:"All comments")
 */
type Content struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Example 3: Using 'on' parameter to target specific inputs
/**
 * @gqlType(name:"Product")
 * @gqlInput(name:"CreateProductInput")
 * @gqlInput(name:"UpdateProductInput")
 * @gqlTypeExtraField(name:"reviews",type:"[Review!]!",description:"Product reviews")
 * @gqlInputExtraField(name:"vendorId",type:"ID!",description:"Vendor ID (only in CreateProductInput)",on:"CreateProductInput")
 */
type ProductItem struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// Example 4: Multiple 'on' targets
/**
 * @gqlType(name:"Order")
 * @gqlType(name:"Invoice")
 * @gqlType(name:"Receipt")
 * @gqlTypeExtraField(name:"customer",type:"User!",description:"Customer",on:"Order,Invoice")
 * @gqlTypeExtraField(name:"total",type:"Float!",description:"Total amount")
 */
type Transaction struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}
