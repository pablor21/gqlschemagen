package models

import "time"

/**
 * @gqlType(name:"Post",description:"A blog post")
 * @gqlInput(name:"CreatePostInput")
 * @gqlInput(name:"UpdatePostInput")
 */
type Post struct {
	ID          string     `gql:"id,type:ID,description:Unique post identifier"`
	Title       string     `gql:"title,required,description:Post title"`
	Slug        string     `gql:"slug,required,description:URL-friendly slug"`
	Content     string     `gql:"content,required,description:Post content (markdown)"`
	Excerpt     *string    `gql:"excerpt,optional,description:Short excerpt"`
	AuthorID    string     `gql:"authorID,type:ID,description:Author's user ID"`
	Author      *User      `gql:"author,forceResolver,description:Post author"`
	Tags        []string   `gql:"tags,description:Post tags"`
	Status      PostStatus `gql:"status,description:Post status"`
	ViewCount   int        `gql:"viewCount,description:Number of views"`
	PublishedAt *time.Time `gql:"publishedAt,type:DateTime,optional,description:Publication timestamp"`
	CreatedAt   time.Time  `gql:"createdAt,type:DateTime,forceResolver,description:Creation timestamp"`
	UpdatedAt   time.Time  `gql:"updatedAt,type:DateTime,forceResolver,description:Last update timestamp"`
}

/**
 * @gqlEnum(description:"Post publication status")
 */
type PostStatus string

const (
	PostStatusDraft     PostStatus = "DRAFT"
	PostStatusPublished PostStatus = "PUBLISHED"
	PostStatusArchived  PostStatus = "ARCHIVED"
)

/**
 * @gqlType(description:"A comment on a post")
 * @gqlInput(name:"CreateCommentInput")
 */
type Comment struct {
	ID        string    `gql:"id,type:ID,description:Comment ID"`
	PostID    string    `gql:"postID,type:ID,description:Associated post ID"`
	Post      *Post     `gql:"post,forceResolver,description:The post this comment belongs to"`
	AuthorID  string    `gql:"authorID,type:ID,description:Comment author's user ID"`
	Author    *User     `gql:"author,forceResolver,description:Comment author"`
	Content   string    `gql:"content,required,description:Comment content"`
	CreatedAt time.Time `gql:"createdAt,type:DateTime,forceResolver,description:Creation timestamp"`
	UpdatedAt time.Time `gql:"updatedAt,type:DateTime,forceResolver,description:Last update timestamp"`
}
