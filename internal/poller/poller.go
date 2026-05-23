package poller

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"justus/internal/auth"
	"justus/internal/models"
)

type Poller struct {
	db          *gorm.DB
	authHandler *auth.Handler
}

func New(db *gorm.DB, authHandler *auth.Handler) *Poller {
	return &Poller{db: db, authHandler: authHandler}
}

// Start runs the polling loop every 10 minutes. Blocks until ctx is cancelled.
func (p *Poller) Start(ctx context.Context) {
	p.pollAll(ctx)

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollAll(ctx)
		}
	}
}

func (p *Poller) PollUser(user *models.User) {
	if err := p.pollUser(context.Background(), user); err != nil {
		log.Printf("poller: user %s: %v", user.SpotifyID, err)
	}
}

func (p *Poller) pollAll(ctx context.Context) {
	var users []models.User
	if err := p.db.Find(&users).Error; err != nil {
		log.Printf("poller: failed to load users: %v", err)
		return
	}

	for _, user := range users {
		if err := p.pollUser(ctx, &user); err != nil {
			log.Printf("poller: user %s: %v", user.SpotifyID, err)
		}
	}
}

func (p *Poller) pollUser(ctx context.Context, user *models.User) error {
	client := p.authHandler.SpotifyClient(ctx, user)

	items, err := client.PlayerRecentlyPlayed(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		artists := make([]string, len(item.Track.Artists))
		for j, a := range item.Track.Artists {
			artists[j] = a.Name
		}

		var song models.Song
		p.db.Where(models.Song{SpotifyID: string(item.Track.ID)}).
			Attrs(models.Song{
				Name:       item.Track.Name,
				ArtistName: strings.Join(artists, ", "),
			}).
			FirstOrCreate(&song)

		listen := models.Listen{
			UserID:   user.ID,
			SongID:   song.ID,
			PlayedAt: item.PlayedAt,
		}
		p.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&listen)
	}

	return nil
}
