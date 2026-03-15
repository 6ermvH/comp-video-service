package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InteractionLog captures client-side events.
type InteractionLog struct {
	ID                 uuid.UUID       `json:"id"`
	ParticipantID      uuid.UUID       `json:"participant_id"`
	PairPresentationID *uuid.UUID      `json:"pair_presentation_id,omitempty"`
	EventType          string          `json:"event_type"`
	EventTS            time.Time       `json:"event_ts"`
	PayloadJSON        json.RawMessage `json:"payload_json,omitempty"`
}

// TaskEventRequest logs one task-level interaction.
type TaskEventRequest struct {
	EventType   string          `json:"event_type" binding:"required"`
	PayloadJSON json.RawMessage `json:"payload_json"`
}
