package handlers

import (
	"encoding/json"
	"go-rag/ent/ent"
	"go-rag/internal/auth"
	"go-rag/internal/projects"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// ProjectHandler handles HTTP requests for projects.
type ProjectHandler struct {
	ProjectService *projects.Service
}

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// respondJSON is a helper to write JSON responses.
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

// respondError is a helper to write JSON error responses.
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{"error": message})
}

// CreateProject handles the POST /projects request.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	serviceReq := projects.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	p, err := h.ProjectService.CreateProject(r.Context(), serviceReq)
	if err != nil {
		logrus.WithError(err).Error("handler: failed to create project")
		respondError(w, http.StatusInternalServerError, "Failed to create project")
		return
	}

	respondJSON(w, http.StatusCreated, p)
}

// GetProject handles the GET /projects/{projectID} request.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	p, err := h.ProjectService.GetProjectByID(r.Context(), projectID, ownerID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Project not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to get project")
			respondError(w, http.StatusInternalServerError, "Failed to retrieve project")
		}
		return
	}

	respondJSON(w, http.StatusOK, p)
}

// ListProjects handles the GET /projects request.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	pList, err := h.ProjectService.ListProjectsByUser(r.Context(), ownerID)
	if err != nil {
		logrus.WithError(err).Error("handler: failed to list projects")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve projects")
		return
	}

	respondJSON(w, http.StatusOK, pList)
}

// UpdateProject handles the PUT /projects/{projectID} request.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	serviceReq := projects.UpdateProjectRequest{
		ProjectID:   projectID,
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	p, err := h.ProjectService.UpdateProject(r.Context(), serviceReq)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Project not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to update project")
			respondError(w, http.StatusInternalServerError, "Failed to update project")
		}
		return
	}

	respondJSON(w, http.StatusOK, p)
}

// DeleteProject handles the DELETE /projects/{projectID} request.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	projectID, err := strconv.Atoi(chi.URLParam(r, "projectID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	err = h.ProjectService.DeleteProject(r.Context(), projectID, ownerID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "Project not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to delete project")
			respondError(w, http.StatusInternalServerError, "Failed to delete project")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Project deleted successfully"})
}
