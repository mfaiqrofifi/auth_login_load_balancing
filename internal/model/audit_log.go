package model

import "time"

type AuditLog struct {
	ID        string
	UserID    *string
	EventType string
	IPAddress string
	UserAgent string
	Metadata  []byte
	CreatedAt time.Time
}
