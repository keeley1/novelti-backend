package database

import (
	"database/sql"
	"fmt"
	"log"
)

// temp function for testing
func AddBook(db *sql.DB, bookID string) error {
	// Prepare the SQL statement
	query := `INSERT INTO books (id) VALUES ($1)`
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
	query := `INSERT INTO reviews (book_id, rating, user_name) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, bookID, rating, user)
	if err != nil {
		log.Println("Error inserting review:", err)
		return err
	}
	fmt.Println("Review added successfully!")
	return nil
}

func InsertBooks(db *sql.DB, bookID string, thumbnail string) error {
	// Prepare the SQL statement
	query := `INSERT INTO books (id, thumbnail) VALUES ($1, $2)`
	_, err := db.Exec(query, bookID, thumbnail)
	if err != nil {
		log.Println("Error inserting book with thumbnail:", err)
		return fmt.Errorf("failed to insert book with thumbnail: %w", err)
	}
	log.Println("Book with thumbnail added successfully!")
	return nil
}

func GetBookThumbnail(db *sql.DB, bookID string) (string, error) {
	// Prepare the SQL statement
	query := `SELECT thumbnail FROM books WHERE id = $1`
	var thumbnail string

	err := db.QueryRow(query, bookID).Scan(&thumbnail)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no thumbnail found for book ID: %s", bookID)
		}
		log.Println("Error querying book thumbnail:", err)
		return "", fmt.Errorf("failed to get book thumbnail: %w", err)
	}

	return thumbnail, nil
}
