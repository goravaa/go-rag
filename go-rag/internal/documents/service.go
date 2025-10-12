package documents

import (
	"context"
	"fmt"
	"go-rag/ent/ent"
	"go-rag/ent/ent/document"
	"go-rag/ent/ent/project"
	"go-rag/ent/ent/user"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service handles the business logic for documents.
type Service struct {
	Client *ent.Client
}

// CreateDocumentRequest defines the parameters for creating a new document.
type CreateDocumentRequest struct {
	Name        string
	Content     string
	ContentHash string
	ProjectID   int
	OwnerID     uuid.UUID
}

type UpdateDocumentRequest struct {
	DocumentID  int
	OwnerID     uuid.UUID
	Name        *string
	Content     *string
	ContentHash *string
}

// CreateDocument creates a new document and associates it with a project.
// It first verifies that the user owns the project before creating the document.
func (s *Service) CreateDocument(ctx context.Context, req CreateDocumentRequest) (*ent.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"project_id":    req.ProjectID,
		"owner_id":      req.OwnerID,
		"document_name": req.Name,
	})
	log.Info("service: creating new document")

	// Security Check: Ensure the user owns the project.
	p, err := s.Client.Project.
		Query().
		Where(
			project.ID(req.ProjectID),
			project.HasOwnerWith(user.ID(req.OwnerID)),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			log.Warn("service: attempt to create document in a non-existent or unowned project")
			return nil, fmt.Errorf("project not found or access denied")
		}
		log.WithError(err).Error("service: failed to verify project ownership")
		return nil, err
	}

	doc, err := s.Client.Document.
		Create().
		SetName(req.Name).
		SetContent(req.Content).
		SetContentHash(req.ContentHash).
		SetProject(p).
		Save(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to save document to database")
		return nil, fmt.Errorf("could not create document: %w", err)
	}

	log.WithField("document_id", doc.ID).Info("service: document created successfully")
	return doc, nil
}

// ListDocumentsByProject retrieves all documents for a specific project, verifying ownership.
func (s *Service) ListDocumentsByProject(ctx context.Context, projectID int, ownerID uuid.UUID) ([]*ent.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"project_id": projectID,
		"owner_id":   ownerID,
	})
	log.Info("service: listing documents for project")

	// The query ensures we only get projects owned by the user, then gets their documents.
	docs, err := s.Client.Project.
		Query().
		Where(
			project.ID(projectID),
			project.HasOwnerWith(user.ID(ownerID)),
		).
		QueryDocuments().
		All(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to list documents from database")
		return nil, err
	}

	log.WithField("count", len(docs)).Info("service: documents listed successfully")
	return docs, nil
}

// GetDocumentByID retrieves a single document, ensuring it belongs to a project owned by the user.
func (s *Service) GetDocumentByID(ctx context.Context, documentID int, ownerID uuid.UUID) (*ent.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"document_id": documentID,
		"owner_id":    ownerID,
	})
	log.Info("service: getting document by id")

	// This query traverses from Document -> Project -> Owner to verify access.
	doc, err := s.Client.Document.
		Query().
		Where(
			document.ID(documentID),
			document.HasProjectWith(
				project.HasOwnerWith(user.ID(ownerID)),
			),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			log.Warn("service: document not found or access denied")
			return nil, fmt.Errorf("document not found or access denied")
		}
		log.WithError(err).Error("service: database error while getting document")
		return nil, err
	}

	log.Info("service: document retrieved successfully")
	return doc, nil
}

func (s *Service) UpdateDocument(ctx context.Context, req UpdateDocumentRequest) (*ent.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"document_id": req.DocumentID,
		"owner_id":    req.OwnerID,
	})
	log.Info("service: updating document")

	// First, get the document while verifying ownership.
	doc, err := s.GetDocumentByID(ctx, req.DocumentID, req.OwnerID)
	if err != nil {
		return nil, err
	}

	// Prepare the update operation.
	updater := doc.Update()

	// Conditionally add fields to the update if they were provided.
	if req.Name != nil {
		updater.SetName(*req.Name)
	}
	if req.Content != nil {
		updater.SetContent(*req.Content)
		updater.SetContentHash(*req.ContentHash) // Also update the hash
	}

	// Save the changes.
	updatedDoc, err := updater.Save(ctx)
	if err != nil {
		log.WithError(err).Error("service: failed to update document in database")
		return nil, err
	}

	log.Info("service: document updated successfully")
	return updatedDoc, nil
}

// DeleteDocument deletes a document, ensuring the user has ownership via the parent project.
func (s *Service) DeleteDocument(ctx context.Context, documentID int, ownerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"document_id": documentID,
		"owner_id":    ownerID,
	})
	log.Info("service: deleting document")

	// The delete operation is filtered by document ID and ownership of the parent project.
	n, err := s.Client.Document.
		Delete().
		Where(
			document.ID(documentID),
			document.HasProjectWith(
				project.HasOwnerWith(user.ID(ownerID)),
			),
		).
		Exec(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to delete document from database")
		return err
	}
	if n == 0 {
		log.Warn("service: document not found or access denied for deletion")
		return fmt.Errorf("document not found or access denied")
	}

	log.Info("service: document deleted successfully")
	return nil
}
