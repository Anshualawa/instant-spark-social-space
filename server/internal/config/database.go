
package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB() {
	var err error
	
	// Configure MySQL connection parameters
	dbUsername := "root"     // Replace with your MySQL username
	dbPassword := ""     // Replace with your MySQL password
	dbHost := "localhost"
	dbPort := "3306"
	dbName := "chat_app"
	
	// Create MySQL connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUsername,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	// Open database connection
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	err = DB.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Successfully connected to MySQL database")

	// Create tables if they don't exist
	if err := createTables(); err != nil {
		log.Fatal("Failed to create tables:", err)
	}
}

// createTables creates all necessary database tables if they don't exist
func createTables() error {
	// Users table
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			avatar VARCHAR(255),
			is_online BOOLEAN DEFAULT false,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating users table: %v", err)
	}

	// Chats table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255),
			is_group BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating chats table: %v", err)
	}

	// Chat participants table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS chat_participants (
			chat_id VARCHAR(36),
			user_id VARCHAR(36),
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (chat_id, user_id),
			FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating chat_participants table: %v", err)
	}

	// Messages table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id VARCHAR(36) PRIMARY KEY,
			chat_id VARCHAR(36),
			sender_id VARCHAR(36),
			content TEXT NOT NULL,
			is_read BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating messages table: %v", err)
	}

	return nil
}
