package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tkozakas/sea-battle-server/internal/repository"
	"github.com/tkozakas/sea-battle-server/internal/service"
)

func newRoomManager() *service.RoomManager {
	return service.NewRoomManager(repository.NewMemoryGameRepository(), 1000)
}

func TestCreateRoom(t *testing.T) {
	rm := newRoomManager()
	code, err := rm.CreateRoom("player1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d", len(code))
	}
	charset := "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("code contains invalid character: %c", c)
		}
	}
}

func TestGetRoom(t *testing.T) {
	rm := newRoomManager()
	code, err := rm.CreateRoom("player1")
	if err != nil {
		t.Fatalf("unexpected error creating room: %v", err)
	}

	game, err := rm.GetRoom(code)
	if err != nil {
		t.Fatalf("unexpected error getting room: %v", err)
	}
	if game.ID != code {
		t.Errorf("expected game ID %s, got %s", code, game.ID)
	}
}

func TestGetRoomNotFound(t *testing.T) {
	rm := newRoomManager()
	_, err := rm.GetRoom("XXXXXX")
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

func TestRemoveRoom(t *testing.T) {
	rm := newRoomManager()
	code, _ := rm.CreateRoom("player1")

	if err := rm.RemoveRoom(code); err != nil {
		t.Fatalf("unexpected error removing room: %v", err)
	}

	_, err := rm.GetRoom(code)
	if err == nil {
		t.Error("expected error after remove, got nil")
	}
}

func TestActiveRoomCount(t *testing.T) {
	rm := newRoomManager()

	if rm.ActiveRoomCount() != 0 {
		t.Errorf("expected 0, got %d", rm.ActiveRoomCount())
	}

	_, _ = rm.CreateRoom("p1")
	_, _ = rm.CreateRoom("p2")

	if rm.ActiveRoomCount() != 2 {
		t.Errorf("expected 2, got %d", rm.ActiveRoomCount())
	}
}

func TestCleanupStaleRooms(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	rm := service.NewRoomManager(repo, 1000)

	code1, _ := rm.CreateRoom("p1")
	code2, _ := rm.CreateRoom("p2")

	game1, _ := repo.FindByID(code1)
	game2, _ := repo.FindByID(code2)

	game1.CreatedAt = game1.CreatedAt.Add(-2 * time.Minute)
	game2.CreatedAt = game2.CreatedAt.Add(-2 * time.Minute)

	_ = repo.Save(game1)
	_ = repo.Save(game2)

	rm.CleanupStaleRooms(1*time.Minute, 1*time.Minute)

	if rm.ActiveRoomCount() != 0 {
		t.Errorf("expected 0 rooms after cleanup, got %d", rm.ActiveRoomCount())
	}
}

func TestMaxRoomsEnforced(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	rm := service.NewRoomManager(repo, 2)

	_, err := rm.CreateRoom("p1")
	if err != nil {
		t.Fatalf("unexpected error creating room 1: %v", err)
	}
	_, err = rm.CreateRoom("p2")
	if err != nil {
		t.Fatalf("unexpected error creating room 2: %v", err)
	}
	_, err = rm.CreateRoom("p3")
	if err == nil {
		t.Fatal("expected error when max rooms exceeded, got nil")
	}
}

func TestGenerateCodeReturnsUniqueCode(t *testing.T) {
	rm := newRoomManager()
	code, err := rm.GenerateCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d", len(code))
	}
}
