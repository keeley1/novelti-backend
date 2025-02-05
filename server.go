package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Book struct {
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	PublishedDate string   `json:"publishedDate"`
	Cover         string   `json:"thumbnail"`
}

func main() {
	router := gin.Default()

	// add CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	router.Use(cors.New(config))

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})

	router.GET("/searchbygenre/:subject", func(c *gin.Context) {
		genre := c.Param("subject")
		encodedSubject := url.QueryEscape(genre)
		googleBooksAPIURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=+subject:%s&maxResults=20", encodedSubject)

		resp, err := http.Get(googleBooksAPIURL)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch data from Google Books API"})
			return
		}
		defer resp.Body.Close()

		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			c.JSON(500, gin.H{"error": "Failed to parse response body"})
			return
		}

		// extract and transform the data
		var books []Book
		if items, ok := data["items"].([]interface{}); ok {
			for _, item := range items {
				volumeInfo, ok := item.(map[string]interface{})["volumeInfo"].(map[string]interface{})
				if !ok {
					continue
				}

				book := Book{
					Title:         volumeInfo["title"].(string),
					PublishedDate: volumeInfo["publishedDate"].(string),
				}

				// handle authors array
				if authors, ok := volumeInfo["authors"].([]interface{}); ok {
					for _, author := range authors {
						book.Authors = append(book.Authors, author.(string))
					}
				}

				// handle thumbnail
				if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
					if thumbnail, ok := imageLinks["thumbnail"].(string); ok {
						book.Cover = thumbnail
					}
				}

				books = append(books, book)
			}
		}

		c.JSON(200, books)
	})

	router.Run(":8080")
}
