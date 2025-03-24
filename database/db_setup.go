package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func InitialiseDB() (*sql.DB, error) {
	dsn := "host=localhost port=5433 user=postgres password=pass dbname=novelti_db sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connection to database failed: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	// Create tables if they do not exist
	err = createTables(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables")
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	// Define SQL statements for creating tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS books (
			id VARCHAR(255) PRIMARY KEY,
			thumbnail TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS reviews (
    		id SERIAL PRIMARY KEY,
    		book_id VARCHAR(255),
    		review_text TEXT,
    		rating FLOAT CHECK (rating BETWEEN 1 AND 5),
    		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    		user_name VARCHAR(255),
    		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
		);`,
	}

	// Execute each table creation statement
	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			return fmt.Errorf("error creating database tables: %w", err)
		}
	}
	// return nil if there is no error
	return nil
}
