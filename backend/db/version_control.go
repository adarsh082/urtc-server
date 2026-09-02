package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type FileVersion struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	UserID    uuid.UUID
	FilePath  string
	FileName  string
	FileType  string
	Version   int
	FileHash  string
	FileSize  int64
	Content   string
	CommitMsg string
	IsDeleted bool
	CreatedAt time.Time
}

type FileConflict struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	FilePath      string
	BaseVersion   int
	LocalUserID   uuid.UUID
	RemoteUserID  uuid.UUID
	LocalContent  string
	RemoteContent string
	Status        string
	ResolvedBy    uuid.UUID
	CreatedAt     time.Time
	ResolvedAt    time.Time
}

type VersionModel struct {
	DB *sql.DB
}

type ConflictModel struct {
	DB *sql.DB
}

// CreateVersion - Creates a new file version
func (m *VersionModel) CreateVersion(projectID, userID uuid.UUID, filePath, fileName, fileType, fileHash string, fileSize int64, content, commitMsg string) (*FileVersion, error) {
	// Get next version number
	var version int
	versionQuery := `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM file_versions
		WHERE project_id = $1 AND file_path = $2
	`
	err := m.DB.QueryRow(versionQuery, projectID, filePath).Scan(&version)
	if err != nil {
		version = 1
	}

	query := `
		INSERT INTO file_versions (id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, content, commit_message, is_deleted, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, content, commit_message, is_deleted, created_at
	`

	id := uuid.New()
	now := time.Now()

	var fv FileVersion
	err = m.DB.QueryRow(
		query,
		id, projectID, userID, filePath, fileName, fileType, version,
		fileHash, fileSize, content, commitMsg, false, now,
	).Scan(
		&fv.ID, &fv.ProjectID, &fv.UserID, &fv.FilePath, &fv.FileName,
		&fv.FileType, &fv.Version, &fv.FileHash, &fv.FileSize, &fv.Content,
		&fv.CommitMsg, &fv.IsDeleted, &fv.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fv, nil
}

// GetLatestVersion - Gets the latest version of a file
func (m *VersionModel) GetLatestVersion(projectID uuid.UUID, filePath string) (*FileVersion, error) {
	query := `
		SELECT id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, content, commit_message, is_deleted, created_at
		FROM file_versions
		WHERE project_id = $1 AND file_path = $2 AND is_deleted = false
		ORDER BY version DESC
		LIMIT 1
	`

	var fv FileVersion
	err := m.DB.QueryRow(query, projectID, filePath).Scan(
		&fv.ID, &fv.ProjectID, &fv.UserID, &fv.FilePath, &fv.FileName,
		&fv.FileType, &fv.Version, &fv.FileHash, &fv.FileSize, &fv.Content,
		&fv.CommitMsg, &fv.IsDeleted, &fv.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fv, nil
}

// GetFileHistory - Gets version history for a file
func (m *VersionModel) GetFileHistory(projectID uuid.UUID, filePath string, limit int) ([]FileVersion, error) {
	query := `
		SELECT id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, commit_message, is_deleted, created_at
		FROM file_versions
		WHERE project_id = $1 AND file_path = $2
		ORDER BY version DESC
		LIMIT $3
	`

	rows, err := m.DB.Query(query, projectID, filePath, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []FileVersion
	for rows.Next() {
		var fv FileVersion
		err := rows.Scan(
			&fv.ID, &fv.ProjectID, &fv.UserID, &fv.FilePath, &fv.FileName,
			&fv.FileType, &fv.Version, &fv.FileHash, &fv.FileSize,
			&fv.CommitMsg, &fv.IsDeleted, &fv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, fv)
	}

	return versions, nil
}

// GetProjectVersions - Gets all recent versions in a project
func (m *VersionModel) GetProjectVersions(projectID uuid.UUID, limit int) ([]FileVersion, error) {
	query := `
		SELECT id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, commit_message, is_deleted, created_at
		FROM file_versions
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := m.DB.Query(query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []FileVersion
	for rows.Next() {
		var fv FileVersion
		err := rows.Scan(
			&fv.ID, &fv.ProjectID, &fv.UserID, &fv.FilePath, &fv.FileName,
			&fv.FileType, &fv.Version, &fv.FileHash, &fv.FileSize,
			&fv.CommitMsg, &fv.IsDeleted, &fv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, fv)
	}

	return versions, nil
}

// GetVersionByID - Gets a specific version by ID
func (m *VersionModel) GetVersionByID(versionID uuid.UUID) (*FileVersion, error) {
	query := `
		SELECT id, project_id, user_id, file_path, file_name, file_type, version, file_hash, file_size, content, commit_message, is_deleted, created_at
		FROM file_versions
		WHERE id = $1
	`

	var fv FileVersion
	err := m.DB.QueryRow(query, versionID).Scan(
		&fv.ID, &fv.ProjectID, &fv.UserID, &fv.FilePath, &fv.FileName,
		&fv.FileType, &fv.Version, &fv.FileHash, &fv.FileSize, &fv.Content,
		&fv.CommitMsg, &fv.IsDeleted, &fv.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fv, nil
}

// Conflict Model Methods

// CreateConflict - Creates a new conflict record
func (m *ConflictModel) CreateConflict(projectID uuid.UUID, filePath string, baseVersion int, localUserID, remoteUserID uuid.UUID, localContent, remoteContent string) (*FileConflict, error) {
	query := `
		INSERT INTO file_conflicts (id, project_id, file_path, base_version, local_user_id, remote_user_id, local_content, remote_content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, project_id, file_path, base_version, local_user_id, remote_user_id, status, created_at
	`

	id := uuid.New()
	now := time.Now()

	var fc FileConflict
	err := m.DB.QueryRow(
		query,
		id, projectID, filePath, baseVersion, localUserID, remoteUserID,
		localContent, remoteContent, "pending", now,
	).Scan(
		&fc.ID, &fc.ProjectID, &fc.FilePath, &fc.BaseVersion,
		&fc.LocalUserID, &fc.RemoteUserID, &fc.Status, &fc.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &fc, nil
}

// GetProjectConflicts - Gets all conflicts for a project
func (m *ConflictModel) GetProjectConflicts(projectID uuid.UUID, status string) ([]FileConflict, error) {
	query := `
		SELECT id, project_id, file_path, base_version, local_user_id, remote_user_id, status, created_at
		FROM file_conflicts
		WHERE project_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := m.DB.Query(query, projectID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []FileConflict
	for rows.Next() {
		var fc FileConflict
		err := rows.Scan(
			&fc.ID, &fc.ProjectID, &fc.FilePath, &fc.BaseVersion,
			&fc.LocalUserID, &fc.RemoteUserID, &fc.Status, &fc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		conflicts = append(conflicts, fc)
	}

	return conflicts, nil
}

// ResolveConflict - Marks a conflict as resolved
func (m *ConflictModel) ResolveConflict(conflictID, resolvedBy uuid.UUID) error {
	query := `
		UPDATE file_conflicts
		SET status = 'resolved', resolved_by = $1, resolved_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := m.DB.Exec(query, resolvedBy, now, conflictID)
	return err
}

// Initialize tables

func InitVersionControlTables() error {
	versionTable := `
	CREATE TABLE IF NOT EXISTS file_versions (
		id UUID PRIMARY KEY,
		project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		file_path TEXT NOT NULL,
		file_name TEXT NOT NULL,
		file_type TEXT NOT NULL,
		version INTEGER NOT NULL,
		file_hash TEXT NOT NULL,
		file_size BIGINT NOT NULL,
		content TEXT,
		commit_message TEXT,
		is_deleted BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(project_id, file_path, version)
	);

	CREATE INDEX IF NOT EXISTS idx_file_versions_project ON file_versions(project_id);
	CREATE INDEX IF NOT EXISTS idx_file_versions_file_path ON file_versions(file_path);
	CREATE INDEX IF NOT EXISTS idx_file_versions_created_at ON file_versions(created_at DESC);
	`

	conflictTable := `
	CREATE TABLE IF NOT EXISTS file_conflicts (
		id UUID PRIMARY KEY,
		project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
		file_path TEXT NOT NULL,
		base_version INTEGER NOT NULL,
		local_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		remote_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		local_content TEXT NOT NULL,
		remote_content TEXT NOT NULL,
		status TEXT NOT NULL CHECK (status IN ('pending', 'resolved', 'ignored')),
		resolved_by UUID REFERENCES users(id),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_file_conflicts_project ON file_conflicts(project_id);
	CREATE INDEX IF NOT EXISTS idx_file_conflicts_status ON file_conflicts(status);
	`

	_, err := DB.Exec(versionTable)
	if err != nil {
		return err
	}

	_, err = DB.Exec(conflictTable)
	return err
}
