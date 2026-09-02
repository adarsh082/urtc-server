package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatal("Error pinging DB: ", err)
	}
	log.Println("Connected to DB")

	log.Println("Initializing User Table")
	InitUserTable()
	log.Println("Initialized User Table Successfully")

	log.Println("Initializing Project Table")
	InitProjectTable()
	log.Println("Initialized Project Table Successfully")

	log.Println("Initializing Token Table")
	InitTokenTable()
	log.Println("Initialized Token Table Successfully")

	log.Println("Initializing Collaborator Table")
	InitCollaboratorTable()
	log.Println("Initialized Collaborator Table Successfully")

	log.Println("Initializing Activity Table")
	err = InitActivityTable()
	if err != nil {
		log.Fatal("Failed to initialize Activity Table: ", err)
	}
	log.Println("Initialized Activity Table Successfully")

	log.Println("Initializing Version Control Tables")
	err = InitVersionControlTables()
	if err != nil {
		log.Fatal("Failed to initialize Version Control Tables: ", err)
	}
	log.Println("Initialized Version Control Tables Successfully")
}

func InitUserTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		github_id BIGINT UNIQUE NOT NULL,
		username TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create users table: ", err)
	}
}

func InitProjectTable() {
	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id UUID PRIMARY KEY,
		owner_id UUID REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT,
		repo_url TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create project table: ", err)
	}

	// Migration: add repo_url column if it was missing from an older schema
	_, err = DB.Exec(`ALTER TABLE projects ADD COLUMN IF NOT EXISTS repo_url TEXT`)
	if err != nil {
		log.Fatal("Failed to migrate projects table (add repo_url): ", err)
	}
}

func InitTokenTable() {
	query := `
	CREATE TABLE IF NOT EXISTS github_data (
		id UUID PRIMARY KEY,
		github_token TEXT NOT NULL,
		username TEXT UNIQUE NOT NULL,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create github_data table: ", err)
	}
}

func InitCollaboratorTable() {
	query := `
	CREATE TABLE IF NOT EXISTS collaborators (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id),
		project_id UUID REFERENCES projects(id),
		status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, project_id)
	)`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create collaborators table: ", err)
	}
}
