package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"go-rag/ent/ent"
	"go-rag/ent/ent/securityquestion"
	"go-rag/ent/ent/session"
	"go-rag/ent/ent/user"
	"go-rag/internal/auth"
	"math/big"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Client *ent.Client
}

type LoginRequest struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type AddSecurityQuestionRequest struct {
	UserID   uuid.UUID
	Question string
	Answer   string
}

type ResetPasswordWithSecurityQuestionRequest struct {
	QuestionID     uuid.UUID
	ProvidedAnswer string
	NewPassword    string
}

func (s *Service) LoginUser(ctx context.Context, req LoginRequest) (*ent.Session, error) {
	log := logrus.WithField("email", req.Email)
	log.Debug("user login attempt")

	u, err := s.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.WithError(err).Warn("login: failed to find user or db error during login attempt")
		return nil, fmt.Errorf("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
	if err != nil {
		log.Warn("login: invalid password provided")
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := auth.GenerateToken(u.ID, 10*time.Minute)
	if err != nil {
		log.WithError(err).Error("login: failed to generate access token")
		return nil, fmt.Errorf("could not process login: %w", err)
	}

	refreshToken, err := auth.GenerateRefreshToken(32)
	if err != nil {
		log.WithError(err).Error("login: failed to generate refresh token")
		return nil, fmt.Errorf("could not process login: %w", err)
	}

	session, err := s.Client.Session.
		Create().
		SetSessionID(uuid.New()).
		SetSessionType("auth").
		SetAccessToken(accessToken).
		SetRefreshToken(refreshToken).
		SetExpiresAt(time.Now().Add(15 * time.Minute)).
		SetIPAddress(req.IPAddress).
		SetUserAgent(req.UserAgent).
		SetUser(u).
		Save(ctx)

	if err != nil {
		log.WithError(err).Error("login: failed to save session to database")
		return nil, fmt.Errorf("could not save session: %w", err)
	}

	log.WithFields(logrus.Fields{
		"user_id":    u.ID,
		"session_id": session.SessionID,
	}).Info("user logged in successfully and session created")

	return session, nil
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
		if !ent.IsNotFound(err) {
			logrus.WithFields(logrus.Fields{
				"email": email,
				"error": err,
			}).Error("getUserByEmail: database error")
		}
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"user_id": u.ID,
		"email":   email,
	}).Debug("getUserByEmail: user found")
	return u, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
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

func (s *Service) RefreshSession(ctx context.Context, oldRefreshToken string) (*ent.Session, error) {
	log := logrus.WithField("refresh_token", oldRefreshToken)
	log.Debug("attempting to refresh session")

	session, err := s.Client.Session.
		Query().
		Where(session.RefreshTokenEQ(oldRefreshToken)).
		WithUser().
		Only(ctx)

	if err != nil {
		log.WithError(err).Warn("refresh: refresh token not found in database")
		return nil, fmt.Errorf("invalid refresh token")
	}

	if session.RevokedAt != nil {
		log.Warn("refresh: attempt to use a revoked refresh token")
		return nil, fmt.Errorf("invalid refresh token")
	}

	newAccessToken, err := auth.GenerateToken(session.Edges.User.ID, 15*time.Minute)
	if err != nil {
		log.WithError(err).Error("refresh: failed to generate new access token")
		return nil, err
	}

	newRefreshToken, err := auth.GenerateRefreshToken(32)
	if err != nil {
		log.WithError(err).Error("refresh: failed to generate new refresh token")
		return nil, err
	}

	updatedSession, err := session.Update().
		SetAccessToken(newAccessToken).
		SetRefreshToken(newRefreshToken).
		SetExpiresAt(time.Now().Add(15 * time.Minute)).
		Save(ctx)
	if err != nil {
		log.WithError(err).Error("refresh: failed to update session with new tokens")
		return nil, err
	}

	log.WithField("user_id", session.Edges.User.ID).Info("session refreshed successfully")
	return updatedSession, nil
}

func (s *Service) LogoutUser(ctx context.Context, accessToken string) error {
	log := logrus.WithField("access_token", accessToken)
	log.Debug("attempting to log out user by revoking session")

	session, err := s.Client.Session.
		Query().
		Where(session.AccessTokenEQ(accessToken)).
		Only(ctx)
	if err != nil {

		log.WithError(err).Warn("logout: could not find session for access token")
		return nil
	}

	_, err = session.Update().
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.WithError(err).Error("logout: failed to update session as revoked")
		return err
	}

	log.Info("session revoked successfully")
	return nil
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*ent.User, error) {
	logrus.WithField("user_id", userID).Debug("looking up user by id")

	u, err := s.Client.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			logrus.WithField("user_id", userID).Warn("getUserByID: user not found")
		} else {
			logrus.WithFields(logrus.Fields{
				"user_id": userID,
				"error":   err,
			}).Error("getUserByID: database error")
		}
		return nil, err
	}

	logrus.WithField("user_id", userID).Debug("getUserByID: user found")
	return u, nil
}

func (s *Service) AddSecurityQuestion(ctx context.Context, req AddSecurityQuestionRequest) (*ent.SecurityQuestion, error) {
	log := logrus.WithFields(logrus.Fields{
		"user_id":  req.UserID,
		"question": req.Question,
	})
	log.Debug("adding security question")

	u, err := s.GetUserByID(ctx, req.UserID)
	if err != nil {
		log.WithError(err).Warn("addSecurityQuestion: user not found")
		return nil, err
	}

	hashedAnswer, err := bcrypt.GenerateFromPassword([]byte(req.Answer), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("addSecurityQuestion: failed to hash answer")
		return nil, fmt.Errorf("could not process security question: %w", err)
	}

	sq, err := s.Client.SecurityQuestion.
		Create().
		SetQuestion(req.Question).
		SetAnswer(string(hashedAnswer)).
		SetUser(u).
		Save(ctx)

	if err != nil {
		log.WithError(err).Error("addSecurityQuestion: failed to save security question to database")
		return nil, fmt.Errorf("could not save security question: %w", err)
	}

	log.WithField("sq_id", sq.ID).Info("security question added successfully")
	return sq, nil
}

func (s *Service) GetRandomSecurityQuestionForUser(ctx context.Context, email string) (*ent.SecurityQuestion, error) {
	log := logrus.WithField("email", email)
	log.Debug("getting random security question for user")

	u, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		log.WithError(err).Warn("getRandomSecurityQuestion: user not found")
		return nil, fmt.Errorf("could not retrieve security question")
	}

	questions, err := u.QuerySecurityQuestions().All(ctx)
	if err != nil {
		log.WithError(err).Error("getRandomSecurityQuestion: failed to query security questions")
		return nil, fmt.Errorf("could not retrieve security question")
	}

	if len(questions) == 0 {
		log.Warn("getRandomSecurityQuestion: user has no security questions set up")
		return nil, fmt.Errorf("no security questions found for this user")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(questions))))
	if err != nil {
		log.WithError(err).Error("getRandomSecurityQuestion: failed to generate random index")
		return nil, fmt.Errorf("could not retrieve security question")
	}
	randomQuestion := questions[int(n.Int64())]
	randomQuestion.Answer = ""

	log.WithField("question_id", randomQuestion.ID).Info("random security question selected")
	return randomQuestion, nil
}

func (s *Service) ResetPasswordWithSecurityQuestion(ctx context.Context, req ResetPasswordWithSecurityQuestionRequest) error {
	log := logrus.WithField("question_id", req.QuestionID)
	log.Debug("attempting to reset password with security question")

	sq, err := s.Client.SecurityQuestion.
		Query().
		Where(securityquestion.ID(req.QuestionID)).
		WithUser().
		Only(ctx)
	if err != nil {
		log.WithError(err).Warn("resetPassword: could not find security question")
		return fmt.Errorf("invalid question or answer")
	}

	err = bcrypt.CompareHashAndPassword([]byte(sq.Answer), []byte(req.ProvidedAnswer))
	if err != nil {
		log.Warn("resetPassword: incorrect answer provided")
		return fmt.Errorf("invalid question or answer")
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("resetPassword: failed to hash new password")
		return fmt.Errorf("could not process password reset")
	}

	user := sq.Edges.User
	_, err = user.Update().
		SetPasswordHash(string(newHashedPassword)).
		Save(ctx)
	if err != nil {
		log.WithError(err).WithField("user_id", user.ID).Error("resetPassword: failed to update user password in db")
		return fmt.Errorf("could not process password reset")
	}

	log.WithField("user_id", user.ID).Info("password has been reset successfully via security question")
	return nil
}
