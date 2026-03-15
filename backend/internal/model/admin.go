package model

import (
	"time"

	"github.com/google/uuid"
)

// Admin is the administrator account stored in PostgreSQL.
type Admin struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// LoginRequest is the admin login payload.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned after successful authentication.
type LoginResponse struct {
	Token     string `json:"token"`
	CSRFToken string `json:"csrf_token"`
	Admin     *Admin `json:"admin"`
}
