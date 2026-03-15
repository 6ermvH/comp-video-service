package model

import (
	"time"

	"github.com/google/uuid"
)

// Response stores respondent answer to one presented pair.
type Response struct {
	ID                 uuid.UUID `json:"id"`
	ParticipantID      uuid.UUID `json:"participant_id"`
	PairPresentationID uuid.UUID `json:"pair_presentation_id"`
	Choice             string    `json:"choice"`
	ReasonCodes        []string  `json:"reason_codes,omitempty"`
	Confidence         *int      `json:"confidence,omitempty"`
	ResponseTimeMS     *int      `json:"response_time_ms,omitempty"`
	ReplayCount        int       `json:"replay_count"`
	CreatedAt          time.Time `json:"created_at"`
}

// TaskResponseRequest is frontend payload for answer.
type TaskResponseRequest struct {
	Choice         string   `json:"choice" binding:"required"`
	ReasonCodes    []string `json:"reason_codes"`
	Confidence     *int     `json:"confidence"`
	ResponseTimeMS *int     `json:"response_time_ms"`
	ReplayCount    int      `json:"replay_count"`
}
