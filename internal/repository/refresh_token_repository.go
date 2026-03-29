package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"load_balancing_project_auth/internal/model"
)

const refreshTokenKeyPrefix = "auth:refresh_token:"
const userRefreshTokensKeyPrefix = "auth:user_refresh_tokens:"
const sessionRefreshTokensKeyPrefix = "auth:session_refresh_tokens:"

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken model.RefreshToken) error
	FindByToken(ctx context.Context, token string) (model.RefreshToken, error)
	UpdateStatus(ctx context.Context, token string, status string) error
	RevokeByToken(ctx context.Context, token string) error
	RevokeAllBySession(ctx context.Context, sessionID string) error
	RevokeAllByUser(ctx context.Context, userID string) error
}

type RedisRefreshTokenRepository struct {
	client *redis.Client
}

type refreshTokenRecord struct {
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewRedisRefreshTokenRepository(client *redis.Client) *RedisRefreshTokenRepository {
	return &RedisRefreshTokenRepository{client: client}
}

func (r *RedisRefreshTokenRepository) Create(ctx context.Context, refreshToken model.RefreshToken) error {
	record := refreshTokenRecord{
		UserID:    refreshToken.UserID,
		SessionID: refreshToken.SessionID,
		Status:    refreshToken.Status,
		CreatedAt: refreshToken.CreatedAt,
		ExpiresAt: refreshToken.ExpiresAt,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	ttl := time.Until(refreshToken.ExpiresAt)
	if ttl <= 0 {
		return ErrRefreshTokenNotFound
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, redisRefreshTokenKey(refreshToken.Token), payload, ttl)
	pipe.SAdd(ctx, redisUserRefreshTokensKey(refreshToken.UserID), refreshToken.Token)
	pipe.SAdd(ctx, redisSessionRefreshTokensKey(refreshToken.SessionID), refreshToken.Token)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisRefreshTokenRepository) FindByToken(ctx context.Context, token string) (model.RefreshToken, error) {
	value, err := r.client.Get(ctx, redisRefreshTokenKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return model.RefreshToken{}, ErrRefreshTokenNotFound
		}

		return model.RefreshToken{}, err
	}

	var record refreshTokenRecord
	if err := json.Unmarshal([]byte(value), &record); err != nil {
		return model.RefreshToken{}, err
	}

	return model.RefreshToken{
		Token:     token,
		UserID:    record.UserID,
		SessionID: record.SessionID,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (r *RedisRefreshTokenRepository) UpdateStatus(ctx context.Context, token string, status string) error {
	refreshToken, err := r.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	refreshToken.Status = status

	record := refreshTokenRecord{
		UserID:    refreshToken.UserID,
		SessionID: refreshToken.SessionID,
		Status:    refreshToken.Status,
		CreatedAt: refreshToken.CreatedAt,
		ExpiresAt: refreshToken.ExpiresAt,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	ttl := time.Until(refreshToken.ExpiresAt)
	if ttl <= 0 {
		return ErrRefreshTokenNotFound
	}

	return r.client.Set(ctx, redisRefreshTokenKey(token), payload, ttl).Err()
}

func (r *RedisRefreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	return r.UpdateStatus(ctx, token, "REVOKED")
}

func (r *RedisRefreshTokenRepository) RevokeAllBySession(ctx context.Context, sessionID string) error {
	tokens, err := r.client.SMembers(ctx, redisSessionRefreshTokensKey(sessionID)).Result()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		refreshToken, err := r.FindByToken(ctx, token)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenNotFound) {
				continue
			}

			return err
		}

		if refreshToken.Status == "ACTIVE" || refreshToken.Status == "USED" {
			if err := r.UpdateStatus(ctx, token, "REVOKED"); err != nil {
				if errors.Is(err, ErrRefreshTokenNotFound) {
					continue
				}

				return err
			}
		}
	}

	return nil
}

func (r *RedisRefreshTokenRepository) RevokeAllByUser(ctx context.Context, userID string) error {
	tokens, err := r.client.SMembers(ctx, redisUserRefreshTokensKey(userID)).Result()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		refreshToken, err := r.FindByToken(ctx, token)
		if err != nil {
			if errors.Is(err, ErrRefreshTokenNotFound) {
				continue
			}

			return err
		}

		if refreshToken.Status == "ACTIVE" || refreshToken.Status == "USED" {
			if err := r.UpdateStatus(ctx, token, "REVOKED"); err != nil {
				if errors.Is(err, ErrRefreshTokenNotFound) {
					continue
				}

				return err
			}
		}
	}

	return nil
}

func redisRefreshTokenKey(token string) string {
	return fmt.Sprintf("%s%s", refreshTokenKeyPrefix, token)
}

func redisUserRefreshTokensKey(userID string) string {
	return fmt.Sprintf("%s%s", userRefreshTokensKeyPrefix, userID)
}

func redisSessionRefreshTokensKey(sessionID string) string {
	return fmt.Sprintf("%s%s", sessionRefreshTokensKeyPrefix, sessionID)
}
