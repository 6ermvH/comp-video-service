package model

import (
	"time"

	"github.com/google/uuid"
)

// Study defines one respondent experiment setup.
type Study struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	EffectType             string    `json:"effect_type"`
	Status                 string    `json:"status"`
	MaxTasksPerParticipant int       `json:"max_tasks_per_participant"`
	InstructionsText       *string   `json:"instructions_text,omitempty"`
	TieOptionEnabled       bool      `json:"tie_option_enabled"`
	ReasonsEnabled         bool      `json:"reasons_enabled"`
	ConfidenceEnabled      bool      `json:"confidence_enabled"`
	CreatedAt              time.Time `json:"created_at"`
}

// CreateStudyRequest is the payload for creating study.
type CreateStudyRequest struct {
	Name                   string `json:"name" binding:"required"`
	EffectType             string `json:"effect_type" binding:"required"`
	MaxTasksPerParticipant int    `json:"max_tasks_per_participant"`
	InstructionsText       string `json:"instructions_text"`
	TieOptionEnabled       *bool  `json:"tie_option_enabled"`
	ReasonsEnabled         *bool  `json:"reasons_enabled"`
	ConfidenceEnabled      *bool  `json:"confidence_enabled"`
}

// UpdateStudyStatusRequest updates study status.
type UpdateStudyStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
