package main

import (
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/joho/godotenv"
    "github.com/sirupsen/logrus"

    "go-rag/internal/auth"
    "go-rag/internal/handlers"
    "go-rag/internal/user"
    "go-rag/internal/db"
)

func main() {
    // Configure logrus
    logrus.SetFormatter(&logrus.JSONFormatter{})
    logrus.SetLevel(logrus.InfoLevel)
    
    logrus.Info("starting server...")

    // Load .env file
    if err := godotenv.Load(); err != nil {
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
    authHandler := &handlers.AuthHandler{UserService: userService}
    logrus.Info("services initialized successfully")

    // setup router
    logrus.Debug("setting up HTTP router")
    r := chi.NewRouter()

    // Public routes
    r.Post("/signup", authHandler.Signup)
    r.Post("/login", authHandler.Login)
    logrus.Info("public routes registered", "routes", []string{"/signup", "/login"})

    // Protected group
    r.Group(func(protected chi.Router) {
        protected.Use(auth.AuthMiddleware)

        protected.Get("/me", func(w http.ResponseWriter, r *http.Request) {
            userID, _ := auth.GetUserID(r.Context())
            logrus.WithField("user_id", userID).Info("user accessed /me endpoint")
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            _, err := w.Write([]byte(fmt.Sprintf(`{"message":"Hello, user %d"}`, userID)))
            if err != nil {
                logrus.WithFields(logrus.Fields{
                    "user_id": userID,
                    "error": err,
                }).Error("error writing response for user")
            }
        })
        
        // User deletion endpoint
        protected.Delete("/user", authHandler.DeleteUser)
    })
    logrus.Info("protected routes registered", "routes", []string{"/me", "DELETE /user"})

    addr := ":8080"
    logrus.WithField("address", addr).Info("server starting")
    if err := http.ListenAndServe(addr, r); err != nil {
        logrus.WithError(err).Fatal("server failed to start")
    }
}
