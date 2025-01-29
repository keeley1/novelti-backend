package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

type Book struct {
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	PublishedDate string   `json:"publishedDate"`
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", handler)
	router.HandleFunc("/books", searchBooks).Methods("GET")
	router.HandleFunc("/searchbysubject", searchBooksBySubject).Methods("GET")
	router.HandleFunc("/searchbytitle", searchBooksByTitle).Methods("GET")

	fmt.Println("Server starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello world!")
}

func searchBooks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	// encode query to handle spaces and special characters
	encodedQuery := url.QueryEscape(query)
	googleBooksAPIURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s", encodedQuery)

	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from Google Books API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	// print raw response body
	fmt.Println("Raw API response:", string(body))

	// parse json response
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse Google Books API response", http.StatusInternalServerError)
		return
	}

	items, ok := data["items"].([]interface{})
	if !ok {
		http.Error(w, "No items found in Google Books API response", http.StatusNotFound)
		return
	}

	var books []Book
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		volumeInfo, ok := itemMap["volumeInfo"].(map[string]interface{})
		if !ok {
			continue
		}

		book := Book{
			Title:         volumeInfo["title"].(string),
			PublishedDate: volumeInfo["publishedDate"].(string),
		}

		if authors, exists := volumeInfo["authors"].([]interface{}); exists {
			for _, author := range authors {
				book.Authors = append(book.Authors, author.(string))
			}
		}

		books = append(books, book)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func searchBooksBySubject(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("subject")
	if genre == "" {
		http.Error(w, "Missing query parameter 'subject'", http.StatusBadRequest)
		return
	}

	encodedSubject := url.QueryEscape(genre)
	googleBooksAPIURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=+subject:%s&maxResults=20", encodedSubject)

	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from Google Books API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func searchBooksByTitle(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		http.Error(w, "Missing query parameter 'title'", http.StatusBadRequest)
		return
	}

	encodedTitle := url.QueryEscape(title)

	googleBooksAPIURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=intitle:%s", encodedTitle)
	resp, err := http.Get(googleBooksAPIURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from Google Books API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
