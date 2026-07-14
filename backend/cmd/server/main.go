// Server entrypoint.
//
//	go run ./cmd/seed    # nạp demo story (chạy 1 lần)
//	go run ./cmd/server  # http://localhost:8080
package main

import (
	"log"
	"net/http"
	"os"

	"fmv-game/backend/internal/api"
	"fmv-game/backend/internal/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dbPath := env("DB_PATH", "data/game.db")
	mediaDir := env("MEDIA_DIR", "media")
	addr := env("ADDR", ":8080")

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("mở DB: %v", err)
	}
	defer st.Close()

	srv := api.New(st, mediaDir)
	log.Printf("FMV Director server chạy tại %s (DB: %s, media: %s)", addr, dbPath, mediaDir)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
