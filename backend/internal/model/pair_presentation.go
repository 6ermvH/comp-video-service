package model

import (
	"time"

	"github.com/google/uuid"
)

// PairPresentation is one randomized task shown to participant.
type PairPresentation struct {
	ID               uuid.UUID `json:"id"`
	ParticipantID    uuid.UUID `json:"participant_id"`
	SourceItemID     uuid.UUID `json:"source_item_id"`
	LeftAssetID      uuid.UUID `json:"left_asset_id"`
	RightAssetID     uuid.UUID `json:"right_asset_id"`
	LeftMethodType   string    `json:"left_method_type"`
	RightMethodType  string    `json:"right_method_type"`
	TaskOrder        int       `json:"task_order"`
	IsAttentionCheck bool      `json:"is_attention_check"`
	IsPractice       bool      `json:"is_practice"`
	CreatedAt        time.Time `json:"created_at"`
}

// TaskPayload is next task payload for respondent flow.
type TaskPayload struct {
	PresentationID   uuid.UUID `json:"presentation_id"`
	SourceItemID     uuid.UUID `json:"source_item_id"`
	TaskOrder        int       `json:"task_order"`
	IsAttentionCheck bool      `json:"is_attention_check"`
	IsPractice       bool      `json:"is_practice"`
	Left             *Video    `json:"left"`
	Right            *Video    `json:"right"`
}
