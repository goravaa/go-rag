package projects
package projects

import (
	"context"
	"fmt"
	"go-rag/ent/ent"
	"go-rag/ent/ent/project"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service handles the business logic for projects.
type Service struct {
	Client *ent.Client
}

// CreateProjectRequest defines the parameters for creating a new project.
type CreateProjectRequest struct {
	Name        string
	Description *string
	OwnerID     uuid.UUID
}

// UpdateProjectRequest defines the parameters for updating an existing project.
type UpdateProjectRequest struct {
	ProjectID   int
	Name        *string
	Description *string
	OwnerID     uuid.UUID // To verify ownership
}

// CreateProject creates a new project for a given user.
func (s *Service) CreateProject(ctx context.Context, req CreateProjectRequest) (*ent.Project, error) {
	log := logrus.WithFields(logrus.Fields{
		"owner_id": req.OwnerID,
		"name":     req.Name,
	})
	log.Info("service: creating new project")

	// The `AddOwnerID` method links the project to the user (owner).
	p, err := s.Client.Project.
		Create().
		SetName(req.Name).
		SetNillableDescription(req.Description).
		SetOwnerID(req.OwnerID).
		Save(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to create project in database")
		return nil, fmt.Errorf("could not create project: %w", err)
	}

	log.WithField("project_id", p.ID).Info("service: project created successfully")
	return p, nil
}

// GetProjectByID retrieves a single project by its ID, ensuring the requester is the owner.
func (s *Service) GetProjectByID(ctx context.Context, projectID int, ownerID uuid.UUID) (*ent.Project, error) {
	log := logrus.WithFields(logrus.Fields{
		"project_id": projectID,
		"owner_id":   ownerID,
	})
	log.Info("service: getting project by id")

	p, err := s.Client.Project.
		Query().
		Where(
			project.ID(projectID),
			project.HasOwnerWith(user.ID(ownerID)), // Security check
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			log.Warn("service: project not found or access denied")
		} else {
			log.WithError(err).Error("service: database error while getting project")
		}
		return nil, err
	}

	log.Info("service: project retrieved successfully")
	return p, nil
}

// ListProjectsByUser retrieves all projects for a specific user.
func (s *Service) ListProjectsByUser(ctx context.Context, ownerID uuid.UUID) ([]*ent.Project, error) {
	log := logrus.WithField("owner_id", ownerID)
	log.Info("service: listing projects for user")

	projects, err := s.Client.User.
		GetX(ctx, ownerID). // Get the user by ID
		QueryProjects().    // Query their projects
		All(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to list projects from database")
		return nil, err
	}

	log.WithField("count", len(projects)).Info("service: projects listed successfully")
	return projects, nil
}

// UpdateProject updates an existing project's details, ensuring the requester is the owner.
func (s *Service) UpdateProject(ctx context.Context, req UpdateProjectRequest) (*ent.Project, error) {
	log := logrus.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"owner_id":   req.OwnerID,
	})
	log.Info("service: updating project")

	// First, verify ownership and get the project.
	p, err := s.GetProjectByID(ctx, req.ProjectID, req.OwnerID)
	if err != nil {
		return nil, err // GetProjectByID already logs the error
	}

	updater := p.Update()
	if req.Name != nil {
		updater.SetName(*req.Name)
	}
	if req.Description != nil {
		updater.SetDescription(*req.Description)
	}

	updatedProject, err := updater.Save(ctx)
	if err != nil {
		log.WithError(err).Error("service: failed to update project in database")
		return nil, err
	}

	log.Info("service: project updated successfully")
	return updatedProject, nil
}

// DeleteProject deletes a project, ensuring the requester is the owner.
func (s *Service) DeleteProject(ctx context.Context, projectID int, ownerID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"project_id": projectID,
		"owner_id":   ownerID,
	})
	log.Info("service: deleting project")

	// The delete operation is filtered by both project ID and owner ID for security.
	n, err := s.Client.Project.
		Delete().
		Where(
			project.ID(projectID),
			project.HasOwnerWith(user.ID(ownerID)),
		).
		Exec(ctx)

	if err != nil {
		log.WithError(err).Error("service: failed to delete project from database")
		return err
	}
	if n == 0 {
		log.Warn("service: project not found or access denied for deletion")
		return ent.NewNotFoundError("project not found or access denied")
	}

	log.Info("service: project deleted successfully")
	return nil
}