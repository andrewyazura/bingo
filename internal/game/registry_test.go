package game

import (
	"sync"
	"testing"
)

func TestRegistryActor_GetNonExistent(t *testing.T) {
	registry := NewRegistryActor()
	go registry.Run()

	replyChan := make(chan *RoomActor, 1)
	registry.Inbox <- RegistryCommand{
		Type:    GetRoomCmd,
		Slug:    "does-not-exist",
		ReplyTo: replyChan,
	}

	actor := <-replyChan
	if actor != nil {
		t.Errorf("expected nil actor for non-existent room, got %v", actor)
	}
}

func TestRegistryActor_CreateAndGet(t *testing.T) {
	registry := NewRegistryActor()
	go registry.Run()

	config := RoomConfig{
		Mode:      Collaborative,
		Size:      5,
		Wordlist:  generateWordlist(25),
		FreeSpace: true,
	}

	replyChan := make(chan *RoomActor, 1)
	registry.Inbox <- RegistryCommand{
		Type:    CreateRoomCmd,
		Slug:    "my-room",
		Config:  config,
		ReplyTo: replyChan,
	}

	actor1 := <-replyChan
	if actor1 == nil {
		t.Fatalf("expected created actor, got nil")
	}

	registry.Inbox <- RegistryCommand{
		Type:    GetRoomCmd,
		Slug:    "my-room",
		ReplyTo: replyChan,
	}

	actor2 := <-replyChan
	if actor2 != actor1 {
		t.Errorf("expected to get the exact same actor pointer, got different")
	}
}

func TestRegistryActor_CreateRace(t *testing.T) {
	registry := NewRegistryActor()
	go registry.Run()

	config := RoomConfig{
		Mode:      Collaborative,
		Size:      5,
		Wordlist:  generateWordlist(25),
		FreeSpace: true,
	}

	var wg sync.WaitGroup
	actors := make([]*RoomActor, 10)

	for i := range 10 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			replyChan := make(chan *RoomActor, 1)
			registry.Inbox <- RegistryCommand{
				Type:    CreateRoomCmd,
				Slug:    "race-room",
				Config:  config,
				ReplyTo: replyChan,
			}
			actors[index] = <-replyChan
		}(i)
	}

	wg.Wait()

	firstActor := actors[0]
	if firstActor == nil {
		t.Fatalf("expected valid actor, got nil")
	}

	for i := 1; i < 10; i++ {
		if actors[i] != firstActor {
			t.Errorf("goroutine %d got a different actor pointer!", i)
		}
	}
}
