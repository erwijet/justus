package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"justus/internal/models"
)

type contextKey struct{}

// RequireUser is middleware that looks up the user from the session cookie.
func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var user models.User
		if err := h.db.Where("spotify_id = ?", cookie.Value).First(&user).Error; err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey{}, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(contextKey{}).(*models.User)
	return user
}

type Handler struct {
	auth      *spotifyauth.Authenticator
	oauth2Cfg oauth2.Config
	db        *gorm.DB
	states    sync.Map
	OnLogin   func(user *models.User)
}

func NewHandler(db *gorm.DB, clientID, clientSecret, redirectURL string) *Handler {
	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPlaybackState, spotifyauth.ScopeUserReadRecentlyPlayed),
	)
	oauth2Cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  spotifyauth.AuthURL,
			TokenURL: spotifyauth.TokenURL,
		},
	}
	return &Handler{auth: auth, oauth2Cfg: oauth2Cfg, db: db}
}

type persistingTokenSource struct {
	inner  oauth2.TokenSource
	db     *gorm.DB
	userID uuid.UUID
	last   string
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.inner.Token()
	if err != nil {
		return nil, err
	}
	if token.AccessToken != s.last {
		s.last = token.AccessToken
		s.db.Model(&models.User{}).Where("id = ?", s.userID).Updates(map[string]any{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"token_expiry":  token.Expiry,
		})
	}
	return token, nil
}

// SpotifyClient returns a Spotify client for the given user.
// Tokens are automatically refreshed and persisted back to the DB.
func (h *Handler) SpotifyClient(ctx context.Context, user *models.User) *spotify.Client {
	token := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       user.TokenExpiry,
	}
	src := &persistingTokenSource{
		inner:  h.oauth2Cfg.TokenSource(ctx, token),
		db:     h.db,
		userID: user.ID,
		last:   user.AccessToken,
	}
	return spotify.New(oauth2.NewClient(ctx, src))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	state := randomState()
	h.states.Store(state, true)
	url := h.auth.AuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if _, ok := h.states.LoadAndDelete(state); !ok {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}

	token, err := h.auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "failed to get token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := spotify.New(h.auth.Client(r.Context(), token))
	spotifyUser, err := client.CurrentUser(r.Context())
	if err != nil {
		http.Error(w, "failed to get user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var user models.User
	h.db.Where(models.User{SpotifyID: string(spotifyUser.ID)}).
		Assign(models.User{
			DisplayName:  spotifyUser.DisplayName,
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenExpiry:  token.Expiry,
		}).
		FirstOrCreate(&user)

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    user.SpotifyID,
		Path:     "/",
		HttpOnly: true,
	})

	if h.OnLogin != nil {
		go h.OnLogin(&user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "logged in",
		"user":    spotifyUser.DisplayName,
	})
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
