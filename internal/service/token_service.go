package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"load_balancing_project_auth/internal/model"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type TokenService struct {
	accessSecret     []byte
	accessTTLMinutes int
	refreshTTLHours  int
}

func NewTokenService(accessSecret string, accessTTLMinutes int, refreshTTLHours int) *TokenService {
	return &TokenService{
		accessSecret:     []byte(accessSecret),
		accessTTLMinutes: accessTTLMinutes,
		refreshTTLHours:  refreshTTLHours,
	}
}

func (s *TokenService) GenerateAccessToken(userID, email string) (string, int64, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(s.accessTTLMinutes) * time.Minute)

	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", 0, err
	}

	return signedToken, int64(time.Until(expiresAt).Seconds()), nil
}

func (s *TokenService) ValidateAccessToken(accessToken string) (model.AuthIdentity, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}

		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return model.AuthIdentity{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return model.AuthIdentity{}, ErrInvalidToken
	}

	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return model.AuthIdentity{}, ErrInvalidToken
	}

	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return model.AuthIdentity{}, ErrInvalidToken
	}

	return model.AuthIdentity{
		UserID: subject,
		Email:  email,
	}, nil
}

func (s *TokenService) GenerateRefreshToken(userID string) (model.RefreshToken, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return model.RefreshToken{}, err
	}

	now := time.Now().UTC()
	return model.RefreshToken{
		Token:     base64.RawURLEncoding.EncodeToString(randomBytes),
		UserID:    userID,
		Status:    "ACTIVE",
		ExpiresAt: now.Add(time.Duration(s.refreshTTLHours) * time.Hour),
		CreatedAt: now,
	}, nil
}
