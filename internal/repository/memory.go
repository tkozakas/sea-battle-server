package repository

import (
	"sync"

	"github.com/tkozakas/sea-battle-server/internal/domain"
)

type MemoryGameRepository struct {
	mu    sync.RWMutex
	games map[string]*domain.Game
}

func NewMemoryGameRepository() *MemoryGameRepository {
	return &MemoryGameRepository{
		games: make(map[string]*domain.Game),
	}
}

func (r *MemoryGameRepository) Save(game *domain.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.games[game.ID] = game.DeepCopy()
	return nil
}

func (r *MemoryGameRepository) FindByID(id string) (*domain.Game, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	game, ok := r.games[id]
	if !ok {
		return nil, domain.ErrGameNotFound
	}
	return game.DeepCopy(), nil
}

func (r *MemoryGameRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.games[id]; !ok {
		return domain.ErrGameNotFound
	}
	delete(r.games, id)
	return nil
}

func (r *MemoryGameRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.games)
}

func (r *MemoryGameRepository) AllIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.games))
	for id := range r.games {
		ids = append(ids, id)
	}
	return ids
}
