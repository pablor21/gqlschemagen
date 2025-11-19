package examples

// Base contains common fields
type Base struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

/**
 * @gqlType(name:"Article")
 */
type Article struct {
	Base           // Embedded struct - fields will be expanded into Article type
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

// Timestamped provides timestamp fields
type Timestamped struct {
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Identifiable provides ID field
type Identifiable struct {
	ID string `json:"id"`
}

/**
 * @gqlType(name:"BlogPost")
 * @gqlInput(name:"BlogPostInput")
 */
type BlogPost struct {
	Identifiable        // Embedded - ID field will be included
	Timestamped         // Embedded - CreatedAt and UpdatedAt will be included
	Title        string `json:"title"`
	Body         string `json:"body"`
	Published    bool   `json:"published"`
}

// Metadata provides metadata fields
type Metadata struct {
	/**
	 * @gqlField(description:"Tags for categorization")
	 */
	Tags []string `json:"tags"`
	/**
	 * @gqlField(name:"viewCount",description:"Number of views")
	 */
	Views int `json:"views"`
}

/**
 * @gqlType(name:"ContentWithMetadata",description:"Content with embedded metadata")
 */
type ContentWithMetadata struct {
	Base
	Metadata
	Title       string `json:"title"`
	Description string `json:"description"`
}
