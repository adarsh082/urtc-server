package services

import (
	"app/urtc/db"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type CollaborationRequest struct {
	ProjectName        string `json:"project_name"`
	UserEmail          string `json:"user_email"`
	ProjectDescription string `json:"project_description"`
	Token              string `json:"token"`
}

type CollaborationResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ProjectID   string `json:"project_id,omitempty"`
	CollabID    string `json:"collab_id,omitempty"`
	RepoURL     string `json:"repo_url,omitempty"`
	Token       string `json:"token,omitempty"`
	GitHubToken string `json:"github_token,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	Username    string `json:"username,omitempty"`
}

func PushProject(w http.ResponseWriter, r *http.Request) {
	var req CollaborationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePluginError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.UserEmail = strings.TrimSpace(req.UserEmail)
	req.ProjectName = strings.TrimSpace(req.ProjectName)
	req.ProjectDescription = strings.TrimSpace(req.ProjectDescription)

	if req.UserEmail == "" || req.ProjectName == "" {
		writePluginError(w, http.StatusBadRequest, "user_email and project_name are required")
		return
	}

	userModel := &db.UserModel{DB: db.DB}
	user, err := userModel.GetUserByEmail(req.UserEmail)
	if err != nil {
		writePluginError(w, http.StatusBadRequest, "User not found. Please login with GitHub first.")
		return
	}

	projectModel := &db.ProjectModel{DB: db.DB}
	tokenModel := &db.TokenModel{DB: db.DB}

	token, err := tokenModel.GetToken(user.USERNAME)
	if err != nil {
		writePluginError(w, http.StatusUnauthorized, "GitHub token not found. Please login with GitHub first.")
		return
	}

	if existingProject, err := projectModel.GetProjectByName(user.ID, req.ProjectName); err == nil {
		writePluginJSON(w, http.StatusOK, CollaborationResponse{
			Success:     true,
			Message:     "Collaboration already exists for this project",
			ProjectID:   existingProject.ID.String(),
			RepoURL:     existingProject.RepoURL,
			GitHubToken: token.GITHUB_TOKEN,
			UserID:      user.ID.String(),
			Username:    user.USERNAME,
		})
		return
	} else if err != sql.ErrNoRows {
		writePluginError(w, http.StatusInternalServerError, "Failed to check existing project")
		return
	}

	payload := map[string]interface{}{
		"name":    req.ProjectName,
		"private": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to prepare GitHub repository request")
		return
	}

	githubReq, err := http.NewRequest("POST", "https://api.github.com/user/repos", bytes.NewBuffer(body))
	if err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to create GitHub request")
		return
	}
	githubReq.Header.Set("Authorization", "token "+token.GITHUB_TOKEN)
	githubReq.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := (&http.Client{}).Do(githubReq)
	if err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to create GitHub repository")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var githubErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&githubErr)
		if githubErr.Message == "" {
			githubErr.Message = "GitHub repository creation failed"
		}
		writePluginError(w, http.StatusBadGateway, githubErr.Message)
		return
	}

	var repo struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to parse GitHub response")
		return
	}

	project, err := projectModel.CreateProject(user.ID, req.ProjectName, req.ProjectDescription, repo.HTMLURL)
	if err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to save project to database: "+err.Error())
		return
	}

	writePluginJSON(w, http.StatusOK, CollaborationResponse{
		Success:     true,
		Message:     "Collaboration started successfully",
		ProjectID:   project.ID.String(),
		RepoURL:     project.RepoURL,
		GitHubToken: token.GITHUB_TOKEN,
		UserID:      user.ID.String(),
		Username:    user.USERNAME,
	})
}

func JoinCollaboration(w http.ResponseWriter, r *http.Request) {
	var req CollaborationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writePluginError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.UserEmail = strings.TrimSpace(req.UserEmail)
	req.Token = strings.TrimSpace(req.Token)
	if req.UserEmail == "" || req.Token == "" {
		writePluginError(w, http.StatusBadRequest, "user_email and token are required")
		return
	}

	collabID, err := uuid.Parse(req.Token)
	if err != nil {
		writePluginError(w, http.StatusBadRequest, "Invalid token format")
		return
	}

	collabModel := &db.CollaboratorModel{DB: db.DB}
	collab, err := collabModel.GetCollaborationByID(collabID)
	if err != nil {
		writePluginError(w, http.StatusNotFound, "Invalid join token")
		return
	}

	userModel := &db.UserModel{DB: db.DB}
	user, err := userModel.GetUserByEmail(req.UserEmail)
	if err != nil {
		writePluginError(w, http.StatusBadRequest, "User not found. Please login with GitHub first.")
		return
	}

	// Check if the user IDs match directly.
	if collab.UserID != user.ID {
		// They differ — this happens when the owner added the collaborator by email before they
		// logged in with GitHub, creating a placeholder user. Now the real GitHub user is joining.
		// Verify by comparing the emails of the collab's placeholder user and the request email.
		placeholderUser, placeholderErr := userModel.GetUserByID(collab.UserID)
		if placeholderErr != nil {
			// Cannot verify — refuse access.
			writePluginError(w, http.StatusForbidden, "This join token does not belong to the provided user_email")
			return
		}

		if !strings.EqualFold(placeholderUser.EMAIL, user.EMAIL) {
			// Emails do not match — this token is for a different user.
			writePluginError(w, http.StatusForbidden, "This join token does not belong to the provided user_email")
			return
		}

		// Emails match — remap the collaboration from the placeholder to the real user.
		fmt.Printf("[JoinCollaboration] Remapping collab %s from placeholder user %s to real user %s\n",
			collabID, placeholderUser.ID, user.ID)
		if remapErr := collabModel.UpdateCollaborationUserID(collabID, user.ID); remapErr != nil {
			writePluginError(w, http.StatusInternalServerError, "Failed to update collaboration: "+remapErr.Error())
			return
		}
		// Refresh collab so it has the updated user_id for the rest of the function.
		collab.UserID = user.ID
	}

	if collab.Status != "approved" {
		if err := collabModel.UpdateCollaborationStatus(collabID, "approved"); err != nil {
			writePluginError(w, http.StatusInternalServerError, "Failed to approve collaboration: "+err.Error())
			return
		}
	}

	projectModel := &db.ProjectModel{DB: db.DB}
	project, err := projectModel.GetProjectByID(collab.ProjectID)
	if err != nil {
		writePluginError(w, http.StatusInternalServerError, "Failed to fetch project details")
		return
	}

	tokenModel := &db.TokenModel{DB: db.DB}
	token, err := tokenModel.GetToken(user.USERNAME)
	if err != nil {
		writePluginError(w, http.StatusUnauthorized, "User token not found. Please re-authenticate.")
		return
	}

	writePluginJSON(w, http.StatusOK, CollaborationResponse{
		Success:     true,
		Message:     "Joined collaboration successfully",
		ProjectID:   project.ID.String(),
		CollabID:    collab.ID.String(),
		RepoURL:     project.RepoURL,
		Token:       collab.ID.String(),
		GitHubToken: token.GITHUB_TOKEN,
		UserID:      user.ID.String(),
		Username:    user.USERNAME,
	})
}

func writePluginError(w http.ResponseWriter, status int, message string) {
	writePluginJSON(w, status, CollaborationResponse{
		Success: false,
		Message: message,
	})
}

func writePluginJSON(w http.ResponseWriter, status int, payload CollaborationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
