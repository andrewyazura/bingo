package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"bingo/internal/platform/web"
	"golang.org/x/crypto/bcrypt"
)

func generateSlug() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	tmpl, err := template.ParseFiles("web/templates/error.html")
	if err != nil {
		http.Error(w, message, statusCode)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	tmpl.Execute(w, struct {
		Message    string
		StatusCode int
	}{
		Message:    message,
		StatusCode: statusCode,
	})
}

func getOrSetPlayerID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("player_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	newID := "guest-" + generateSlug()
	http.SetCookie(w, &http.Cookie{
		Name:     "player_id",
		Value:    newID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 365,
	})
	return newID
}

func BuildHandleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		getOrSetPlayerID(w, r)
		http.ServeFile(w, r, "web/templates/index.html")
	}
}

func BuildHandleCreateRoom(saveLobby func(slug string, config RoomConfig) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			renderError(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		modeStr := r.FormValue("mode")
		mode := 0
		if modeStr == "1" {
			mode = 1
		}

		sizeStr := r.FormValue("size")
		size := 5
		fmt.Sscanf(sizeStr, "%d", &size)

		freeSpace := r.FormValue("freeSpace") == "true"

		wordlistStr := r.FormValue("wordlist")
		var wordlist []string
		for w := range strings.SplitSeq(wordlistStr, ",") {
			trimmed := strings.TrimSpace(w)
			if trimmed != "" {
				wordlist = append(wordlist, trimmed)
			}
		}

		slug := generateSlug()

		password := r.FormValue("password")
		var hashedPassword string
		if password != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				renderError(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			hashedPassword = string(hash)
		}

		config := RoomConfig{
			Mode:        RoomMode(mode),
			Size:        size,
			Wordlist:    wordlist,
			FreeSpace:   freeSpace,
			Password:    hashedPassword,
			HasPassword: password != "",
		}

		if err := config.Validate(); err != nil {
			renderError(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := saveLobby(slug, config)
		if err != nil {
			renderError(w, "Failed to save lobby", http.StatusInternalServerError)
			traceID, _ := r.Context().Value(web.TraceIDKey).(string)
			slog.Error("Failed to save lobby", slog.String("trace_id", traceID), slog.String("error", err.Error()))
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/room/%s", slug), http.StatusSeeOther)
	}
}

func BuildHandleViewRoom(getLobby func(slug string) (RoomConfig, error), checkRoomAccess func(string, string) (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		playerID := getOrSetPlayerID(w, r)

		config, err := getLobby(slug)
		if err != nil {
			renderError(w, "Room not found", http.StatusNotFound)
			return
		}

		_, nameErr := r.Cookie("player_name")
		if nameErr != nil {
			tmpl, err := template.ParseFiles("web/templates/name_prompt.html")
			if err != nil {
				renderError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			tmpl.Execute(w, struct{ Slug string }{Slug: slug})
			return
		}

		if config.HasPassword {
			hasAccess, err := checkRoomAccess(slug, playerID)
			if err != nil {
				renderError(w, "Error checking access", http.StatusInternalServerError)
				return
			}
			if !hasAccess {
				tmpl, err := template.ParseFiles("web/templates/password_prompt.html")
				if err != nil {
					renderError(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				tmpl.Execute(w, struct{ Slug string }{Slug: slug})
				return
			}
		}

		tmpl, err := template.ParseFiles("web/templates/room.html")
		if err != nil {
			renderError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct{ Slug string }{Slug: slug})
	}
}

func BuildHandleRoomWS(registry *RegistryActor, getLobby func(slug string) (RoomConfig, error), checkRoomAccess func(string, string) (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		playerID := getOrSetPlayerID(w, r)

		config, err := getLobby(slug)
		if err == nil && config.HasPassword {
			hasAccess, err := checkRoomAccess(slug, playerID)
			if err != nil || !hasAccess {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		replyChan := make(chan *RoomActor, 1)

		registry.Inbox <- RegistryCommand{
			Type:    GetRoomCmd,
			Slug:    slug,
			ReplyTo: replyChan,
		}
		actor := <-replyChan

		if actor == nil {
			config, err := getLobby(slug)
			if err != nil {
				http.Error(w, "Room not found", http.StatusNotFound)
				return
			}

			registry.Inbox <- RegistryCommand{
				Type:    CreateRoomCmd,
				Slug:    slug,
				Config:  config,
				ReplyTo: replyChan,
			}
			actor = <-replyChan

			if actor == nil {
				http.Error(w, "Failed to start room", http.StatusInternalServerError)
				return
			}
		}

		ServeWS(actor, playerID, w, r)
	}
}

func BuildHandleAuthRoom(getRoomPasswordHash func(string) (string, error), grantRoomAccess func(string, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		playerID := getOrSetPlayerID(w, r)

		if err := r.ParseForm(); err != nil {
			renderError(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")
		if password == "" {
			renderError(w, "Password is required", http.StatusBadRequest)
			return
		}

		hash, err := getRoomPasswordHash(slug)
		if err != nil {
			renderError(w, "Room not found or error", http.StatusInternalServerError)
			return
		}

		if hash == "" {

			http.Redirect(w, r, fmt.Sprintf("/room/%s", slug), http.StatusSeeOther)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {

			renderError(w, "Incorrect password", http.StatusUnauthorized)
			return
		}

		err = grantRoomAccess(slug, playerID)
		if err != nil {
			renderError(w, "Failed to grant access", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/room/%s", slug), http.StatusSeeOther)
	}
}

func BuildHandleNameRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		_ = getOrSetPlayerID(w, r)

		if err := r.ParseForm(); err != nil {
			renderError(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		playerName := strings.TrimSpace(r.FormValue("player_name"))
		if playerName == "" {
			renderError(w, "Name is required", http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "player_name",
			Value:    playerName,
			Path:     "/",
			HttpOnly: false,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 365,
		})

		http.Redirect(w, r, fmt.Sprintf("/room/%s", slug), http.StatusSeeOther)
	}
}
