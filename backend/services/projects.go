package services

import (
	"app/urtc/db"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func NProjects(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		fmt.Println("Invalid Username")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username"})
		return
	} else {
		owner_id := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		projects, err := projectModel.GetProjectsByUser(owner_id)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch projects"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"owner":    username,
			"count":    len(projects),
			"projects": projects,
		})
	}
}

func GetProjects(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		fmt.Println("Invalid Username")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username"})
		return
	} else {
		owner_id := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		projects, err := projectModel.GetProjectsByUser(owner_id)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch projects"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}
}
func GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	nameStr := vars["name"]

	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid owner ID"})
		return
	} else {
		ownerID := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		project, err := projectModel.GetProjectByName(ownerID, nameStr)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Project not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["owner"]
	projectName := vars["name"]

	userModel := &db.UserModel{
		DB: db.DB,
	}
	user, err := userModel.GetUser(username)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)

		return
	} else {
		ownerID := user.ID
		projectModel := &db.ProjectModel{
			DB: db.DB,
		}
		project, err := projectModel.GetProjectByName(ownerID, projectName)
		if err != nil {
			fmt.Println("Error : ", err)
			w.WriteHeader(http.StatusBadRequest)

		}
		projectID := project.ID
		err = projectModel.DeleteProject(ownerID, projectID)
		if err != nil {
			fmt.Println("Error : ", err)
		}
	}
}
