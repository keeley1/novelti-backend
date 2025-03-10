package database

import (
	"database/sql"
	"fmt"
	"log"
)

// temp function for testing
func AddBook(db *sql.DB, bookID string) error {
	// Prepare the SQL statement
	query := `INSERT INTO books (id) VALUES (?)`
	_, err := db.Exec(query, bookID)
	if err != nil {
		log.Println("Error inserting book:", err)
		return err
	}
	fmt.Println("Book added successfully!")
	return nil
}

func AddBookReview(db *sql.DB, bookID string, rating float64, user string) error {
	// Prepare the SQL statement
	query := `INSERT INTO reviews (book_id, rating, user) VALUES (?, ?, ?)`
	_, err := db.Exec(query, bookID, rating, user)
	if err != nil {
		log.Println("Error inserting review:", err)
		return err
	}
	fmt.Println("Review added successfully!")
	return nil
}
