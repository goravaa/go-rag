package handlers

import (
	"context"
	"encoding/json"
	"go-rag/internal/auth"
	"go-rag/internal/user"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type addSecurityQuestionRequest struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type forgotPasswordResponse struct {
	QuestionID string `json:"question_id"`
	Question   string `json:"question"`
}

type resetPasswordRequest struct {
	QuestionID  string `json:"question_id"`
	Answer      string `json:"answer"`
	NewPassword string `json:"new_password"`
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
			"error": err,
		}).Warn("signup: invalid request body")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	u, err := h.UserService.CreateUser(context.Background(), req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid email") {
			http.Error(w, "invalid email format", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "email already exists") {
			http.Error(w, "email already exists", http.StatusConflict)
		} else {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
		}
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id": u.ID,
		"email":   u.Email,
	}).Info("signup: user created successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "user created successfully"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"ip":     r.RemoteAddr,
	}).Info("login request received")

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	loginReq := user.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: r.RemoteAddr,
		UserAgent: r.Header.Get("User-Agent"),
	}

	session, err := h.UserService.LoginUser(r.Context(), loginReq)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
		} else {
			logrus.WithFields(logrus.Fields{
				"email": req.Email,
				"error": err,
			}).Error("login: an internal error occurred")
			http.Error(w, "an internal error occurred", http.StatusInternalServerError)
		}
		return
	}

	response := loginResponse{
		AccessToken:  session.AccessToken,
		RefreshToken: *session.RefreshToken,
	}

	logrus.WithField("email", req.Email).Info("login: user authenticated successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
		"ip":     r.RemoteAddr,
	}).Info("delete user request received")

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.UserService.DeleteUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	logrus.WithField("user_id", userID).Info("deleteUser: user deleted successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "user deleted successfully"})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	session, err := h.UserService.RefreshSession(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	response := loginResponse{
		AccessToken:  session.AccessToken,
		RefreshToken: *session.RefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	err := h.UserService.LogoutUser(r.Context(), tokenStr)
	if err != nil {
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out successfully"})
}

func (h *AuthHandler) AddSecurityQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req addSecurityQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	serviceReq := user.AddSecurityQuestionRequest{
		UserID:   userID,
		Question: req.Question,
		Answer:   req.Answer,
	}

	_, err := h.UserService.AddSecurityQuestion(r.Context(), serviceReq)
	if err != nil {
		http.Error(w, "failed to add security question", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "security question added successfully"})
}

func (h *AuthHandler) ForgotPasswordRequest(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	sq, err := h.UserService.GetRandomSecurityQuestionForUser(r.Context(), req.Email)
	if err != nil {
		if strings.Contains(err.Error(), "no security questions found for this user") {
			http.Error(w, "You have not added any security questions for password recovery.", http.StatusBadRequest)
		} else {
			http.Error(w, "Could not process request. Please check the email and try again.", http.StatusNotFound)
		}
		return
	}

	response := forgotPasswordResponse{
		QuestionID: sq.ID.String(),
		Question:   sq.Question,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	qID, err := uuid.Parse(req.QuestionID)
	if err != nil {
		http.Error(w, "invalid question ID format", http.StatusBadRequest)
		return
	}

	serviceReq := user.ResetPasswordWithSecurityQuestionRequest{
		QuestionID:     qID,
		ProvidedAnswer: req.Answer,
		NewPassword:    req.NewPassword,
	}

	err = h.UserService.ResetPasswordWithSecurityQuestion(r.Context(), serviceReq)
	if err != nil {
		if strings.Contains(err.Error(), "invalid question or answer") {
			http.Error(w, "Incorrect Answer.", http.StatusUnauthorized)
		} else {
			http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password has been reset successfully. Please log in."})
}
