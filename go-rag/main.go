package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"go-rag/internal/auth"
	"go-rag/internal/db"
	"go-rag/internal/documents" // Import the new documents package
	"go-rag/internal/handlers"
	"go-rag/internal/projects"
	"go-rag/internal/user"
)

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Info("starting server...")

	// Load .env file
	if err := godotenv.Load(".env.dev"); err != nil {
		logrus.Warn("no .env file found, using system environment variables")
	} else {
		logrus.Info(".env file loaded successfully")
	}

	// Load JWT secret
	auth.LoadSecret()

	// setup DB client
	logrus.Debug("initializing database client")
	client := db.NewClient()
	defer func() {
		logrus.Debug("closing database client")
		if err := client.Close(); err != nil {
			logrus.WithError(err).Error("error closing DB client")
		} else {
			logrus.Debug("DB client closed successfully")
		}
	}()

	// setup services
	logrus.Debug("initializing services")
	userService := &user.Service{Client: client}
	projectService := &projects.Service{Client: client}
	documentService := &documents.Service{Client: client} // Initialize Document Service
	authHandler := &handlers.AuthHandler{UserService: userService}
	projectHandler := &handlers.ProjectHandler{ProjectService: projectService}
	documentHandler := &handlers.DocumentHandler{DocumentService: documentService} // Initialize Document Handler
	logrus.Info("services initialized successfully")

	logrus.Debug("setting up HTTP router")
	r := chi.NewRouter()

	// --- Public Routes ---
	r.Post("/signup", authHandler.Signup)
	r.Post("/login", authHandler.Login)
	r.Post("/auth/refreshAccessToken", authHandler.RefreshToken)

	// Password Recovery Routes
	r.Post("/auth/forgot-password/request", authHandler.ForgotPasswordRequest)
	r.Post("/auth/forgot-password/reset", authHandler.ResetPassword)
	logrus.Info("public routes registered")

	// --- Protected Routes ---
	r.Group(func(protected chi.Router) {
		protected.Use(auth.AuthMiddleware)

		// User and Auth Routes
		protected.Post("/logout", authHandler.Logout)
		protected.Delete("/user", authHandler.DeleteUser)
		protected.Post("/user/security-questions", authHandler.AddSecurityQuestion)

		// Project and Document Routes
		protected.Route("/projects", func(r chi.Router) {
			// Routes for the collection of projects
			r.Post("/", projectHandler.CreateProject)
			r.Get("/", projectHandler.ListProjects)

			// Routes for a specific project
			r.Route("/{projectID}", func(r chi.Router) {
				r.Get("/", projectHandler.GetProject)
				r.Put("/", projectHandler.UpdateProject)
				r.Delete("/", projectHandler.DeleteProject)

				// Nested Document Routes for the specific project
				r.Route("/documents", func(r chi.Router) {
					r.Post("/", documentHandler.CreateDocument)
					r.Get("/", documentHandler.ListDocuments)

					// Routes for a specific document
					r.Route("/{documentID}", func(r chi.Router) {
						r.Get("/", documentHandler.GetDocument)
						r.Delete("/", documentHandler.DeleteDocument)
						r.Put("/", documentHandler.UpdateDocument)
					})
				})
			})
		})

		protected.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.GetUserID(r.Context())
			if !ok {
				http.Error(w, "Unauthorized: User ID not found in context", http.StatusUnauthorized)
				return
			}

			user, err := userService.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}

			logrus.WithField("user_id", userID).Info("user accessed /me endpoint")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			responseJSON := fmt.Sprintf(`{"message":"Hello, %s"}`, user.Email)

			_, err = w.Write([]byte(responseJSON))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"user_id": userID,
					"error":   err,
				}).Error("error writing response for /me endpoint")
			}
		})
	})
	logrus.Info("protected routes registered")

	addr := ":8080"
	logrus.WithField("address", addr).Info("server starting")
	if err := http.ListenAndServe(addr, r); err != nil {
		logrus.WithError(err).Fatal("server failed to start")
	}
}
