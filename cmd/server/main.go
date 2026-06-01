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

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		slog.Error("Failed to read schema.sql", slog.String("error", err.Error()))
		os.Exit(1)
	}
	_, err = dbConn.Exec(string(schema))
	if err != nil {
		slog.Error("Failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	saveLobby := db.BuildSaveLobbyClosure(dbConn)
	getLobby := db.BuildGetLobbyClosure(dbConn)

	checkRoomAccess := func(slug string, playerID string) (bool, error) {
		return db.CheckRoomAccess(dbConn, slug, playerID)
	}

	grantRoomAccess := func(slug string, playerID string) error {
		return db.GrantRoomAccess(dbConn, slug, playerID)
	}

	getRoomPasswordHash := func(slug string) (string, error) {
		return db.GetRoomPasswordHash(dbConn, slug)
	}

	registry := game.NewRegistryActor(slog.Default())
	go registry.Run()

	mux := http.NewServeMux()

	slog.Info("🎮 Starting Bingo Server", slog.String("port", "8080"))

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /", game.BuildHandleIndex())
	mux.HandleFunc("POST /rooms", game.BuildHandleCreateRoom(saveLobby))
	mux.HandleFunc("GET /room/{slug}", game.BuildHandleViewRoom(getLobby, checkRoomAccess))
	mux.HandleFunc("POST /room/{slug}/auth", game.BuildHandleAuthRoom(getRoomPasswordHash, grantRoomAccess))
	mux.HandleFunc("POST /room/{slug}/name", game.BuildHandleNameRoom())
	mux.HandleFunc("GET /ws/room/{slug}", game.BuildHandleRoomWS(registry, getLobby, checkRoomAccess))

	handler := web.LoggerMiddleware(mux)

	if err := http.ListenAndServe("0.0.0.0:8080", handler); err != nil {
		slog.Error("Server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
