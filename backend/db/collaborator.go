package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Collaborator struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ProjectID uuid.UUID `json:"project_id"`
	Status    string    `json:"status"` // "pending", "approved", "rejected"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CollaboratorModel struct {
	DB *sql.DB
}

// CreateCollaboration - Creates a new collaboration request
func (m *CollaboratorModel) CreateCollaboration(userID, projectID uuid.UUID, status string) (*Collaborator, error) {
	query := `
		INSERT INTO collaborators (id, user_id, project_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, project_id, status, created_at, updated_at
	`

	id := uuid.New()
	now := time.Now()

	var collab Collaborator
	err := m.DB.QueryRow(query, id, userID, projectID, status, now, now).Scan(
		&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &collab, nil
}

// GetCollaborationByID - Gets a collaboration by ID
func (m *CollaboratorModel) GetCollaborationByID(collabID uuid.UUID) (*Collaborator, error) {
	query := `
		SELECT id, user_id, project_id, status, created_at, updated_at
		FROM collaborators
		WHERE id = $1
	`

	var collab Collaborator
	err := m.DB.QueryRow(query, collabID).Scan(
		&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &collab, nil
}

// GetCollaborationByUserAndProject - Gets collaboration by user and project
func (m *CollaboratorModel) GetCollaborationByUserAndProject(userID, projectID uuid.UUID) (*Collaborator, error) {
	query := `
		SELECT id, user_id, project_id, status, created_at, updated_at
		FROM collaborators
		WHERE user_id = $1 AND project_id = $2
	`

	var collab Collaborator
	err := m.DB.QueryRow(query, userID, projectID).Scan(
		&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &collab, nil
}

// GetProjectCollaborators - Gets all collaborators for a project
func (m *CollaboratorModel) GetProjectCollaborators(projectID uuid.UUID) ([]Collaborator, error) {
	query := `
		SELECT id, user_id, project_id, status, created_at, updated_at
		FROM collaborators
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collaborators []Collaborator
	for rows.Next() {
		var collab Collaborator
		err := rows.Scan(
			&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		collaborators = append(collaborators, collab)
	}

	return collaborators, nil
}

// GetUserPendingRequests - Gets pending collaboration requests for a user
func (m *CollaboratorModel) GetUserPendingRequests(userID uuid.UUID) ([]Collaborator, error) {
	query := `
		SELECT id, user_id, project_id, status, created_at, updated_at
		FROM collaborators
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []Collaborator
	for rows.Next() {
		var collab Collaborator
		err := rows.Scan(
			&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, collab)
	}

	return requests, nil
}

// GetUserCollaborations - Gets all collaborations for a user (any status)
func (m *CollaboratorModel) GetUserCollaborations(userID uuid.UUID) ([]Collaborator, error) {
	query := `
		SELECT id, user_id, project_id, status, created_at, updated_at
		FROM collaborators
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collaborations []Collaborator
	for rows.Next() {
		var collab Collaborator
		err := rows.Scan(
			&collab.ID, &collab.UserID, &collab.ProjectID, &collab.Status, &collab.CreatedAt, &collab.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		collaborations = append(collaborations, collab)
	}

	return collaborations, nil
}

// UpdateCollaborationStatus - Updates the status of a collaboration
func (m *CollaboratorModel) UpdateCollaborationStatus(collabID uuid.UUID, status string) error {
	query := `
		UPDATE collaborators
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := m.DB.Exec(query, status, now, collabID)
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

// DeleteCollaboration - Deletes a collaboration
func (m *CollaboratorModel) DeleteCollaboration(collabID uuid.UUID) error {
	query := `DELETE FROM collaborators WHERE id = $1`
	result, err := m.DB.Exec(query, collabID)
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

// UpdateCollaborationUserID - Remaps a collaboration record from a placeholder user to the real user.
// This is needed when the owner added a collaborator by email before that collaborator had a real
// GitHub account in the system. On first join the placeholder ID gets replaced by the real user ID.
func (m *CollaboratorModel) UpdateCollaborationUserID(collabID, newUserID uuid.UUID) error {
	query := `
		UPDATE collaborators
		SET user_id = $1, updated_at = $2
		WHERE id = $3
	`
	now := time.Now()
	result, err := m.DB.Exec(query, newUserID, now, collabID)
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

// IsUserCollaborator - Checks if a user is an approved collaborator on a project
func (m *CollaboratorModel) IsUserCollaborator(userID, projectID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM collaborators 
			WHERE user_id = $1 AND project_id = $2 AND status = 'approved'
		)
	`

	var exists bool
	err := m.DB.QueryRow(query, userID, projectID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
