// Seed: nạp kịch bản JSON vào DB content.
//
//	go run ./cmd/seed                       # nạp content/demo-story.json
//	go run ./cmd/seed path/to/story.json    # nạp file khác
package main

import (
	"log"
	"os"

	"fmv-game/backend/internal/seed"
	"fmv-game/backend/internal/store"
)

func main() {
	path := "content/demo-story.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/game.db"
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("mở DB: %v", err)
	}
	defer st.Close()

	if err := seed.LoadFile(st, path); err != nil {
		log.Fatalf("seed %s: %v", path, err)
	}
	log.Printf("Seed OK: %s → %s", path, dbPath)
}
