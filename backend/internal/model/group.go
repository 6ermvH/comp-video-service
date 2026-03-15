package model

import (
	"time"

	"github.com/google/uuid"
)

// Group is a category inside a study.
type Group struct {
	ID                 uuid.UUID `json:"id"`
	StudyID            uuid.UUID `json:"study_id"`
	Name               string    `json:"name"`
	Description        *string   `json:"description,omitempty"`
	Priority           int       `json:"priority"`
	TargetVotesPerPair int       `json:"target_votes_per_pair"`
	CreatedAt          time.Time `json:"created_at"`
}

// CreateGroupRequest creates a new group in study.
type CreateGroupRequest struct {
	Name               string `json:"name" binding:"required"`
	Description        string `json:"description"`
	Priority           int    `json:"priority"`
	TargetVotesPerPair int    `json:"target_votes_per_pair"`
}
