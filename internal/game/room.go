package game

import (
	"errors"
	"hash/fnv"
	"log/slog"
	"math/rand"
)

type RoomMode int

const (
	Competitive RoomMode = iota
	Collaborative
)

func getSeededShuffle(roomSlug string, playerID string) func(int, func(i, j int)) {
	h := fnv.New64a()
	h.Write([]byte(roomSlug + playerID))
	src := rand.NewSource(int64(h.Sum64()))
	r := rand.New(src)
	return r.Shuffle
}

type CommandType int

const (
	JoinCommand CommandType = iota
	MarkCommand
	SubscribeCommand
	UnsubscribeCommand
)

type Command struct {
	Type       CommandType
	PlayerID   string
	PlayerName string
	TileIndex  int
	ReplyTo    chan Event
}

type EventType int

const (
	BingoEvent EventType = iota
	OneToBingoEvent
	TileMarkedEvent
	TileUnmarkedEvent
	BoardStateEvent
	OpponentBoardsStateEvent
)

type OpponentBoardState struct {
	PlayerID   string
	PlayerName string
	Board      *Board
}

type Event struct {
	Type       EventType
	PlayerID   string
	PlayerName string
	TileIndex  *int
	TileWord   *string
	Board      *Board
	Opponents  []OpponentBoardState
}

type RoomActor struct {
	Slug      string
	Mode      RoomMode
	Size      int
	Wordlist  []string
	FreeSpace bool

	boards      map[string]*Board
	playerNames map[string]string
	subscribers map[string]chan Event
	Inbox       chan Command
	logger      *slog.Logger
	saveEvent   func(Event)
	history     []Event
}

func NewRoomActor(slug string, mode RoomMode, size int, wordlist []string, freeSpace bool, logger *slog.Logger, saveEvent func(Event), history []Event) (*RoomActor, error) {
	actor := &RoomActor{
		Slug:        slug,
		Mode:        mode,
		Size:        size,
		Wordlist:    wordlist,
		FreeSpace:   freeSpace,
		boards:      make(map[string]*Board),
		playerNames: make(map[string]string),
		subscribers: make(map[string]chan Event),
		Inbox:       make(chan Command, 256),
		logger:      logger,
		saveEvent:   saveEvent,
		history:     history,
	}

	if mode == Collaborative {
		board, err := NewBoard(size, wordlist, freeSpace, getSeededShuffle(slug, "global"))
		if err != nil {
			return nil, err
		}
		actor.replayHistory("global", board)
		actor.boards["global"] = board
	}

	return actor, nil
}

func (a *RoomActor) replayHistory(playerID string, board *Board) {
	for _, e := range a.history {
		if e.PlayerID == playerID {
			if e.Type == TileMarkedEvent && e.TileWord != nil {
				board.MarkByWord(*e.TileWord, true)
			} else if e.Type == TileUnmarkedEvent && e.TileWord != nil {
				board.MarkByWord(*e.TileWord, false)
			}
		}
	}
}

func (a *RoomActor) Run() {
	for cmd := range a.Inbox {
		switch cmd.Type {
		case SubscribeCommand:
			a.subscribers[cmd.PlayerID] = cmd.ReplyTo
			a.playerNames[cmd.PlayerID] = cmd.PlayerName
			a.logger.Info("Player connected via WebSocket", slog.String("player_id", cmd.PlayerID))
		case UnsubscribeCommand:
			delete(a.subscribers, cmd.PlayerID)
			delete(a.playerNames, cmd.PlayerID)
			close(cmd.ReplyTo)
			a.logger.Info("Player disconnected", slog.String("player_id", cmd.PlayerID))
			a.broadcastOpponents()
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

func (a *RoomActor) broadcastOpponents() {
	if a.Mode == Collaborative {
		return
	}

	activePlayers := make([]string, 0)
	for pID := range a.subscribers {
		activePlayers = append(activePlayers, pID)
	}

	for pID, subChan := range a.subscribers {
		opps := make([]OpponentBoardState, 0)
		for _, oppID := range activePlayers {
			if oppID != pID {
				opps = append(opps, OpponentBoardState{
					PlayerID:   oppID,
					PlayerName: a.playerNames[oppID],
					Board:      a.boards[oppID],
				})
			}
		}
		select {
		case subChan <- Event{
			Type:      OpponentBoardsStateEvent,
			Opponents: opps,
		}:
		default:
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
		board, err = NewBoard(a.Size, a.Wordlist, a.FreeSpace, getSeededShuffle(a.Slug, cmd.PlayerID))
		if err != nil {
			return err
		}
		a.replayHistory(key, board)
		a.boards[key] = board
	}

	if sub, ok := a.subscribers[cmd.PlayerID]; ok {
		sub <- Event{
			Type:       BoardStateEvent,
			PlayerID:   cmd.PlayerID,
			PlayerName: a.playerNames[cmd.PlayerID],
			Board:      board,
		}
	}

	a.broadcastOpponents()

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
		Type:       eventType,
		PlayerID:   cmd.PlayerID,
		PlayerName: a.playerNames[cmd.PlayerID],
		TileIndex:  &cmd.TileIndex,
		TileWord:   word,
	}

	if a.Mode == Collaborative {
		event.PlayerID = "global"
	}

	if a.saveEvent != nil {
		a.saveEvent(event)
	}

	a.broadcast(event)

	if !isMarked {
		return nil
	}

	analysis := board.Analyze(cmd.TileIndex)

	if analysis.HasBingo {
		a.logger.Info("Player achieved Bingo!", slog.String("player_id", cmd.PlayerID))
		a.broadcast(Event{
			Type:       BingoEvent,
			PlayerID:   cmd.PlayerID,
			PlayerName: a.playerNames[cmd.PlayerID],
		})
	} else if analysis.MaxInLine == a.Size-1 {
		a.broadcast(Event{
			Type:       OneToBingoEvent,
			PlayerID:   cmd.PlayerID,
			PlayerName: a.playerNames[cmd.PlayerID],
		})
	}

	return nil
}
