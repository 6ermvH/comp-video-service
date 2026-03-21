package model

import (
	"time"

	"github.com/google/uuid"
)

// SourceItem is one image source that has a video pair.
type SourceItem struct {
	ID               uuid.UUID `json:"id"`
	StudyID          uuid.UUID `json:"study_id"`
	GroupID          uuid.UUID `json:"group_id"`
	SourceImageID    *string   `json:"source_image_id,omitempty"`
	PairCode         *string   `json:"pair_code,omitempty"`
	Difficulty       *string   `json:"difficulty,omitempty"`
	IsAttentionCheck bool      `json:"is_attention_check"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// SourceItemDetail extends SourceItem with denormalised aggregate fields
// returned by the admin list endpoint.
type SourceItemDetail struct {
	ID               uuid.UUID `json:"id"`
	StudyID          uuid.UUID `json:"study_id"`
	GroupID          uuid.UUID `json:"group_id"`
	GroupName        string    `json:"group_name"`
	SourceImageID    *string   `json:"source_image_id,omitempty"`
	PairCode         *string   `json:"pair_code,omitempty"`
	Difficulty       *string   `json:"difficulty,omitempty"`
	IsAttentionCheck bool      `json:"is_attention_check"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	AssetCount       int       `json:"asset_count"`
	ResponseCount    int       `json:"response_count"`
}

// UpdateSourceItemRequest is the body for PATCH /admin/source-items/:id.
type UpdateSourceItemRequest struct {
	IsAttentionCheck bool `json:"is_attention_check"`
}

// CreatePairRequest creates a source item and links two existing video assets.
type CreatePairRequest struct {
	GroupID          uuid.UUID `json:"group_id" binding:"required"`
	BaselineVideoID  uuid.UUID `json:"baseline_video_id" binding:"required"`
	CandidateVideoID uuid.UUID `json:"candidate_video_id" binding:"required"`
	PairCode         string    `json:"pair_code"`
	Difficulty       string    `json:"difficulty"`
	Notes            string    `json:"notes"`
	IsAttentionCheck bool      `json:"is_attention_check"`
}
