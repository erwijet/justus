package models

import (
	"time"

	"github.com/google/uuid"
	"justus/internal/db"
)

type Listen struct {
	db.Base
	UserID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_played_at"`
	SongID   uuid.UUID `gorm:"type:uuid"`
	PlayedAt time.Time `gorm:"uniqueIndex:idx_user_played_at"`
}
