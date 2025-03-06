package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/keeley1/novelti-backend/models"
)

func ConstructAPIURL(query string, searchType string) string {
	var googleBooksAPIURL string

	if searchType == "id" {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/%s", query)
	} else if searchType == "searchquery" {
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20", query)
	} else {
		encodedQuery := url.QueryEscape(query + " books")
		googleBooksAPIURL = fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=20", encodedQuery)
	}

	return googleBooksAPIURL
}

func MakeAPICall(apiURL string) (*http.Response, error) {
	fmt.Println("Calling google books api.....")

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch data: %v", err)
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

func ExtractBookInfo(volumeData map[string]interface{}, searchType string, query string, isDetailed bool) ([]models.Book, error) {
	fmt.Println("Extracting book info.....")

	var books []models.Book

	if searchType == "id" {
		if volumeInfo, ok := volumeData["volumeInfo"].(map[string]interface{}); ok {
			book := CreateBook(volumeInfo, searchType, query, isDetailed)
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
						book := CreateBook(volumeInfo, searchType, id, isDetailed)
						books = append(books, book)
					}
				}
			}
		}
	}
	return books, nil
}

func CreateBook(volumeInfo map[string]interface{}, searchType string, query string, isDetailed bool) models.Book {
	fmt.Println("Creating book objects.....")

	var thumbnail string
	if imageLinks, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
		if _, ok := imageLinks["thumbnail"].(string); ok {
			bookID := query
			thumbnail = fmt.Sprintf("https://books.google.com/books/publisher/content/images/frontcover/%s?fife=w600-h800&source=gbs_api", bookID)
		}
	}
	if thumbnail == "" {
		fmt.Println("No thumbnail available")
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
			book.Authors = append(book.Authors, author.(string))
		}
	}

	if isDetailed {
		if description, ok := volumeInfo["description"].(string); ok {
			book.Description = template.HTML(description)
		}
	}

	return book
}
