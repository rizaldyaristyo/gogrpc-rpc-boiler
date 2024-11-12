package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Database connections
var (
	BookDB     *sql.DB
	AuthorDB   *sql.DB
	CategoryDB *sql.DB
	UserDB     *sql.DB
)

const (
	maxRetries = 5           // max retries
	retryDelay = 5 * time.Second // 5sec
)

// dsn generator
func createDSN(dbNameEnvVar string) string {
	// debug
	fmt.Println("DB_USER:", os.Getenv("DB_USER"))
	fmt.Println("DB_PASSWORD:", os.Getenv("DB_PASSWORD"))
	fmt.Println("DB_HOST:", os.Getenv("DB_HOST"))
	fmt.Println("DB_PORT:", os.Getenv("DB_PORT"))
	fmt.Println("DB_NAME:", os.Getenv(dbNameEnvVar))
	fmt.Println("DB_SSLMODE:", os.Getenv("DB_SSLMODE"))
	
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv(dbNameEnvVar)
	sslMode := os.Getenv("DB_SSLMODE")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)
}

func connectWithRetry(dsn string, dbName string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("Attempt %d: Failed to connect to %s database: %v", i+1, dbName, err)
		} else if err = db.Ping(); err != nil {
			log.Printf("Attempt %d: Failed to ping %s database: %v", i+1, dbName, err)
		} else {
			log.Printf("Connected to %s database", dbName)
			return db, nil
		}

		// Wait before retrying
		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("failed to connect to %s database after %d attempts: %v", dbName, maxRetries, err)
}

func ConnectBookDB() {
	dsn := createDSN("DB_NAME_BOOK")
	var err error
	BookDB, err = connectWithRetry(dsn, "Book Service")
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func ConnectAuthorDB() {
	dsn := createDSN("DB_NAME_AUTHOR")
	var err error
	AuthorDB, err = connectWithRetry(dsn, "Author Service")
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func ConnectCategoryDB() {
	dsn := createDSN("DB_NAME_CATEGORY")
	var err error
	CategoryDB, err = connectWithRetry(dsn, "Category Service")
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func ConnectUserDB() {
	dsn := createDSN("DB_NAME_USER")
	var err error
	UserDB, err = connectWithRetry(dsn, "User Service")
	if err != nil {
		log.Fatalf("%v", err)
	}
}
