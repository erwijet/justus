package models

import "justus/internal/db"

type Song struct {
	db.Base
	SpotifyID  string `gorm:"uniqueIndex"`
	Name       string
	ArtistName string
}
