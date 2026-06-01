package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func generateSlug() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getOrSetPlayerID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("player_id")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	newID := "guest_" + generateSlug()
	http.SetCookie(w, &http.Cookie{
		Name:     "player_id",
		Value:    newID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in prod with HTTPS
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
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
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

		err := saveLobby(slug, RoomConfig{
			Mode:      RoomMode(mode),
			Size:      size,
			Wordlist:  wordlist,
			FreeSpace: freeSpace,
		})
		if err != nil {
			http.Error(w, "Failed to save lobby", http.StatusInternalServerError)
			log.Printf("DB error: %v", err)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/room/%s", slug), http.StatusSeeOther)
	}
}

func BuildHandleViewRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		getOrSetPlayerID(w, r)

		tmpl, err := template.ParseFiles("web/templates/room.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		tmpl.Execute(w, struct{ Slug string }{Slug: slug})
	}
}

func BuildHandleRoomWS(registry *RegistryActor, getLobby func(slug string) (RoomConfig, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		playerID := getOrSetPlayerID(w, r)

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
