package models

import (
	"time"

	"justus/internal/db"
)

type User struct {
	db.Base
	SpotifyID    string `gorm:"uniqueIndex"`
	DisplayName  string
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	TokenExpiry  time.Time `json:"-"`
}
