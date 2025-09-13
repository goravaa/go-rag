package auth

import (
    "errors"
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/sirupsen/logrus"
)

var jwtSecret []byte

func LoadSecret() {
    logrus.Debug("loading JWT secret from environment")
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        logrus.Fatal("JWT_SECRET not set in environment")
    }
    jwtSecret = []byte(secret)
    logrus.Info("JWT secret loaded successfully")
}

type Claims struct {
    UserID int `json:"user_id"`
    jwt.RegisteredClaims
}

func GenerateToken(userID int) (string, error) {
    logrus.WithField("user_id", userID).Debug("generating JWT token")
    
    claims := Claims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString(jwtSecret)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": userID,
            "error": err,
        }).Error("failed to sign JWT token")
        return "", err
    }
    logrus.WithField("user_id", userID).Info("JWT token generated successfully")
    return signed, nil
}

func ValidateToken(tokenStr string) (*Claims, error) {
    logrus.Debug("validating JWT token")
    
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })
    if err != nil {
        logrus.WithError(err).Warn("failed to parse JWT token")
        return nil, errors.New("invalid token")
    }
    
    if !token.Valid {
        logrus.Warn("JWT token is not valid")
        return nil, errors.New("invalid token")
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok {
        logrus.Warn("failed to extract claims from JWT token")
        return nil, errors.New("invalid claims")
    }
    
    logrus.WithField("user_id", claims.UserID).Debug("JWT token validated successfully")
    return claims, nil
}
