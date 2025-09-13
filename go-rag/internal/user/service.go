package user

import (
    "context"
    "fmt"
    "net/mail"

    "golang.org/x/crypto/bcrypt"
    "github.com/sirupsen/logrus"

    "go-rag/internal/ent"
    "go-rag/internal/ent/user"
)

type Service struct {
    Client *ent.Client
}

func isValidEmail(e string) bool {
    _, err := mail.ParseAddress(e)
    return err == nil
}

func (s *Service) CreateUser(ctx context.Context, email, password string) (*ent.User, error) {
    logrus.WithField("email", email).Debug("creating new user")
    
    if !isValidEmail(email) {
        logrus.WithField("email", email).Warn("createUser: invalid email format")
        return nil, fmt.Errorf("invalid email")
    }

    _, err := s.GetUserByEmail(ctx, email)
    if err == nil {
        logrus.WithField("email", email).Warn("createUser: email already exists")
        return nil, fmt.Errorf("email already exists")
    }
    if !ent.IsNotFound(err) {
        logrus.WithFields(logrus.Fields{
            "email": email,
            "error": err,
        }).Error("createUser: DB error when checking existing email")
        return nil, err
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "email": email,
            "error": err,
        }).Error("createUser: failed to hash password")
        return nil, err
    }

    u, err := s.Client.User.
        Create().
        SetEmail(email).
        SetPasswordHash(string(hashedPassword)).
        Save(ctx)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "email": email,
            "error": err,
        }).Error("createUser: failed to save user to database")
        return nil, err
    }

    logrus.WithFields(logrus.Fields{
        "user_id": u.ID,
        "email":   u.Email,
    }).Info("createUser: user created successfully")
    return u, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*ent.User, error) {
    logrus.WithField("email", email).Debug("looking up user by email")
    
    u, err := s.Client.User.
        Query().
        Where(user.EmailEQ(email)).
        Only(ctx)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "email": email,
            "error": err,
        }).Warn("getUserByEmail: user not found")
        return nil, err
    }
    
    logrus.WithFields(logrus.Fields{
        "user_id": u.ID,
        "email":   email,
    }).Debug("getUserByEmail: user found")
    return u, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID int) error {
    logrus.WithField("user_id", userID).Debug("deleting user")
    
    err := s.Client.User.DeleteOneID(userID).Exec(ctx)
    if err != nil {
        logrus.WithFields(logrus.Fields{
            "user_id": userID,
            "error":   err,
        }).Error("deleteUser: failed to delete user from database")
        return err
    }
    
    logrus.WithField("user_id", userID).Info("deleteUser: user deleted successfully")
    return nil
}
