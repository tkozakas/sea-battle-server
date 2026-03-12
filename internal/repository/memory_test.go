package repository_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/repository"
)

func TestSaveAndFind(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	game := domain.NewGame("code1", "player1")

	if err := repo.Save(game); err != nil {
		t.Fatalf("unexpected error saving game: %v", err)
	}

	found, err := repo.FindByID("code1")
	if err != nil {
		t.Fatalf("unexpected error finding game: %v", err)
	}
	if found.ID != game.ID {
		t.Errorf("expected game ID %s, got %s", game.ID, found.ID)
	}
}

func TestFindNotFound(t *testing.T) {
	repo := repository.NewMemoryGameRepository()

	_, err := repo.FindByID("nonexistent")
	if err != domain.ErrGameNotFound {
		t.Errorf("expected ErrGameNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	game := domain.NewGame("code1", "player1")

	_ = repo.Save(game)

	if err := repo.Delete("code1"); err != nil {
		t.Fatalf("unexpected error deleting game: %v", err)
	}

	_, err := repo.FindByID("code1")
	if err != domain.ErrGameNotFound {
		t.Errorf("expected ErrGameNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo := repository.NewMemoryGameRepository()

	err := repo.Delete("nonexistent")
	if err != domain.ErrGameNotFound {
		t.Errorf("expected ErrGameNotFound, got %v", err)
	}
}

func TestCount(t *testing.T) {
	repo := repository.NewMemoryGameRepository()

	if repo.Count() != 0 {
		t.Errorf("expected count 0, got %d", repo.Count())
	}

	_ = repo.Save(domain.NewGame("code1", "p1"))
	_ = repo.Save(domain.NewGame("code2", "p2"))

	if repo.Count() != 2 {
		t.Errorf("expected count 2, got %d", repo.Count())
	}
}

func TestConcurrentAccess(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("code%d", i)
			game := domain.NewGame(id, fmt.Sprintf("player%d", i))
			_ = repo.Save(game)
		}(i)
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("code%d", i)
			_, _ = repo.FindByID(id)
		}(i)
	}

	wg.Wait()

	if repo.Count() != n {
		t.Errorf("expected count %d, got %d", n, repo.Count())
	}
}

func TestFindByIDReturnsDeepCopy(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	game := domain.NewGame("code1", "player1")
	_ = repo.Save(game)

	found1, err := repo.FindByID("code1")
	if err != nil {
		t.Fatal(err)
	}
	found1.State = domain.StatePlaying

	found2, err := repo.FindByID("code1")
	if err != nil {
		t.Fatal(err)
	}
	if found2.State == domain.StatePlaying {
		t.Error("FindByID should return a deep copy; modifying result should not affect stored game")
	}
}

func TestSaveStoresDeepCopy(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	game := domain.NewGame("code1", "player1")
	_ = repo.Save(game)

	game.State = domain.StatePlaying

	found, err := repo.FindByID("code1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.State == domain.StatePlaying {
		t.Error("Save should store a deep copy; modifying the original after save should not affect stored game")
	}
}
