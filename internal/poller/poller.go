package poller

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"justus/internal/auth"
	"justus/internal/models"
)

const pollInterval = 10 * time.Second
const listenThreshold = 0.80

type trackingState struct {
	spotifyTrackID string
	trackName      string
	artistNames    []string
	durationMs     int
	accumulatedMs  int
	lastPollTime   time.Time
}

type Poller struct {
	db          *gorm.DB
	authHandler *auth.Handler
	mu          sync.Mutex
	tracking    map[uuid.UUID]*trackingState
}

func New(db *gorm.DB, authHandler *auth.Handler) *Poller {
	return &Poller{
		db:          db,
		authHandler: authHandler,
		tracking:    make(map[uuid.UUID]*trackingState),
	}
}

func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
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
			log.Printf("poller: processing user %s (%s)", user.SpotifyID, user.DisplayName)
			log.Printf("poller: user %s: %v", user.SpotifyID, err)
		}
	}
}

func (p *Poller) pollUser(ctx context.Context, user *models.User) error {
	log.Printf("poller [%s, %s]: polling...", user.DisplayName, user.SpotifyID)
	client := p.authHandler.SpotifyClient(ctx, user)

	playing, err := client.PlayerCurrentlyPlaying(ctx)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state := p.tracking[user.ID]
	now := time.Now()

	// Nothing playing or no track
	if playing == nil || playing.Item == nil {
		log.Printf("poller [%s, %s]: not playing", user.DisplayName, user.SpotifyID)
		if state != nil {
			p.finalize(user, state)
			delete(p.tracking, user.ID)
		}
		return nil
	}

	trackID := string(playing.Item.ID)

	// Track changed — finalize the old one
	if state != nil && state.spotifyTrackID != trackID {
		log.Printf("poller [%s, %s]: track switched", user.DisplayName, user.SpotifyID)
		p.finalize(user, state)
		state = nil
	}

	// Start tracking a new track, seed with current progress
	if state == nil {
		log.Printf("poller [%s, %s]: begin tracking %s", user.DisplayName, user.SpotifyID, playing.Item.Name)
		artists := make([]string, len(playing.Item.Artists))
		for i, a := range playing.Item.Artists {
			artists[i] = a.Name
		}
		p.tracking[user.ID] = &trackingState{
			spotifyTrackID: trackID,
			trackName:      playing.Item.Name,
			artistNames:    artists,
			durationMs:     int(playing.Item.Duration),
			accumulatedMs:  int(playing.Progress),
			lastPollTime:   now,
		}
		return nil
	}

	// Same track, accumulate only while actively playing
	if playing.Playing {
		log.Printf("poller [%s, %s]: playback in progress", user.DisplayName, user.SpotifyID)
		elapsed := int(now.Sub(state.lastPollTime).Milliseconds())
		state.accumulatedMs += elapsed
	}
	state.lastPollTime = now

	// Record and reset if the threshold has been reached
	if state.durationMs > 0 && float64(state.accumulatedMs) >= float64(state.durationMs)*listenThreshold {
		log.Printf("poller [%s, %s]: threshold reached, resetting", user.DisplayName, user.SpotifyID)
		p.recordListen(user, state)
		state.accumulatedMs = 0
	}

	return nil
}

func (p *Poller) finalize(user *models.User, state *trackingState) {
	log.Printf("poller [%s, %s]: finalizing", user.DisplayName, user.SpotifyID)
	if state.durationMs <= 0 {
		return
	}

	ratio := float64(state.accumulatedMs) / float64(state.durationMs)
	if ratio < listenThreshold {
		log.Printf("poller [%s, %s]: skipped %s (%.0f%%)", user.DisplayName, user.SpotifyID, state.trackName, ratio*100)
		return
	}

	p.recordListen(user, state)
}

func (p *Poller) recordListen(user *models.User, state *trackingState) {
	log.Printf("poller [%s, %s]: writing listen for %s", user.DisplayName, user.SpotifyID, state.trackName)

	var song models.Song
	p.db.Where(models.Song{SpotifyID: state.spotifyTrackID}).
		Attrs(models.Song{
			Name:       state.trackName,
			ArtistName: strings.Join(state.artistNames, ", "),
		}).
		FirstOrCreate(&song)

	listen := models.Listen{
		UserID:   user.ID,
		SongID:   song.ID,
		PlayedAt: time.Now(),
	}
	p.db.Create(&listen)
}
