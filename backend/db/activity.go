package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Activity struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	ProjectID   uuid.UUID              `json:"project_id,omitempty"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type ActivityModel struct {
	DB *sql.DB
}

// CreateActivity - Logs a new activity
func (m *ActivityModel) CreateActivity(userID, projectID uuid.UUID, action, description string, metadata map[string]interface{}, ipAddress, userAgent string) error {
	query := `
		INSERT INTO activities (id, user_id, project_id, action, description, metadata, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New()
	now := time.Now()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = m.DB.Exec(query, id, userID, projectID, action, description, metadataJSON, ipAddress, userAgent, now)
	return err
}

// GetUserActivities - Gets recent activities for a user
func (m *ActivityModel) GetUserActivities(userID uuid.UUID, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, project_id, action, description, metadata, ip_address, user_agent, created_at
		FROM activities
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := m.DB.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanActivities(rows)
}

// GetProjectActivities - Gets recent activities for a project
func (m *ActivityModel) GetProjectActivities(projectID uuid.UUID, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, project_id, action, description, metadata, ip_address, user_agent, created_at
		FROM activities
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := m.DB.Query(query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanActivities(rows)
}

// GetActivitiesByAction - Gets activities filtered by action type
func (m *ActivityModel) GetActivitiesByAction(action string, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, project_id, action, description, metadata, ip_address, user_agent, created_at
		FROM activities
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := m.DB.Query(query, action, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanActivities(rows)
}

// GetActivitiesByDateRange - Gets activities within a date range
func (m *ActivityModel) GetActivitiesByDateRange(startDate, endDate time.Time, limit int) ([]Activity, error) {
	query := `
		SELECT id, user_id, project_id, action, description, metadata, ip_address, user_agent, created_at
		FROM activities
		WHERE created_at BETWEEN $1 AND $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := m.DB.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanActivities(rows)
}

// Helper function to scan activity rows
func (m *ActivityModel) scanActivities(rows *sql.Rows) ([]Activity, error) {
	var activities []Activity

	for rows.Next() {
		var activity Activity
		var metadataJSON []byte
		var projectID sql.NullString

		err := rows.Scan(
			&activity.ID,
			&activity.UserID,
			&projectID,
			&activity.Action,
			&activity.Description,
			&metadataJSON,
			&activity.IPAddress,
			&activity.UserAgent,
			&activity.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse project ID if not null
		if projectID.Valid {
			activity.ProjectID, _ = uuid.Parse(projectID.String)
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &activity.Metadata)
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// InitActivityTable - Creates the activities table
func InitActivityTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS activities (
		id UUID PRIMARY KEY,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
		action TEXT NOT NULL,
		description TEXT NOT NULL,
		metadata JSONB,
		ip_address TEXT,
		user_agent TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_activities_user_id ON activities(user_id);
	CREATE INDEX IF NOT EXISTS idx_activities_project_id ON activities(project_id);
	CREATE INDEX IF NOT EXISTS idx_activities_action ON activities(action);
	CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);
	`

	_, err := DB.Exec(query)
	return err
}
