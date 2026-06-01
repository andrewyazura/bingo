package game

import (
	"errors"
	"log/slog"
	"math/rand"
)

type RoomMode int

const (
	Competitive RoomMode = iota
	Collaborative
)

type CommandType int

const (
	JoinCommand CommandType = iota
	MarkCommand
	SubscribeCommand
	UnsubscribeCommand
)

type Command struct {
	Type      CommandType
	PlayerID  string
	TileIndex int
	ReplyTo   chan Event
}

type EventType int

const (
	BingoEvent EventType = iota
	OneToBingoEvent
	TileMarkedEvent
	TileUnmarkedEvent
	BoardStateEvent
)

type Event struct {
	Type      EventType
	PlayerID  string
	TileIndex *int
	TileWord  *string
	Board     *Board
}

type RoomActor struct {
	Mode      RoomMode
	Size      int
	Wordlist  []string
	FreeSpace bool

	boards      map[string]*Board
	subscribers map[string]chan Event
	Inbox       chan Command
	logger      *slog.Logger
}

func NewRoomActor(mode RoomMode, size int, wordlist []string, freeSpace bool, logger *slog.Logger) (*RoomActor, error) {
	actor := &RoomActor{
		Mode:        mode,
		Size:        size,
		Wordlist:    wordlist,
		FreeSpace:   freeSpace,
		boards:      make(map[string]*Board),
		subscribers: make(map[string]chan Event),
		Inbox:       make(chan Command, 100),
		logger:      logger,
	}

	if mode == Collaborative {
		board, err := NewBoard(size, wordlist, freeSpace, rand.Shuffle)
		if err != nil {
			return nil, err
		}
		actor.boards["global"] = board
	}

	return actor, nil
}

func (a *RoomActor) Run() {
	for cmd := range a.Inbox {
		switch cmd.Type {
		case SubscribeCommand:
			a.subscribers[cmd.PlayerID] = cmd.ReplyTo
			a.logger.Info("Player connected via WebSocket", slog.String("player_id", cmd.PlayerID))
		case UnsubscribeCommand:
			delete(a.subscribers, cmd.PlayerID)
			close(cmd.ReplyTo)
			a.logger.Info("Player disconnected", slog.String("player_id", cmd.PlayerID))
		case JoinCommand:
			a.Join(&cmd)
		case MarkCommand:
			a.Mark(&cmd)
		}
	}
}

func (a *RoomActor) broadcast(e Event) {
	for id, subChan := range a.subscribers {
		select {
		case subChan <- e:
		default:
			println("Warning: Dropping event for slow client", id)
		}
	}
}

func (a *RoomActor) sendToPlayer(playerID string, e Event) {
	if subChan, ok := a.subscribers[playerID]; ok {
		select {
		case subChan <- e:
		default:
			println("Warning: Dropping event for slow client", playerID)
		}
	}
}

func (a *RoomActor) Join(cmd *Command) error {
	key := cmd.PlayerID
	if a.Mode == Collaborative {
		key = "global"
	}

	board, exists := a.boards[key]
	if !exists {
		var err error
		board, err = NewBoard(a.Size, a.Wordlist, a.FreeSpace, rand.Shuffle)
		if err != nil {
			return err
		}
		a.boards[key] = board
	}

	if sub, ok := a.subscribers[cmd.PlayerID]; ok {
		sub <- Event{
			Type:     BoardStateEvent,
			PlayerID: cmd.PlayerID,
			Board:    board,
		}
	}

	return nil
}

func (a *RoomActor) Mark(cmd *Command) error {
	key := cmd.PlayerID

	if a.Mode == Collaborative {
		key = "global"
	}

	board, exists := a.boards[key]
	if !exists {
		return errors.New("board doesn't exist")
	}

	word, isMarked := board.MarkTile(cmd.TileIndex)
	if word == nil {
		return nil
	}

	eventType := TileMarkedEvent
	if !isMarked {
		eventType = TileUnmarkedEvent
	}

	event := Event{
		Type:      eventType,
		PlayerID:  cmd.PlayerID,
		TileIndex: &cmd.TileIndex,
		TileWord:  word,
	}

	if a.Mode == Collaborative {
		a.broadcast(event)
	} else {
		a.sendToPlayer(cmd.PlayerID, event)
	}

	if !isMarked {
		return nil
	}

	analysis := board.Analyze(cmd.TileIndex)

	if analysis.HasBingo {
		a.logger.Info("Player achieved Bingo!", slog.String("player_id", cmd.PlayerID))
		a.broadcast(Event{
			Type:      BingoEvent,
			PlayerID:  cmd.PlayerID,
			TileIndex: nil,
		})
	} else if analysis.MaxInLine == a.Size-1 {
		a.broadcast(Event{
			Type:      OneToBingoEvent,
			PlayerID:  cmd.PlayerID,
			TileIndex: nil,
		})
	}

	return nil
}
