package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func InitialiseDB() (*sql.DB, error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	// create db
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS novelti_db")
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// use db
	_, err = db.Exec("USE novelti_db")
	if err != nil {
		return nil, fmt.Errorf("failed to use database: %w", err)
	}

	// create tables
	createTables(db)

	fmt.Println("Database setup completed successfully!")
	return db, nil
}

func createTables(db *sql.DB) {
	// define SQL statements for creating tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS books (
			id VARCHAR(255) PRIMARY KEY,
			thumbnail TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS reviews (
    		id INT AUTO_INCREMENT PRIMARY KEY,
    		book_id VARCHAR(255),
    		review_text TEXT,
    		rating FLOAT CHECK (rating BETWEEN 1 AND 5),
    		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    		user VARCHAR(255),
    		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
		);`,
	}

	// execute each table creation statement
	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			log.Fatal("Failed to create table:", err)
		}
	}
}
