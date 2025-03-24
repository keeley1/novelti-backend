package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/keeley1/novelti-backend/database"
	"github.com/keeley1/novelti-backend/models"
)

func constructAPIURL(query string, searchType string, startIndex int) string {
	var googleBooksAPIURL string

	if searchType == "id" {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/%s", query)
	} else if searchType == "searchquery" {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20&startIndex=%d", query, startIndex)
	} else {
		encodedQuery := url.QueryEscape(query + " books")
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20&startIndex=%d", encodedQuery, startIndex)
	}

	fmt.Println("API URL:", googleBooksAPIURL)
	return googleBooksAPIURL
}

func MakeAPICall(query string, searchType string, startIndex int) (*http.Response, error) {
	var googleBooksAPIURL = constructAPIURL(query, searchType, startIndex)

	fmt.Println("Calling google books api.....")
	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	return resp, err
}

func DecodeResponse(resp *http.Response) (map[string]interface{}, error) {
	fmt.Println("Decoding json response.....")

	var volumeData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&volumeData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	return volumeData, nil
}

func ExtractBookInfo(volumeData map[string]interface{}, searchType string, query string, isDetailed bool, db *sql.DB) ([]models.Book, error) {
	fmt.Println("Extracting book info.....")

	var books []models.Book

	if searchType == "id" {
		if volumeInfo, ok := volumeData["volumeInfo"].(map[string]interface{}); ok {
			book := CreateBook(volumeInfo, searchType, query, isDetailed, db)
			books = append(books, book)
		}
	} else {
		if items, ok := volumeData["items"].([]interface{}); ok {
			seenIDs := make(map[string]bool)
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					id := itemMap["id"].(string)
					if seenIDs[id] {
						continue
					}
					seenIDs[id] = true
					if volumeInfo, ok := itemMap["volumeInfo"].(map[string]interface{}); ok {
						book := CreateBook(volumeInfo, searchType, id, isDetailed, db)
						books = append(books, book)
					}
				}
			}
		}
	}
	return books, nil
}

func CreateBook(volumeInfo map[string]interface{}, searchType string, query string, isDetailed bool, db *sql.DB) models.Book {
	fmt.Println("Creating book objects.....")

	var thumbnail string

	// First try to get thumbnail from database
	dbThumbnail, err := database.GetBookThumbnail(db, query)
	if err != nil {
		log.Println("Error querying database for thumbnail:", err)
	} else if dbThumbnail != "" {
		thumbnail = dbThumbnail
		log.Println("Using database thumbnail:", thumbnail)
	}

	// If no database thumbnail, try to get from volumeInfo
	if thumbnail == "" {
		if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
			if _, ok := imageLinks["thumbnail"].(string); ok {
				thumbnail = fmt.Sprintf("https://books.google.com/books/publisher/content/images/frontcover/%s?fife=w600-h800&source=gbs_api", query)
				log.Println("Using volumeInfo thumbnail:", thumbnail)
			} else {
				log.Println("No thumbnail URL found in imageLinks")
			}
		} else {
			log.Println("No imageLinks found in volumeInfo")
		}
	}

	if thumbnail == "" {
		log.Println("No thumbnail available for book:", query)
	}

	book := models.Book{
		Title: volumeInfo["title"].(string),
		ID:    query,
		Cover: thumbnail,
	}

	if publishedDate, ok := volumeInfo["publishedDate"].(string); ok {
		book.PublishedDate = publishedDate
	}

	if authors, ok := volumeInfo["authors"].([]interface{}); ok {
		for _, author := range authors {
			if author == nil {
				book.Authors = append(book.Authors, "Author Unknown")
			} else {
				book.Authors = append(book.Authors, author.(string))
			}
		}
	} else {
		book.Authors = append(book.Authors, "Author Unknown")
	}

	if isDetailed {
		if description, ok := volumeInfo["description"].(string); ok {
			book.Description = template.HTML(description)
		}
	}

	return book
}
