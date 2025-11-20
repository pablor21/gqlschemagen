package examples

// This example demonstrates how to use Go generics with gqlschemagen
// to create reusable Relay-style connection types.

// Edge represents a Relay connection edge with a generic node type
type Edge[T any] struct {
	Node   T      `json:"node"`
	Cursor string `json:"cursor"`
}

// Connection represents a Relay connection with generic edge type
type Connection[T any] struct {
	Edges    []*Edge[T] `json:"edges"`
	PageInfo *PageInfo  `json:"pageInfo"`
}

/**
 * @gqlType
 * Information about pagination in a connection
 */
type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

/**
 * @gqlType
 * A user in the system
 */
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

/**
 * @gqlType
 * A blog post
 */
type Post struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

/**
 * @gqlType
 * Paginated list of users with total count
 */
type UserConnection struct {
	Connection[*User]
	TotalCount int `gql:"type:Int!"`
}

/**
 * @gqlType
 * Paginated list of posts with metadata
 */
type PostConnection struct {
	Connection[*Post]
	TotalCount     int  `gql:"type:Int!"`
	HasUnpublished bool `json:"hasUnpublished"`
}

// Generated GraphQL schema will include:
//
// type UserConnection {
//   edges: [Edge!]!
//   pageInfo: PageInfo!
//   totalCount: Int!
// }
//
// type PostConnection {
//   edges: [Edge!]!
//   pageInfo: PageInfo!
//   totalCount: Int!
//   hasUnpublished: Boolean!
// }
//
// type PageInfo {
//   hasNextPage: Boolean!
//   hasPreviousPage: Boolean!
//   startCursor: String!
//   endCursor: String!
// }
