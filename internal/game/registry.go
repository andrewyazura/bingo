package game

type RoomConfig struct {
	Mode      RoomMode
	Size      int
	Wordlist  []string
	FreeSpace bool
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
}

func NewRegistryActor() *RegistryActor {
	return &RegistryActor{
		Inbox:  make(chan RegistryCommand, 100),
		actors: make(map[string]*RoomActor),
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

			actor, err := NewRoomActor(cmd.Config.Mode, cmd.Config.Size, cmd.Config.Wordlist, cmd.Config.FreeSpace)
			if err != nil {
				cmd.ReplyTo <- nil
				continue
			}

			go actor.Run()
			r.actors[cmd.Slug] = actor
			cmd.ReplyTo <- actor
		}
	}
}
