//go:build patchdb
// +build patchdb

package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using system environment variables")
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	DB, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer DB.Close()

	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging DB: ", err)
	}

	query := `ALTER TABLE projects ADD COLUMN IF NOT EXISTS repo_url TEXT;`
	_, err = DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to alter projects table: ", err)
	}

	log.Println("Successfully added 'repo_url' column to 'projects' table.")
}
