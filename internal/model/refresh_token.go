package model

import "time"

type RefreshToken struct {
	Token     string    `json:"-"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
