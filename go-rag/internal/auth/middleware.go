package auth

import (
    "context"
    "net/http"
    "strings"

    "github.com/sirupsen/logrus"
)

type contextKey string

const UserIDKey contextKey = "userID"

// Middleware for Chi
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logrus.WithFields(logrus.Fields{
            "method": r.Method,
            "path":   r.URL.Path,
            "ip":     r.RemoteAddr,
        }).Debug("auth middleware processing request")
        
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            logrus.WithFields(logrus.Fields{
                "method": r.Method,
                "path":   r.URL.Path,
                "ip":     r.RemoteAddr,
            }).Warn("auth middleware: missing authorization header")
            http.Error(w, "missing token", http.StatusUnauthorized)
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := ValidateToken(tokenStr)
        if err != nil {
            logrus.WithFields(logrus.Fields{
                "method": r.Method,
                "path":   r.URL.Path,
                "ip":     r.RemoteAddr,
                "error":  err,
            }).Warn("auth middleware: token validation failed")
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        logrus.WithFields(logrus.Fields{
            "user_id": claims.UserID,
            "method":  r.Method,
            "path":    r.URL.Path,
            "ip":      r.RemoteAddr,
        }).Info("auth middleware: user authenticated successfully")
        
        ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetUserID(ctx context.Context) (int, bool) {
    userID, ok := ctx.Value(UserIDKey).(int)
    return userID, ok
}