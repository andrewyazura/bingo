package game

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 50 * time.Second
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ClientActor struct {
	Conn     *websocket.Conn
	PlayerID string
	Inbox    chan Event
	Room     *RoomActor
}

func (c *ClientActor) readPump() {
	defer func() {
		c.Room.Inbox <- Command{
			Type:     UnsubscribeCommand,
			PlayerID: c.PlayerID,
			ReplyTo:  c.Inbox,
		}
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Room.logger.Error("WebSocket read error", slog.String("error", err.Error()), slog.String("player_id", c.PlayerID))
			}
			break
		}

		var payload struct {
			TileIndex int `json:"tileIndex"`
		}
		if err := json.Unmarshal(message, &payload); err == nil {
			c.Room.Inbox <- Command{
				Type:      MarkCommand,
				PlayerID:  c.PlayerID,
				TileIndex: payload.TileIndex,
			}
		}
	}
}

func (c *ClientActor) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.Inbox:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			html := RenderEvent(event, c.Room.Size)
			if html != "" {
				w.Write([]byte(html))
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWS(actor *RoomActor, playerID string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		actor.logger.Error("Failed to upgrade websocket", slog.String("error", err.Error()), slog.String("player_id", playerID))
		return
	}

	client := &ClientActor{
		Conn:     conn,
		PlayerID: playerID,
		Inbox:    make(chan Event, 256),
		Room:     actor,
	}

	actor.Inbox <- Command{
		Type:     SubscribeCommand,
		PlayerID: playerID,
		ReplyTo:  client.Inbox,
	}

	actor.Inbox <- Command{
		Type:     JoinCommand,
		PlayerID: playerID,
	}

	go client.writePump()
	go client.readPump()
}
