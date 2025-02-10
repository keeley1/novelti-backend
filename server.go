package main

import (
	"encoding/json"
	"fmt"
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
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	PublishedDate string   `json:"publishedDate"`
	Cover         string   `json:"thumbnail"`
	ISBN          string   `json:"isbn"`
	Description   string   `json:"description"`
}

func getFromGoogleBooks(query string, searchType string, isDetailed bool) ([]Book, error) {
	var googleBooksAPIURL string
	encodedQuery := url.QueryEscape(query + " books")

	if searchType != "isbn" {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20", encodedQuery)
	} else {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s:%s&maxResults=20", searchType, encodedQuery)
	}

	fmt.Println("API URL:", googleBooksAPIURL)

	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var books []Book
	seenISBNs := make(map[string]bool)

	if items, ok := data["items"].([]interface{}); ok {
		for _, item := range items {
			volumeInfo, ok := item.(map[string]interface{})["volumeInfo"].(map[string]interface{})
			if !ok {
				continue
			}

			// check for thumbnail
			var thumbnail string
			if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
				if thumb, ok := imageLinks["thumbnail"].(string); ok {
					thumbnail = thumb
				}
			}

			if thumbnail == "" {
				continue
			}

			// check for ISBN
			hasISBN := false
			var isbn string
			if industryIdentifiers, ok := volumeInfo["industryIdentifiers"].([]interface{}); ok {
				for _, identifier := range industryIdentifiers {
					if id, ok := identifier.(map[string]interface{}); ok {
						if id["type"].(string) == "ISBN_13" {
							hasISBN = true
							isbn = id["identifier"].(string)
							break
						}
					}
				}
			}

			if !hasISBN || seenISBNs[isbn] {
				continue
			}
			seenISBNs[isbn] = true

			book := Book{
				Title: volumeInfo["title"].(string),
				ISBN:  isbn,
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

			if isDetailed {
				if description, ok := volumeInfo["description"].(string); ok {
					book.Description = description
				}
			}

			books = append(books, book)
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

	router.GET("/searchbyisbn/:isbn", func(c *gin.Context) {
		isbn := c.Param("isbn")

		if books, found := getFromCache(isbn); found {
			c.JSON(200, books)
			return
		}

		books, err := getFromGoogleBooks(isbn, "isbn", true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		saveToCache(isbn, books)
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
