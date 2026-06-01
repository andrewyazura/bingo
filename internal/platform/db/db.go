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
			INSERT INTO rooms (slug, mode, size, wordlist, free_space, password_hash)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		wordlistJson, err := json.Marshal(config.Wordlist)
		if err != nil {
			return err
		}

		var passwordHash *string
		if config.Password != "" {
			passwordHash = &config.Password
		}

		_, err = db.Exec(query, slug, int(config.Mode), config.Size, wordlistJson, config.FreeSpace, passwordHash)
		return err
	}
}

func BuildGetLobbyClosure(db *sql.DB) func(slug string) (game.RoomConfig, error) {
	return func(slug string) (game.RoomConfig, error) {
		query := `
			SELECT mode, size, wordlist, free_space, password_hash IS NOT NULL
			FROM rooms
			WHERE slug = $1
		`
		var config game.RoomConfig
		var mode int
		var wordlistJson []byte
		var hasPassword bool

		err := db.QueryRow(query, slug).Scan(&mode, &config.Size, &wordlistJson, &config.FreeSpace, &hasPassword)
		if err != nil {
			return config, err
		}

		config.Mode = game.RoomMode(mode)
		config.HasPassword = hasPassword
		err = json.Unmarshal(wordlistJson, &config.Wordlist)
		return config, err
	}
}

func CheckRoomAccess(db *sql.DB, roomSlug string, playerID string) (bool, error) {
	query := `
		SELECT 1 FROM room_access WHERE room_slug = $1 AND player_id = $2
	`
	var dummy int
	err := db.QueryRow(query, roomSlug, playerID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func GrantRoomAccess(db *sql.DB, roomSlug string, playerID string) error {
	query := `
		INSERT INTO room_access (room_slug, player_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	_, err := db.Exec(query, roomSlug, playerID)
	return err
}

func GetRoomPasswordHash(db *sql.DB, roomSlug string) (string, error) {
	query := `SELECT password_hash FROM rooms WHERE slug = $1`
	var hash *string
	err := db.QueryRow(query, roomSlug).Scan(&hash)
	if err != nil {
		return "", err
	}
	if hash == nil {
		return "", nil
	}
	return *hash, nil
}
