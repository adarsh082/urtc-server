package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID `json:"id"`
	GITHUB_ID  int64     `json:"github_id"`
	USERNAME   string    `json:"username"`
	EMAIL      string    `json:"email"`
	CREATED_AT time.Time `json:"created_at"`
}

type UserModel struct {
	DB *sql.DB
}

// CreateUser - Creates a new user
func (m *UserModel) CreateUser(githubID int64, username, email string) (*User, error) {
	query := `
		INSERT INTO users (id, github_id, username, email, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, github_id, username, email, created_at
	`

	id := uuid.New()
	now := time.Now()

	var user User
	err := m.DB.QueryRow(query, id, githubID, username, email, now).Scan(
		&user.ID, &user.GITHUB_ID, &user.USERNAME, &user.EMAIL, &user.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUser - Gets user by username
func (m *UserModel) GetUser(username string) (*User, error) {
	query := `
		SELECT id, github_id, username, email, created_at
		FROM users
		WHERE username = $1
	`

	var user User
	err := m.DB.QueryRow(query, username).Scan(
		&user.ID, &user.GITHUB_ID, &user.USERNAME, &user.EMAIL, &user.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail - Gets user by email
func (m *UserModel) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, github_id, username, email, created_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := m.DB.QueryRow(query, email).Scan(
		&user.ID, &user.GITHUB_ID, &user.USERNAME, &user.EMAIL, &user.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID - Gets user by ID
func (m *UserModel) GetUserByID(userID uuid.UUID) (*User, error) {
	query := `
		SELECT id, github_id, username, email, created_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := m.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.GITHUB_ID, &user.USERNAME, &user.EMAIL, &user.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAllUsers - Gets all users
func (m *UserModel) GetAllUsers() ([]User, error) {
	query := `
		SELECT id, github_id, username, email, created_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.GITHUB_ID, &user.USERNAME, &user.EMAIL, &user.CREATED_AT,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// DeleteUser - Deletes a user
func (m *UserModel) DeleteUser(username string) error {
	query := `DELETE FROM users WHERE username = $1`
	_, err := m.DB.Exec(query, username)
	return err
}

// UpdateUser - Updates user information
func (m *UserModel) UpdateUser(userID uuid.UUID, username, email string) error {
	query := `
		UPDATE users
		SET username = $1, email = $2
		WHERE id = $3
	`
	_, err := m.DB.Exec(query, username, email, userID)
	return err
}
