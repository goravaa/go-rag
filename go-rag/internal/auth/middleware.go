package auth

import (
	"context"
	"go-rag/internal/db"      
	"go-rag/ent/ent/session"
	"net/http"
	"strings"
   "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type contextKey string

const UserIDKey contextKey = "userID"

// Middleware for Chi router
// This middleware now validates the JWT AND checks the database for session revocation.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithFields(logrus.Fields{
			"method": r.Method,
			"path":   r.URL.Path,
			"ip":     r.RemoteAddr,
		})
		log.Debug("auth middleware processing request")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateToken(tokenStr)
		if err != nil {
			log.WithError(err).Warn("auth middleware: token validation failed")
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		client := db.NewClient()
		if client == nil {
			log.Error("auth middleware: database client is not initialized")
			http.Error(w, "server configuration error", http.StatusInternalServerError)
			return
		}

		s, err := client.Session.
			Query().
			Where(session.AccessTokenEQ(tokenStr)).
			Only(r.Context())


		if err != nil {
			log.WithError(err).Warn("auth middleware: could not find session for token")
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}

		if s.RevokedAt != nil {
			log.Warn("auth middleware: attempt to use a revoked session")
			http.Error(w, "Please login again.", http.StatusUnauthorized)
			return
		}
		log.WithField("user_id", claims.UserID).Info("auth middleware: user authenticated successfully")
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}
