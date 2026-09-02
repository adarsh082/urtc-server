package services

import (
	"app/urtc/db"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type FileVersion struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	FilePath  string    `json:"file_path"`
	FileName  string    `json:"file_name"`
	FileType  string    `json:"file_type"`
	Version   int       `json:"version"`
	FileHash  string    `json:"file_hash"`
	FileSize  int64     `json:"file_size"`
	Content   string    `json:"content,omitempty"` // Base64 or text content
	CommitMsg string    `json:"commit_message"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username,omitempty"`
}

type VersionHistoryRequest struct {
	ProjectID string `json:"project_id"`
	FilePath  string `json:"file_path"`
	Limit     int    `json:"limit"`
}

type FileConflict struct {
	ID            uuid.UUID `json:"id"`
	ProjectID     uuid.UUID `json:"project_id"`
	FilePath      string    `json:"file_path"`
	BaseVersion   int       `json:"base_version"`
	LocalUserID   uuid.UUID `json:"local_user_id"`
	RemoteUserID  uuid.UUID `json:"remote_user_id"`
	LocalContent  string    `json:"local_content"`
	RemoteContent string    `json:"remote_content"`
	Status        string    `json:"status"` // "pending", "resolved", "ignored"
	ResolvedBy    uuid.UUID `json:"resolved_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ResolvedAt    time.Time `json:"resolved_at,omitempty"`
}

type CommitRequest struct {
	ProjectID   string `json:"project_id"`
	UserEmail   string `json:"user_email"`
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	Content     string `json:"content"`
	FileHash    string `json:"file_hash"`
	FileSize    int64  `json:"file_size"`
	CommitMsg   string `json:"commit_message"`
	BaseVersion int    `json:"base_version,omitempty"`
}

// CommitFileVersion - Creates a new version of a file
func CommitFileVersion(w http.ResponseWriter, r *http.Request) {
	var req CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.ProjectID == "" || req.UserEmail == "" || req.FilePath == "" || req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Missing required fields",
		})
		return
	}

	projectUUID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	// Get user
	userModel := &db.UserModel{DB: db.DB}
	user, err := userModel.GetUserByEmail(req.UserEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	// Check for conflicts
	versionModel := &db.VersionModel{DB: db.DB}
	latestVersion, err := versionModel.GetLatestVersion(projectUUID, req.FilePath)

	hasConflict := false
	if err == nil && req.BaseVersion > 0 && latestVersion.Version > req.BaseVersion {
		// Conflict detected - someone else committed while user was editing
		hasConflict = true

		// Create conflict record
		conflictModel := &db.ConflictModel{DB: db.DB}
		conflict, _ := conflictModel.CreateConflict(
			projectUUID,
			req.FilePath,
			req.BaseVersion,
			user.ID,
			latestVersion.UserID,
			req.Content,
			latestVersion.Content,
		)

		// Notify remote user about conflict
		SendNotificationToUser(
			latestVersion.UserID.String(),
			"file_conflict",
			"File conflict detected",
			map[string]interface{}{
				"conflict_id": conflict.ID,
				"project_id":  req.ProjectID,
				"file_path":   req.FilePath,
				"user_email":  req.UserEmail,
			},
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        false,
			"conflict":       true,
			"conflict_id":    conflict.ID,
			"message":        "Conflict detected. Please resolve before committing.",
			"base_version":   req.BaseVersion,
			"latest_version": latestVersion.Version,
		})
		return
	}

	// Create new version
	version, err := versionModel.CreateVersion(
		projectUUID,
		user.ID,
		req.FilePath,
		req.FileName,
		req.FileType,
		req.FileHash,
		req.FileSize,
		req.Content,
		req.CommitMsg,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to create version: " + err.Error(),
		})
		return
	}

	// Log activity
	LogActivity(
		user.ID,
		projectUUID,
		"file_commit",
		"Committed "+req.FilePath,
		map[string]interface{}{
			"file_path": req.FilePath,
			"version":   version.Version,
			"file_hash": req.FileHash,
		},
		r,
	)

	// Notify collaborators
	collabModel := &db.CollaboratorModel{DB: db.DB}
	collaborators, _ := collabModel.GetProjectCollaborators(projectUUID)

	for _, collab := range collaborators {
		if collab.Status == "approved" && collab.UserID != user.ID {
			SendNotificationToUser(
				collab.UserID.String(),
				"file_updated",
				user.USERNAME+" committed "+req.FilePath,
				map[string]interface{}{
					"project_id": req.ProjectID,
					"file_path":  req.FilePath,
					"version":    version.Version,
					"user_email": req.UserEmail,
				},
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"version_id":   version.ID,
		"version":      version.Version,
		"file_path":    req.FilePath,
		"commit_msg":   req.CommitMsg,
		"has_conflict": hasConflict,
	})
}

// GetFileHistory - Retrieves version history for a file
func GetFileHistory(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	filePath := r.URL.Query().Get("file_path")

	if projectID == "" || filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_id and file_path are required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	versionModel := &db.VersionModel{DB: db.DB}
	versions, err := versionModel.GetFileHistory(projectUUID, filePath, 50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch file history",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"project_id": projectID,
		"file_path":  filePath,
		"versions":   versions,
		"total":      len(versions),
	})
}

// GetProjectVersions - Get all recent versions in a project
func GetProjectVersions(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_id is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	versionModel := &db.VersionModel{DB: db.DB}
	versions, err := versionModel.GetProjectVersions(projectUUID, 100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch project versions",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"project_id": projectID,
		"versions":   versions,
		"total":      len(versions),
	})
}

// GetFileConflicts - Get all pending conflicts for a project
func GetFileConflicts(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "project_id is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid project ID",
		})
		return
	}

	conflictModel := &db.ConflictModel{DB: db.DB}
	conflicts, err := conflictModel.GetProjectConflicts(projectUUID, "pending")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch conflicts",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"project_id": projectID,
		"conflicts":  conflicts,
		"total":      len(conflicts),
	})
}

// ResolveConflict - Mark a conflict as resolved
func ResolveConflict(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConflictID      string `json:"conflict_id"`
		ResolvedByEmail string `json:"resolved_by_email"`
		ResolvedContent string `json:"resolved_content"`
		CommitMsg       string `json:"commit_message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	conflictUUID, err := uuid.Parse(req.ConflictID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid conflict ID",
		})
		return
	}

	userModel := &db.UserModel{DB: db.DB}
	user, err := userModel.GetUserByEmail(req.ResolvedByEmail)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "User not found",
		})
		return
	}

	conflictModel := &db.ConflictModel{DB: db.DB}
	err = conflictModel.ResolveConflict(conflictUUID, user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to resolve conflict",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"conflict_id": req.ConflictID,
		"message":     "Conflict resolved successfully",
	})
}
