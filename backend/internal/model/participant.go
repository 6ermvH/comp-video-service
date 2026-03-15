package model

import (
	"time"

	"github.com/google/uuid"
)

// Participant is one respondent session.
type Participant struct {
	ID           uuid.UUID  `json:"id"`
	SessionToken string     `json:"session_token"`
	StudyID      uuid.UUID  `json:"study_id"`
	DeviceType   *string    `json:"device_type,omitempty"`
	Browser      *string    `json:"browser,omitempty"`
	Role         *string    `json:"role,omitempty"`
	Experience   *string    `json:"experience,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	QualityFlag  *string    `json:"quality_flag,omitempty"`
}

// StartSessionRequest starts respondent session.
type StartSessionRequest struct {
	StudyID    uuid.UUID `json:"study_id" binding:"required"`
	DeviceType string    `json:"device_type"`
	Browser    string    `json:"browser"`
	Role       string    `json:"role"`
	Experience string    `json:"experience"`
}
