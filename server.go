package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CacheItem struct {
	Data      []Book
	Timestamp time.Time
}

var (
	cache    sync.Map
	cacheTTL = 10 * time.Minute
)

type Book struct {
	Title         string        `json:"title"`
	Authors       []string      `json:"authors"`
	PublishedDate string        `json:"publishedDate"`
	Cover         string        `json:"thumbnail"`
	ID            string        `json:"id"`
	Description   template.HTML `json:"description"`
}

func getFromGoogleBooks(query string, searchType string, detailed bool) ([]Book, error) {
	var googleBooksAPIURL string

	if searchType == "id" {
		// Direct volume endpoint for ID searches
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/%s", query)
	} else {
		// Regular search endpoint for other types
		encodedQuery := url.QueryEscape(query + " books")
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20", encodedQuery)
	}

	fmt.Println("API URL:", googleBooksAPIURL)

	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	var books []Book

	if searchType == "id" {
		// Parse single volume response
		var volumeData map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&volumeData); err != nil {
			return nil, fmt.Errorf("failed to parse response: %v", err)
		}

		// Get volume info
		if volumeInfo, ok := volumeData["volumeInfo"].(map[string]interface{}); ok {
			// Check for thumbnail
			var thumbnail string
			if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
				if _, ok := imageLinks["thumbnail"].(string); ok {
					// Extract the book ID from the thumbnail URL or use the existing ID
					bookID := query // we already have the book ID from earlier
					thumbnail = fmt.Sprintf("https://books.google.com/books/publisher/content/images/frontcover/%s?fife=w600-h800&source=gbs_api", bookID)
				}
			}
			if thumbnail == "" {
				return nil, fmt.Errorf("no thumbnail available")
			}

			book := Book{
				Title: volumeInfo["title"].(string),
				ID:    query,
				Cover: thumbnail,
			}

			if publishedDate, ok := volumeInfo["publishedDate"].(string); ok {
				book.PublishedDate = publishedDate
			}

			if authors, ok := volumeInfo["authors"].([]interface{}); ok {
				for _, author := range authors {
					book.Authors = append(book.Authors, author.(string))
				}
			}

			if detailed {
				if description, ok := volumeInfo["description"].(string); ok {
					book.Description = template.HTML(description)
				}
			}

			books = append(books, book)
		}
	} else {
		// Existing search results parsing code
		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to parse response: %v", err)
		}

		var seenIDs = make(map[string]bool)

		if items, ok := data["items"].([]interface{}); ok {
			for _, item := range items {
				id, ok := item.(map[string]interface{})["id"].(string)
				if !ok || seenIDs[id] {
					continue
				}
				seenIDs[id] = true

				volumeInfo, ok := item.(map[string]interface{})["volumeInfo"].(map[string]interface{})
				if !ok {
					continue
				}

				var thumbnail string
				if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
					if _, ok := imageLinks["thumbnail"].(string); ok {
						// Extract the book ID from the thumbnail URL or use the existing ID
						bookID := id // we already have the book ID from earlier
						thumbnail = fmt.Sprintf("https://books.google.com/books/publisher/content/images/frontcover/%s?fife=w400-h600&source=gbs_api", bookID)
					}
				}

				if thumbnail == "" {
					continue
				}

				book := Book{
					Title: volumeInfo["title"].(string),
					ID:    id,
					Cover: thumbnail,
				}

				if publishedDate, ok := volumeInfo["publishedDate"].(string); ok {
					book.PublishedDate = publishedDate
				} else {
					continue
				}

				if authors, ok := volumeInfo["authors"].([]interface{}); ok {
					for _, author := range authors {
						book.Authors = append(book.Authors, author.(string))
					}
				}

				if detailed {
					if description, ok := volumeInfo["description"].(string); ok {
						book.Description = template.HTML(description)
					}
				}

				books = append(books, book)
			}
		}
	}

	return books, nil
}

func getFromCache(key string) ([]Book, bool) {
	if cacheItem, ok := cache.Load(key); ok {
		item := cacheItem.(CacheItem)
		if time.Since(item.Timestamp) < cacheTTL {
			return item.Data, true
		}
		cache.Delete(key)
	}
	return nil, false
}

func saveToCache(key string, books []Book) {
	cache.Store(key, CacheItem{
		Data:      books,
		Timestamp: time.Now(),
	})
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

		if books, found := getFromCache(genre); found {
			c.JSON(200, books)
			return
		}

		books, err := getFromGoogleBooks(genre, "subject", false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		saveToCache(genre, books)
		c.JSON(200, books)
	})

	router.GET("/searchbytitle/:title", func(c *gin.Context) {
		title := c.Param("title")
		cacheKey := "title:" + title

		if books, found := getFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		books, err := getFromGoogleBooks(title, "intitle", false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		saveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	router.GET("/searchbyid/:id", func(c *gin.Context) {
		id := c.Param("id")
		cacheKey := "id:" + id

		if books, found := getFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		books, err := getFromGoogleBooks(id, "id", true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if len(books) == 0 {
			c.JSON(404, gin.H{
				"message": fmt.Sprintf("No book found with ID: %s", id),
				"id":      id,
			})
			return
		}

		saveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	// test endpoint
	router.GET("/test/:query", func(c *gin.Context) {
		query := c.Param("query")
		encodedQuery := url.QueryEscape(query)
		googleBooksAPIURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20", encodedQuery)

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

		fmt.Println(googleBooksAPIURL)
		c.JSON(200, data)
	})

	router.Run(":8080")
}
