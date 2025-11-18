package models

import "time"

/**
 * @gqlType(name:"User",description:"Represents a user in the system")
 * @gqlInput(name:"CreateUserInput")
 * @gqlInput(name:"UpdateUserInput")
 */
type User struct {
	ID        string    `gql:"id,type:ID,description:Unique user identifier"`
	Email     string    `gql:"email,required,description:User's email address"`
	Username  string    `gql:"username,required,description:Unique username"`
	FirstName string    `gql:"firstName,description:User's first name"`
	LastName  string    `gql:"lastName,description:User's last name"`
	Bio       *string   `gql:"bio,optional,description:User biography"`
	Avatar    *string   `gql:"avatar,optional,description:Avatar URL"`
	IsActive  bool      `gql:"isActive,description:Whether the user is active"`
	Role      UserRole  `gql:"role,description:User's role in the system"`
	CreatedAt time.Time `gql:"createdAt,type:DateTime,forceResolver,description:Account creation timestamp"`
	UpdatedAt time.Time `gql:"updatedAt,type:DateTime,forceResolver,description:Last update timestamp"`
}

/**
 * @gqlEnum(description:"User role in the system")
 */
type UserRole string

const (
	UserRoleAdmin     UserRole = "ADMIN"
	UserRoleModerator UserRole = "MODERATOR"
	UserRoleUser      UserRole = "USER"
	UserRoleGuest     UserRole = "GUEST"
)

/**
 * @gqlType(description:"User profile information")
 */
type UserProfile struct {
	UserID      string  `gql:"userId,type:ID,required,description:Associated user ID"`
	DisplayName string  `gql:"displayName,description:Display name"`
	Location    *string `gql:"location,optional,description:User's location"`
	Website     *string `gql:"website,optional,description:User's website URL"`
	Twitter     *string `gql:"twitter,optional,description:Twitter handle"`
	GitHub      *string `gql:"github,optional,description:GitHub username"`
}
