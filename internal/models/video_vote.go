package models

import (
	"time"

	"github.com/google/uuid"
)

type VideoVote struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	VideoID   uuid.UUID `json:"video_id" db:"video_id"`
	VoteType  int       `json:"vote_type" db:"vote_type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
