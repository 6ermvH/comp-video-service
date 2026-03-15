package model

import (
	"time"

	"github.com/google/uuid"
)

// VideoStatus represents the lifecycle state of a video asset.
type VideoStatus string

const (
	VideoStatusActive   VideoStatus = "active"
	VideoStatusArchived VideoStatus = "archived"
)

// Video represents a single video file stored in S3 (table: video_assets).
type Video struct {
	ID           uuid.UUID   `json:"id"`
	SourceItemID *uuid.UUID  `json:"source_item_id,omitempty"`
	MethodType   *string     `json:"method_type,omitempty"`
	Title        string      `json:"title"`
	Description  *string     `json:"description,omitempty"`
	S3Key        string      `json:"s3_key"`
	DurationMS   *int        `json:"duration_ms,omitempty"`
	Width        *int        `json:"width,omitempty"`
	Height       *int        `json:"height,omitempty"`
	FPS          *float32    `json:"fps,omitempty"`
	Codec        *string     `json:"codec,omitempty"`
	Checksum     *string     `json:"checksum,omitempty"`
	Status       VideoStatus `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	PresignedURL string      `json:"presigned_url,omitempty"`
}
