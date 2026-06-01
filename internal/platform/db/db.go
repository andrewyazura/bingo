package db

import (
	"database/sql"
	"encoding/json"
	"log"

	"bingo/internal/game"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func BuildSaveEventClosure(db *sql.DB, roomSlug string) func(game.Event) {
	return func(e game.Event) {
		query := `
			INSERT INTO game_events (room_slug, player_id, event_type, tile_index, tile_word) 
			VALUES ($1, $2, $3, $4, $5)
		`

		_, err := db.Exec(query, roomSlug, e.PlayerID, int(e.Type), e.TileIndex, e.TileWord)

		if err != nil {
			log.Printf("failed to save event for room %s: %v", roomSlug, err)
		}
	}
}

func BuildSaveLobbyClosure(db *sql.DB) func(slug string, config game.RoomConfig) error {
	return func(slug string, config game.RoomConfig) error {
		query := `
			INSERT INTO rooms (slug, mode, size, wordlist, free_space)
			VALUES ($1, $2, $3, $4, $5)
		`
		wordlistJson, err := json.Marshal(config.Wordlist)
		if err != nil {
			return err
		}

		_, err = db.Exec(query, slug, int(config.Mode), config.Size, wordlistJson, config.FreeSpace)
		return err
	}
}

func BuildGetLobbyClosure(db *sql.DB) func(slug string) (game.RoomConfig, error) {
	return func(slug string) (game.RoomConfig, error) {
		query := `
			SELECT mode, size, wordlist, free_space
			FROM rooms
			WHERE slug = $1
		`
		var config game.RoomConfig
		var mode int
		var wordlistJson []byte

		err := db.QueryRow(query, slug).Scan(&mode, &config.Size, &wordlistJson, &config.FreeSpace)
		if err != nil {
			return config, err
		}

		config.Mode = game.RoomMode(mode)
		err = json.Unmarshal(wordlistJson, &config.Wordlist)
		return config, err
	}
}
