package model

import "time"

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	HashedPassword  string    `json:"-"`
	IsEmailVerified bool      `json:"is_email_verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
