package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "golang.org/x/crypto/bcrypt"
    "github.com/sirupsen/logrus"

    "go-rag/internal/auth"
    "go-rag/internal/user"
)

type AuthHandler struct {
    UserService *user.Service
}

type signupRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type loginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}



func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
    logrus.WithFields(logrus.Fields{
        "method": r.Method,
        "path":   r.URL.Path,
        "ip":     r.RemoteAddr,
    }).Info("signup request received")
    
    var req signupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        logrus.WithFields(logrus.Fields{
            "error":  err,
            "method": r.Method,
            "path":   r.URL.Path,
            "ip":     r.RemoteAddr,
        }).Warn("signup: invalid request body")
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    logrus.WithField("email", req.Email).Debug("processing signup request")
    
    user, err := h.UserService.CreateUser(context.Background(), req.Email, req.Password)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "email": req.Email,
            "error": err,
        }).Error("signup: failed to create user")
        
        if strings.Contains(err.Error(), "invalid email") {
            logrus.WithField("email", req.Email).Warn("signup: invalid email format")
            http.Error(w, "invalid email format", http.StatusBadRequest)
            return
        }
        if strings.Contains(err.Error(), "email already exists") {
            logrus.WithField("email", req.Email).Warn("signup: email already exists")
            http.Error(w, "email already exists", http.StatusConflict)
            return
        }
        http.Error(w, "failed to create user", http.StatusInternalServerError)
        return
    }

    token, err := auth.GenerateToken(user.ID)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": user.ID,
            "email":   req.Email,
            "error":   err,
        }).Error("signup: failed to generate token")
        http.Error(w, "failed to generate token", http.StatusInternalServerError)
        return
    }

    logrus.WithFields(logrus.Fields{
        "user_id": user.ID,
        "email":   user.Email,
    }).Info("signup: user created successfully")
    
    json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    logrus.WithFields(logrus.Fields{
        "method": r.Method,
        "path":   r.URL.Path,
        "ip":     r.RemoteAddr,
    }).Info("login request received")
    
    var req loginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        logrus.WithFields(logrus.Fields{
            "error":  err,
            "method": r.Method,
            "path":   r.URL.Path,
            "ip":     r.RemoteAddr,
        }).Warn("login: invalid request body")
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    logrus.WithField("email", req.Email).Debug("processing login request")
    
    user, err := h.UserService.GetUserByEmail(context.Background(), req.Email)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "email": req.Email,
            "error": err,
        }).Warn("login: user not found")
        http.Error(w, "user not found", http.StatusUnauthorized)
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": user.ID,
            "email":   req.Email,
            "error":   err,
        }).Warn("login: invalid password")
        http.Error(w, "invalid password", http.StatusUnauthorized)
        return
    }

    token, err := auth.GenerateToken(user.ID)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": user.ID,
            "email":   req.Email,
            "error":   err,
        }).Error("login: failed to generate token")
        http.Error(w, "failed to generate token", http.StatusInternalServerError)
        return
    }

    logrus.WithFields(logrus.Fields{
        "user_id": user.ID,
        "email":   user.Email,
    }).Info("login: user authenticated successfully")
    
    json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// DeleteUser handles user deletion
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    logrus.WithFields(logrus.Fields{
        "method": r.Method,
        "path":   r.URL.Path,
        "ip":     r.RemoteAddr,
    }).Info("delete user request received")
    
    // Get user ID from context (set by auth middleware)
    userID, ok := auth.GetUserID(r.Context())
    if !ok {
        logrus.WithFields(logrus.Fields{
            "method": r.Method,
            "path":   r.URL.Path,
            "ip":     r.RemoteAddr,
        }).Error("deleteUser: user ID not found in context")
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    
    logrus.WithField("user_id", userID).Debug("processing delete user request")
    
    err := h.UserService.DeleteUser(r.Context(), userID)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": userID,
            "error":   err,
        }).Error("deleteUser: failed to delete user")
        http.Error(w, "failed to delete user", http.StatusInternalServerError)
        return
    }
    
    logrus.WithField("user_id", userID).Info("deleteUser: user deleted successfully")
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "user deleted successfully"})
}
