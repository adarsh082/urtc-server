package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID           uuid.UUID `json:"id"`
	GITHUB_TOKEN string    `json:"github_token"`
	USERNAME     string    `json:"username"`
	USER_ID      uuid.UUID `json:"user_id"`
	CREATED_AT   time.Time `json:"created_at"`
}

type TokenModel struct {
	DB *sql.DB
}

// SaveToken - Saves a new GitHub token
func (m *TokenModel) SaveToken(githubToken, username string, userID uuid.UUID) (*Token, error) {
	query := `
		INSERT INTO github_data (id, github_token, username, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, github_token, username, user_id, created_at
	`

	id := uuid.New()
	now := time.Now()

	var token Token
	err := m.DB.QueryRow(query, id, githubToken, username, userID, now).Scan(
		&token.ID, &token.GITHUB_TOKEN, &token.USERNAME, &token.USER_ID, &token.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

// GetToken - Gets GitHub token by username
func (m *TokenModel) GetToken(username string) (*Token, error) {
	query := `
		SELECT id, github_token, username, user_id, created_at
		FROM github_data
		WHERE username = $1
	`

	var token Token
	err := m.DB.QueryRow(query, username).Scan(
		&token.ID, &token.GITHUB_TOKEN, &token.USERNAME, &token.USER_ID, &token.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

// GetTokenByUserID - Gets GitHub token by user ID
func (m *TokenModel) GetTokenByUserID(userID uuid.UUID) (*Token, error) {
	query := `
		SELECT id, github_token, username, user_id, created_at
		FROM github_data
		WHERE user_id = $1
	`

	var token Token
	err := m.DB.QueryRow(query, userID).Scan(
		&token.ID, &token.GITHUB_TOKEN, &token.USERNAME, &token.USER_ID, &token.CREATED_AT,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

// UpdateToken - Updates an existing GitHub token
func (m *TokenModel) UpdateToken(username, githubToken string) error {
	query := `
		UPDATE github_data
		SET github_token = $1
		WHERE username = $2
	`

	result, err := m.DB.Exec(query, githubToken, username)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteToken - Deletes a GitHub token
func (m *TokenModel) DeleteToken(username string) error {
	query := `DELETE FROM github_data WHERE username = $1`
	_, err := m.DB.Exec(query, username)
	return err
}

// DeleteTokenByUserID - Deletes a GitHub token by user ID
func (m *TokenModel) DeleteTokenByUserID(userID uuid.UUID) error {
	query := `DELETE FROM github_data WHERE user_id = $1`
	_, err := m.DB.Exec(query, userID)
	return err
}

// TokenExists - Checks if a token exists for a username
func (m *TokenModel) TokenExists(username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM github_data WHERE username = $1)`

	var exists bool
	err := m.DB.QueryRow(query, username).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
