package main

import (
	"log/slog"
	"net/http"
	"os"

	"bingo/internal/game"
	"bingo/internal/platform/db"
	"bingo/internal/platform/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5432/bingo?sslmode=disable"
	}

	dbConn, err := db.Connect(dsn)
	if err != nil {
		slog.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			slug TEXT PRIMARY KEY,
			mode INT NOT NULL,
			size INT NOT NULL,
			wordlist JSONB NOT NULL,
			free_space BOOLEAN NOT NULL
		);
		CREATE TABLE IF NOT EXISTS game_events (
			id SERIAL PRIMARY KEY,
			room_slug TEXT REFERENCES rooms(slug),
			player_id TEXT NOT NULL,
			event_type INT NOT NULL,
			tile_index INT NOT NULL,
			tile_word TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		slog.Error("Failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	saveLobby := db.BuildSaveLobbyClosure(dbConn)
	getLobby := db.BuildGetLobbyClosure(dbConn)

	registry := game.NewRegistryActor(slog.Default())
	go registry.Run()

	mux := http.NewServeMux()

	slog.Info("🎮 Starting Bingo Server", slog.String("port", "8080"))

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /", game.BuildHandleIndex())
	mux.HandleFunc("POST /rooms", game.BuildHandleCreateRoom(saveLobby))
	mux.HandleFunc("GET /room/{slug}", game.BuildHandleViewRoom())
	mux.HandleFunc("GET /ws/room/{slug}", game.BuildHandleRoomWS(registry, getLobby))

	handler := web.LoggerMiddleware(mux)

	if err := http.ListenAndServe("0.0.0.0:8080", handler); err != nil {
		slog.Error("Server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
