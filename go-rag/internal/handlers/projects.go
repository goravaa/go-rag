package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
	"go-rag/internal/ent"
	"go-rag/internal/ent/project"
	"go-rag/internal/ent/user"
	"go-rag/internal/auth"
)

type ProjectHandler struct {
	Client *ent.Client
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		logrus.WithField("ok", ok).Warn("unauthorized access to CreateProject")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if decodeErr := json.NewDecoder(r.Body).Decode(&input); decodeErr != nil {
		logrus.WithError(decodeErr).Warn("invalid input for CreateProject")
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	p, saveErr := h.Client.Project.
		Create().
		SetName(input.Name).
		SetDescription(input.Description).
		SetOwnerID(userID). // adjust to your schema field, maybe SetOwnerID
		Save(r.Context())
	if saveErr != nil {
		logrus.WithError(saveErr).Error("failed to create project")
		http.Error(w, "could not create project", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id":    userID,
		"project_id": p.ID,
	}).Info("Project created")

	json.NewEncoder(w).Encode(p)
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		logrus.WithField("ok", ok).Warn("unauthorized access to CreateProject")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projects, queryErr := h.Client.Project.
		Query().
		Where(project.HasOwnerWith(user.IDEQ(userID))).
		All(r.Context())
	if queryErr != nil {
		logrus.WithError(queryErr).Error("failed to list projects")
		http.Error(w, "could not fetch projects", http.StatusInternalServerError)
		return
	}

	logrus.WithField("user_id", userID).Info("Listed projects")
	json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, parseErr := strconv.Atoi(idStr)
	if parseErr != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	p, getErr := h.Client.Project.Get(r.Context(), id)
	if getErr != nil {
		logrus.WithError(getErr).Warn("project not found")
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	logrus.WithField("project_id", id).Info("Fetched project")
	json.NewEncoder(w).Encode(p)
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, parseErr := strconv.Atoi(idStr)
	if parseErr != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	delErr := h.Client.Project.DeleteOneID(id).Exec(r.Context())
	if delErr != nil {
		logrus.WithError(delErr).Error("failed to delete project")
		http.Error(w, "could not delete project", http.StatusInternalServerError)
		return
	}

	logrus.WithField("project_id", id).Info("Deleted project")
	w.WriteHeader(http.StatusNoContent)
}
