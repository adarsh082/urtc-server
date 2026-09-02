package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          uuid.UUID `json:"id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RepoURL     string    `json:"repo_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProjectModel struct {
	DB *sql.DB
}

// CreateProject - Creates a new project
func (m *ProjectModel) CreateProject(ownerID uuid.UUID, name, description, repoURL string) (*Project, error) {
	query := `
		INSERT INTO projects (id, owner_id, name, description, repo_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, owner_id, name, description, repo_url, created_at
	`

	id := uuid.New()
	now := time.Now()

	var project Project
	err := m.DB.QueryRow(query, id, ownerID, name, description, repoURL, now).Scan(
		&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.RepoURL, &project.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectByID - Gets project by ID
func (m *ProjectModel) GetProjectByID(projectID uuid.UUID) (*Project, error) {
	query := `
		SELECT id, owner_id, name, description, repo_url, created_at
		FROM projects
		WHERE id = $1
	`

	var project Project
	err := m.DB.QueryRow(query, projectID).Scan(
		&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.RepoURL, &project.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectByName - Gets project by owner ID and name
func (m *ProjectModel) GetProjectByName(ownerID uuid.UUID, name string) (*Project, error) {
	query := `
		SELECT id, owner_id, name, description, repo_url, created_at
		FROM projects
		WHERE owner_id = $1 AND name = $2
	`

	var project Project
	err := m.DB.QueryRow(query, ownerID, name).Scan(
		&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.RepoURL, &project.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProjectsByUser - Gets all projects for a user
func (m *ProjectModel) GetProjectsByUser(ownerID uuid.UUID) ([]Project, error) {
	query := `
		SELECT id, owner_id, name, description, repo_url, created_at
		FROM projects
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.RepoURL, &project.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// DeleteProject - Deletes a project
func (m *ProjectModel) DeleteProject(ownerID, projectID uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1 AND owner_id = $2`
	result, err := m.DB.Exec(query, projectID, ownerID)
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

// UpdateProject - Updates project information
func (m *ProjectModel) UpdateProject(projectID uuid.UUID, name, description string) error {
	query := `
		UPDATE projects
		SET name = $1, description = $2
		WHERE id = $3
	`
	_, err := m.DB.Exec(query, name, description, projectID)
	return err
}

// GetAllProjects - Gets all projects (admin function)
func (m *ProjectModel) GetAllProjects() ([]Project, error) {
	query := `
		SELECT id, owner_id, name, description, repo_url, created_at
		FROM projects
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.RepoURL, &project.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}
