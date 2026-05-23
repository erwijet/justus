package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"justus/internal/auth"
	"justus/internal/poller"
)

func main() {
	godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database: ", err)
	}

	spotifyAuth := auth.NewHandler(db,
		os.Getenv("SPOTIFY_CLIENT_ID"),
		os.Getenv("SPOTIFY_CLIENT_SECRET"),
		os.Getenv("SPOTIFY_REDIRECT_URL"),
	)

	p := poller.New(db, spotifyAuth)
	spotifyAuth.OnLogin = p.PollUser
	go p.Start(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", spotifyAuth.Login)
	mux.HandleFunc("GET /callback", spotifyAuth.Callback)
	mux.Handle("GET /healthcheck", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{
			"ok": true,
		})
	}))
	mux.Handle("GET /me", spotifyAuth.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})))

	addr := ":3000"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
