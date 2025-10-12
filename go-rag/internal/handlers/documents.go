package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go-rag/internal/auth"
	"go-rag/internal/documents"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// DocumentHandler handles HTTP requests for documents.
type DocumentHandler struct {
	DocumentService *documents.Service
}

type createDocumentRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func getContentHash(content []byte) string {
	hashBytes := sha256.Sum256(content)
	return fmt.Sprintf("%x", hashBytes)
}

type updateDocumentRequest struct {
	Name    *string `json:"name"`
	Content *string `json:"content"`
}

// CreateDocument handles POST /projects/{projectID}/documents
func (h *DocumentHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
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

	var req createDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.Name == "" || req.Content == "" {
		respondError(w, http.StatusBadRequest, "Fields 'name' and 'content' are required")
		return
	}

	// Calculate the hash from the content provided by the user.
	contentBytes := []byte(req.Content)
	hash := getContentHash(contentBytes)

	// Call the service with the content AND the new hash.
	serviceReq := documents.CreateDocumentRequest{
		Name:        req.Name,
		Content:     req.Content,
		ContentHash: hash,
		ProjectID:   projectID,
		OwnerID:     ownerID,
	}

	doc, err := h.DocumentService.CreateDocument(r.Context(), serviceReq)
	if err != nil {
		if strings.Contains(err.Error(), "project not found or access denied") {
			respondError(w, http.StatusNotFound, "Project not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to create document")
			respondError(w, http.StatusInternalServerError, "Failed to create document")
		}
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

// ListDocuments handles GET /projects/{projectID}/documents
func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
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

	docList, err := h.DocumentService.ListDocumentsByProject(r.Context(), projectID, ownerID)
	if err != nil {
		logrus.WithError(err).Error("handler: failed to list documents")
		respondError(w, http.StatusInternalServerError, "Failed to retrieve documents")
		return
	}

	respondJSON(w, http.StatusOK, docList)
}

// GetDocument handles GET /projects/{projectID}/documents/{documentID}
func (h *DocumentHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	documentID, err := strconv.Atoi(chi.URLParam(r, "documentID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	doc, err := h.DocumentService.GetDocumentByID(r.Context(), documentID, ownerID)
	if err != nil {
		if strings.Contains(err.Error(), "document not found or access denied") {
			respondError(w, http.StatusNotFound, "Document not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to get document")
			respondError(w, http.StatusInternalServerError, "Failed to retrieve document")
		}
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// UpdateDocument handles PUT /projects/{projectID}/documents/{documentID}
func (h *DocumentHandler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	documentID, err := strconv.Atoi(chi.URLParam(r, "documentID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	var req updateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// At least one field must be provided for an update.
	if req.Name == nil && req.Content == nil {
		respondError(w, http.StatusBadRequest, "At least one field ('name' or 'content') must be provided for an update")
		return
	}

	serviceReq := documents.UpdateDocumentRequest{
		DocumentID: documentID,
		OwnerID:    ownerID,
		Name:       req.Name,
	}

	// If content is being updated, we must re-calculate the hash.
	if req.Content != nil {
		contentBytes := []byte(*req.Content)
		hash := getContentHash(contentBytes)
		serviceReq.Content = req.Content
		serviceReq.ContentHash = &hash
	}

	doc, err := h.DocumentService.UpdateDocument(r.Context(), serviceReq)
	if err != nil {
		if strings.Contains(err.Error(), "document not found or access denied") {
			respondError(w, http.StatusNotFound, "Document not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to update document")
			respondError(w, http.StatusInternalServerError, "Failed to update document")
		}
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// DeleteDocument handles DELETE /projects/{projectID}/documents/{documentID}
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := auth.GetUserID(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	documentID, err := strconv.Atoi(chi.URLParam(r, "documentID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid document ID")
		return
	}

	err = h.DocumentService.DeleteDocument(r.Context(), documentID, ownerID)
	if err != nil {
		if strings.Contains(err.Error(), "document not found or access denied") {
			respondError(w, http.StatusNotFound, "Document not found or access denied")
		} else {
			logrus.WithError(err).Error("handler: failed to delete document")
			respondError(w, http.StatusInternalServerError, "Failed to delete document")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Document deleted successfully"})
}
