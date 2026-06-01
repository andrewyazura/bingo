package game

import "log/slog"

type RoomConfig struct {
	Mode        RoomMode
	Size        int
	Wordlist    []string
	FreeSpace   bool
	Password    string `json:"-"` 
	HasPassword bool   
}

type RegistryCommandType int

const (
	GetRoomCmd RegistryCommandType = iota
	CreateRoomCmd
)

type RegistryCommand struct {
	Type    RegistryCommandType
	Slug    string
	Config  RoomConfig
	ReplyTo chan *RoomActor
}

type RegistryActor struct {
	Inbox  chan RegistryCommand
	actors map[string]*RoomActor
	logger *slog.Logger
}

func NewRegistryActor(logger *slog.Logger) *RegistryActor {
	return &RegistryActor{
		Inbox:  make(chan RegistryCommand, 100),
		actors: make(map[string]*RoomActor),
		logger: logger,
	}
}

func (r *RegistryActor) Run() {
	for cmd := range r.Inbox {
		switch cmd.Type {
		case GetRoomCmd:
			cmd.ReplyTo <- r.actors[cmd.Slug]

		case CreateRoomCmd:
			if existing, exists := r.actors[cmd.Slug]; exists {
				cmd.ReplyTo <- existing
				continue
			}

			roomLogger := r.logger.With("room_slug", cmd.Slug, "mode", cmd.Config.Mode)
			actor, err := NewRoomActor(cmd.Config.Mode, cmd.Config.Size, cmd.Config.Wordlist, cmd.Config.FreeSpace, roomLogger)
			if err != nil {
				r.logger.Error("Failed to create room actor", slog.String("slug", cmd.Slug), slog.String("error", err.Error()))
				cmd.ReplyTo <- nil
				continue
			}

			go actor.Run()
			r.actors[cmd.Slug] = actor
			r.logger.Info("Room created", slog.String("slug", cmd.Slug), slog.Int("size", cmd.Config.Size))
			cmd.ReplyTo <- actor
		}
	}
}
