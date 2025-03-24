package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/keeley1/novelti-backend/api"
	"github.com/keeley1/novelti-backend/caching"
	"github.com/keeley1/novelti-backend/database"
	"github.com/keeley1/novelti-backend/models"
)

func callGoogleBooksAPI(query string, searchType string, startIndex int, detailed bool, db *sql.DB) ([]models.Book, error) {
	fmt.Println("Call api:", startIndex)

	resp, err := api.MakeAPICall(query, searchType, startIndex)
	if err != nil {
		// log and return error
		log.Println("Error calling API:", err)
		return nil, fmt.Errorf("failed to make API call: %w", err)
	}
	defer resp.Body.Close()

	volumeData, err := api.DecodeResponse(resp)
	if err != nil {
		log.Println("Error decoding response:", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	books, err := api.ExtractBookInfo(volumeData, searchType, query, detailed, db)
	if err != nil {
		log.Println("Error extracting book information:", err)
		return nil, fmt.Errorf("failed to extract book information: %w", err)
	}

	return books, nil
}

func main() {
	router := gin.Default()

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	router.Use(cors.New(config))

	// initialise database
	db, err := database.InitialiseDB()
	if err != nil {
		log.Println("Error initialising database:", err)
	}

	// ---------- testing db insertion ----------
	bookID := "hUZWAAAAcAAJ"
	rating := 4.5
	user := "userOne"

	// call add book function
	err = database.AddBook(db, bookID)
	if err != nil {
		fmt.Println("Failed to add book:", err)
	}

	// call add review function
	err = database.AddBookReview(db, bookID, rating, user)
	if err != nil {
		fmt.Println("Failed to add review:", err)
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World",
		})
	})

	router.GET("/searchbygenre/:subject", func(c *gin.Context) {
		genre := c.Param("subject")
		startIndexStr := c.Query("startIndex")
		startIndex := 0

		// could move this logic to function
		if startIndexStr != "" {
			var err error
			startIndex, err = strconv.Atoi(strings.TrimSpace(startIndexStr))
			if err != nil || startIndex < 0 {
				startIndex = 0
			}
		}

		fmt.Println("Query param:", startIndex)

		// save start index in cache key to ensure correct data is cached
		cacheKey := fmt.Sprintf("%s:%d", genre, startIndex)
		if books, found := caching.GetFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		// check database

		books, err := callGoogleBooksAPI(genre, "subject", startIndex, false, db)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		caching.SaveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	router.GET("/searchbytitle/:title", func(c *gin.Context) {
		title := c.Param("title")
		startIndexStr := c.Query("startIndex")
		startIndex := 0

		if startIndexStr != "" {
			var err error
			startIndex, err = strconv.Atoi(strings.TrimSpace(startIndexStr))
			if err != nil || startIndex < 0 {
				startIndex = 0
			}
		}

		cacheKey := fmt.Sprintf("%s:%d", title, startIndex)
		if books, found := caching.GetFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		books, err := callGoogleBooksAPI(title, "intitle", startIndex, false, db)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// err1 := database.AddBookRating("some-book-id", 4.5)
		// if err1 != nil {
		// 	fmt.Println("Error adding rating:", err)
		// } else {
		// 	fmt.Println("Rating added successfully.")
		// }

		caching.SaveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	router.GET("/searchbooks/:searchquery", func(c *gin.Context) {
		searchQuery := c.Param("searchquery")
		startIndexStr := c.Query("startIndex")
		startIndex := 0

		if startIndexStr != "" {
			var err error
			startIndex, err = strconv.Atoi(strings.TrimSpace(startIndexStr))
			if err != nil || startIndex < 0 {
				startIndex = 0
			}
		}

		encodedQuery := url.QueryEscape(searchQuery)

		cacheKey := fmt.Sprintf("%s:%d", encodedQuery, startIndex)
		if books, found := caching.GetFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		books, err := callGoogleBooksAPI(encodedQuery, "searchquery", startIndex, false, db)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		caching.SaveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	router.GET("/searchbyid/:id", func(c *gin.Context) {
		id := c.Param("id")
		cacheKey := "id:" + id

		if books, found := caching.GetFromCache(cacheKey); found {
			c.JSON(200, books)
			return
		}

		books, err := callGoogleBooksAPI(id, "id", 0, true, db)
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

		caching.SaveToCache(cacheKey, books)
		c.JSON(200, books)
	})

	router.POST("/addthumbnail", func(c *gin.Context) {
		var request struct {
			BookID    string `json:"book_id" binding:"required"`
			Thumbnail string `json:"thumbnail" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		err := database.InsertBooks(db, request.BookID, request.Thumbnail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Thumbnail added successfully"})
	})

	// ------ TESTING ENDPOINT ------
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

	router.GET("/thumbnail/:book_id", func(ctx *gin.Context) {
		bookID := ctx.Param("book_id")

		thumbnail, err := database.GetBookThumbnail(db, bookID)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"thumbnail": thumbnail})
	})

	router.Run(":8080")
}
