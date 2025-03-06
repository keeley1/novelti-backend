package models

import (
	"html/template"
)

type Book struct {
	Title         string        `json:"title"`
	Authors       []string      `json:"authors"`
	PublishedDate string        `json:"publishedDate"`
	Cover         string        `json:"thumbnail"`
	ID            string        `json:"id"`
	Description   template.HTML `json:"description"`
}
