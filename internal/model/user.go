package model

import "time"

type User struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

type Session struct {
	Token     string    `json:"token"`
	UserID    uint64    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
