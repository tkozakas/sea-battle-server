package repository

import "github.com/tkozakas/sea-battle-server/internal/domain"

type GameRepository interface {
	Save(game *domain.Game) error
	FindByID(id string) (*domain.Game, error)
	Delete(id string) error
	Count() int
	AllIDs() []string
}
