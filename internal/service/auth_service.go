package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"load_balancing_project_auth/internal/model"
	"load_balancing_project_auth/internal/repository"
)

var (
	ErrInvalidCredentials        = errors.New("invalid email or password")
	ErrInvalidLoginInput         = errors.New("email and password are required")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrInvalidRefreshToken       = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected; all active sessions have been revoked")
)

type AuthService struct {
	userRepository         repository.UserRepository
	sessionRepository      repository.SessionRepository
	refreshTokenRepository repository.RefreshTokenRepository
	tokenService           *TokenService
	auditService           *AuditService
}

func NewAuthService(
	userRepository repository.UserRepository,
	sessionRepository repository.SessionRepository,
	refreshTokenRepository repository.RefreshTokenRepository,
	tokenService *TokenService,
	auditService *AuditService,
) *AuthService {
	return &AuthService{
		userRepository:         userRepository,
		sessionRepository:      sessionRepository,
		refreshTokenRepository: refreshTokenRepository,
		tokenService:           tokenService,
		auditService:           auditService,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string, metadata model.RequestMetadata) (model.LoginResponse, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		s.logAuthEvent(ctx, nil, "login_failure", metadata, map[string]any{
			"email":  email,
			"reason": ErrInvalidLoginInput.Error(),
		})
		return model.LoginResponse{}, ErrInvalidLoginInput
	}

	user, err := s.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logAuthEvent(ctx, nil, "login_failure", metadata, map[string]any{
				"email":  email,
				"reason": ErrInvalidCredentials.Error(),
			})
			return model.LoginResponse{}, ErrInvalidCredentials
		}

		return model.LoginResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		s.logAuthEvent(ctx, stringPointer(user.ID), "login_failure", metadata, map[string]any{
			"email":  user.Email,
			"reason": ErrInvalidCredentials.Error(),
		})
		return model.LoginResponse{}, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	session := model.Session{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		DeviceName: strings.TrimSpace(metadata.DeviceName),
		IPAddress:  strings.TrimSpace(metadata.IPAddress),
		Status:     "ACTIVE",
		CreatedAt:  now,
		LastUsedAt: now,
	}

	if session.DeviceName == "" {
		session.DeviceName = "unknown device"
	}

	if _, err := s.sessionRepository.Create(ctx, session); err != nil {
		return model.LoginResponse{}, err
	}

	accessToken, expiresIn, err := s.tokenService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return model.LoginResponse{}, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return model.LoginResponse{}, err
	}
	refreshToken.SessionID = session.ID

	if err := s.refreshTokenRepository.Create(ctx, refreshToken); err != nil {
		return model.LoginResponse{}, err
	}

	s.logAuthEvent(ctx, stringPointer(user.ID), "login_success", metadata, map[string]any{
		"email":      user.Email,
		"session_id": session.ID,
	})

	return model.LoginResponse{
		Message:      "login successful",
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, email, password string, metadata model.RequestMetadata) (model.RegisterResponse, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		s.logAuthEvent(ctx, nil, "register_failure", metadata, map[string]any{
			"email":  email,
			"reason": ErrInvalidLoginInput.Error(),
		})
		return model.RegisterResponse{}, ErrInvalidLoginInput
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.RegisterResponse{}, err
	}

	now := time.Now().UTC()
	user := model.User{
		ID:              uuid.NewString(),
		Email:           strings.ToLower(email),
		HashedPassword:  string(passwordHash),
		IsEmailVerified: false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	createdUser, err := s.userRepository.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			s.logAuthEvent(ctx, nil, "register_failure", metadata, map[string]any{
				"email":  email,
				"reason": ErrEmailAlreadyExists.Error(),
			})
			return model.RegisterResponse{}, ErrEmailAlreadyExists
		}

		return model.RegisterResponse{}, err
	}

	s.logAuthEvent(ctx, stringPointer(createdUser.ID), "register_success", metadata, map[string]any{
		"email": createdUser.Email,
	})

	return model.RegisterResponse{
		Message:         "registration successful",
		UserID:          createdUser.ID,
		Email:           createdUser.Email,
		IsEmailVerified: createdUser.IsEmailVerified,
	}, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenValue string, metadata model.RequestMetadata) (model.RefreshTokenResponse, error) {
	refreshTokenValue = strings.TrimSpace(refreshTokenValue)
	if refreshTokenValue == "" {
		s.logAuthEvent(ctx, nil, "refresh_failure", metadata, map[string]any{
			"reason": ErrInvalidRefreshToken.Error(),
		})
		return model.RefreshTokenResponse{}, ErrInvalidRefreshToken
	}

	refreshToken, err := s.refreshTokenRepository.FindByToken(ctx, refreshTokenValue)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			s.logAuthEvent(ctx, nil, "refresh_failure", metadata, map[string]any{
				"reason": ErrInvalidRefreshToken.Error(),
			})
			return model.RefreshTokenResponse{}, ErrInvalidRefreshToken
		}

		return model.RefreshTokenResponse{}, err
	}

	if refreshToken.Status == "USED" {
		if err := s.refreshTokenRepository.RevokeAllByUser(ctx, refreshToken.UserID); err != nil {
			return model.RefreshTokenResponse{}, err
		}
		if err := s.sessionRepository.RevokeAllByUserID(ctx, refreshToken.UserID); err != nil {
			return model.RefreshTokenResponse{}, err
		}
		s.logAuthEvent(ctx, stringPointer(refreshToken.UserID), "token_reuse_detected", metadata, map[string]any{
			"session_id": refreshToken.SessionID,
		})

		return model.RefreshTokenResponse{}, ErrRefreshTokenReuseDetected
	}

	if refreshToken.Status != "ACTIVE" || time.Now().UTC().After(refreshToken.ExpiresAt) {
		s.logAuthEvent(ctx, stringPointer(refreshToken.UserID), "refresh_failure", metadata, map[string]any{
			"session_id": refreshToken.SessionID,
			"reason":     ErrInvalidRefreshToken.Error(),
		})
		return model.RefreshTokenResponse{}, ErrInvalidRefreshToken
	}

	user, err := s.userRepository.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RefreshTokenResponse{}, ErrInvalidRefreshToken
		}

		return model.RefreshTokenResponse{}, err
	}

	accessToken, expiresIn, err := s.tokenService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return model.RefreshTokenResponse{}, err
	}

	if err := s.refreshTokenRepository.UpdateStatus(ctx, refreshToken.Token, "USED"); err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return model.RefreshTokenResponse{}, ErrInvalidRefreshToken
		}

		return model.RefreshTokenResponse{}, err
	}

	newRefreshToken, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return model.RefreshTokenResponse{}, err
	}
	newRefreshToken.SessionID = refreshToken.SessionID

	if err := s.refreshTokenRepository.Create(ctx, newRefreshToken); err != nil {
		return model.RefreshTokenResponse{}, err
	}

	if err := s.sessionRepository.UpdateLastUsedAt(ctx, refreshToken.SessionID, sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}); err != nil {
		return model.RefreshTokenResponse{}, err
	}

	s.logAuthEvent(ctx, stringPointer(user.ID), "refresh_success", metadata, map[string]any{
		"session_id": refreshToken.SessionID,
	})

	return model.RefreshTokenResponse{
		Message:      "tokens refreshed successfully",
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenValue string, metadata model.RequestMetadata) (model.LogoutResponse, error) {
	refreshTokenValue = strings.TrimSpace(refreshTokenValue)
	if refreshTokenValue == "" {
		s.logAuthEvent(ctx, nil, "logout", metadata, map[string]any{
			"message": "logout requested without refresh token",
		})
		return model.LogoutResponse{Message: "logout successful"}, nil
	}

	refreshToken, err := s.refreshTokenRepository.FindByToken(ctx, refreshTokenValue)
	if err != nil && !errors.Is(err, repository.ErrRefreshTokenNotFound) {
		return model.LogoutResponse{}, err
	}

	err = s.refreshTokenRepository.RevokeByToken(ctx, refreshTokenValue)
	if err != nil && !errors.Is(err, repository.ErrRefreshTokenNotFound) {
		return model.LogoutResponse{}, err
	}

	if refreshToken.SessionID != "" {
		if err := s.sessionRepository.UpdateStatus(ctx, refreshToken.SessionID, "REVOKED"); err != nil {
			return model.LogoutResponse{}, err
		}
	}

	var userID *string
	if refreshToken.UserID != "" {
		userID = stringPointer(refreshToken.UserID)
	}
	s.logAuthEvent(ctx, userID, "logout", metadata, map[string]any{
		"session_id": refreshToken.SessionID,
	})

	return model.LogoutResponse{
		Message: "logout successful",
	}, nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string, metadata model.RequestMetadata) (model.LogoutResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		s.logAuthEvent(ctx, nil, "logout_all", metadata, map[string]any{
			"message": "logout all requested without user id",
		})
		return model.LogoutResponse{
			Message: "logout from all devices successful",
		}, nil
	}

	if err := s.refreshTokenRepository.RevokeAllByUser(ctx, userID); err != nil {
		return model.LogoutResponse{}, err
	}
	if err := s.sessionRepository.RevokeAllByUserID(ctx, userID); err != nil {
		return model.LogoutResponse{}, err
	}

	s.logAuthEvent(ctx, stringPointer(userID), "logout_all", metadata, map[string]any{})

	return model.LogoutResponse{
		Message: "logout from all devices successful",
	}, nil
}

func (s *AuthService) ListSessions(ctx context.Context, userID string) (model.SessionsResponse, error) {
	sessions, err := s.sessionRepository.ListByUserID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return model.SessionsResponse{}, err
	}

	return model.SessionsResponse{Sessions: sessions}, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) (model.SessionDeleteResponse, error) {
	session, err := s.sessionRepository.FindByID(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return model.SessionDeleteResponse{}, err
	}

	if session.UserID != strings.TrimSpace(userID) {
		return model.SessionDeleteResponse{}, ErrInvalidRefreshToken
	}

	if err := s.refreshTokenRepository.RevokeAllBySession(ctx, session.ID); err != nil {
		return model.SessionDeleteResponse{}, err
	}
	if err := s.sessionRepository.UpdateStatus(ctx, session.ID, "REVOKED"); err != nil {
		return model.SessionDeleteResponse{}, err
	}

	return model.SessionDeleteResponse{
		Message: "session revoked successfully",
	}, nil
}

func (s *AuthService) logAuthEvent(ctx context.Context, userID *string, eventType string, metadata model.RequestMetadata, extra map[string]any) {
	if s.auditService == nil {
		return
	}

	s.auditService.LogAuthEvent(ctx, userID, eventType, metadata, extra)
}

func stringPointer(value string) *string {
	return &value
}
