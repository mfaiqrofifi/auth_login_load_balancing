package model

import "time"

type RequestMetadata struct {
	UserAgent  string
	IPAddress  string
	DeviceName string
}

type Session struct {
	ID         string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	DeviceName string    `json:"device_name"`
	IPAddress  string    `json:"ip_address"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type SessionsResponse struct {
	Sessions []Session `json:"sessions"`
}
