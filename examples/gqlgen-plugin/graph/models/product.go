package models

import "time"

/**
 * @gqlType(name:"Product",description:"A product in the catalog")
 * @gqlInput(name:"CreateProductInput")
 * @gqlInput(name:"UpdateProductInput")
 */
type Product struct {
	ID          string    `gql:"id,type:ID,description:Product ID"`
	SKU         string    `gql:"sku,required,description:Stock keeping unit"`
	Name        string    `gql:"name,required,description:Product name"`
	Description string    `gql:"description,description:Product description"`
	Price       float64   `gql:"price,required,description:Product price"`
	Currency    string    `gql:"currency,description:Price currency code"`
	Stock       int       `gql:"stock,description:Available stock quantity"`
	CategoryID  string    `gql:"categoryID,type:ID,description:Category ID"`
	Category    *Category `gql:"category,forceResolver,description:Product category"`
	Images      []string  `gql:"images,description:Product image URLs"`
	IsActive    bool      `gql:"isActive,description:Whether product is active"`
	CreatedAt   time.Time `gql:"createdAt,type:DateTime,forceResolver"`
	UpdatedAt   time.Time `gql:"updatedAt,type:DateTime,forceResolver"`
}

/**
 * @gqlType(description:"Product category")
 * @gqlInput(name:"CreateCategoryInput")
 */
type Category struct {
	ID          string    `gql:"id,type:ID,description:Category ID"`
	Name        string    `gql:"name,required,description:Category name"`
	Slug        string    `gql:"slug,required,description:URL-friendly slug"`
	Description *string   `gql:"description,optional,description:Category description"`
	ParentID    *string   `gql:"parentID,type:ID,optional,description:Parent category ID"`
	Parent      *Category `gql:"parent,forceResolver,description:Parent category"`
	CreatedAt   time.Time `gql:"createdAt,type:DateTime,forceResolver"`
}
