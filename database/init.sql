-- create books table
CREATE TABLE IF NOT EXISTS books (
    id VARCHAR(255) PRIMARY KEY,
    thumbnail TEXT
);

-- create reviews/ratings table
CREATE TABLE IF NOT EXISTS reviews (
    id INT AUTO_INCREMENT PRIMARY KEY,
    book_id VARCHAR(255),
    review_text TEXT,
    rating FLOAT CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user VARCHAR(255),
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);
