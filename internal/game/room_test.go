package game

import (
	"log/slog"
	"testing"
)

func TestActor_CollaborativeInit(t *testing.T) {
	words := generateWordlist(25)

	actor, err := NewRoomActor("test-collab", Collaborative, 5, words, false, slog.Default(), nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, exists := actor.boards["global"]; !exists {
		t.Errorf("expected global board to be initialized")
	}
}

func TestActor_CompetitiveJoin(t *testing.T) {
	words := generateWordlist(25)

	actor, err := NewRoomActor("test-comp", Competitive, 5, words, false, slog.Default(), nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(actor.boards) != 0 {
		t.Errorf("expected 0 boards initially")
	}

	err = actor.Join(&Command{Type: JoinCommand, PlayerID: "alice"})
	if err != nil {
		t.Fatalf("expected no error on join, got %v", err)
	}

	if _, exists := actor.boards["alice"]; !exists {
		t.Errorf("expected board for alice to be initialized")
	}
}

func TestActor_MarkAndEvents(t *testing.T) {
	words := generateWordlist(25)

	actor, err := NewRoomActor("test-comp", Competitive, 5, words, false, slog.Default(), nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	actor.Join(&Command{Type: JoinCommand, PlayerID: "alice"})

	board := actor.boards["alice"]
	board.Marks[0] = true
	board.Marks[1] = true
	board.Marks[2] = true
	board.Marks[3] = true

	inbox := make(chan Event, 10)
	actor.subscribers["alice"] = inbox

	actor.Mark(&Command{Type: MarkCommand, PlayerID: "alice", TileIndex: 4})

	event1 := <-inbox
	event2 := <-inbox

	if event1.Type != TileMarkedEvent {
		t.Errorf("expected first event to be TileMarkedEvent")
	}
	if event2.Type != BingoEvent {
		t.Errorf("expected second event to be BingoEvent")
	}
}
