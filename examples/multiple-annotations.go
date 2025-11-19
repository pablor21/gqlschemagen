package examples

/**
 * @gqlType(name:"User",description:"Regular user account")
 * @gqlType(name:"AdminUser",description:"Administrator account with elevated privileges")
 * @gqlInput(name:"CreateUserInput",description:"Input for creating a new user")
 * @gqlInput(name:"UpdateUserInput",description:"Input for updating existing user")
 */
type UserDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	/**
	 * @gqlField(name:"fullName")
	 */
	Name string `json:"name"`
	/**
	 * @gqlField(ignore:"true")
	 */
	PasswordHash string `json:"-"`
}

/**
 * @gqlType(name:"Product")
 * @gqlType(name:"ProductPreview",ignoreAll:"true")
 * @gqlTypeExtraField(name:"title",type:"String!",description:"Product title for preview")
 * @gqlInput(name:"ProductInput")
 */
type ProductDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	/**
	 * @gqlField(include:"true")
	 */
	InternalCode string `json:"internal_code"`
}
