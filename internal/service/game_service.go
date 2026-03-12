package service

import (
	"errors"

	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/repository"
)

var ErrPlayerNotFound = errors.New("player not found in any game")

type GameService struct {
	rooms *RoomManager
	repo  repository.GameRepository
}

func NewGameService(repo repository.GameRepository) *GameService {
	return &GameService{
		rooms: NewRoomManager(repo),
		repo:  repo,
	}
}

func (s *GameService) CreateGame(creatorID string) (string, error) {
	return s.rooms.CreateRoom(creatorID)
}

func (s *GameService) JoinGame(code string, playerID string) error {
	game, err := s.repo.FindByID(code)
	if err != nil {
		return err
	}
	if err := game.Join(playerID); err != nil {
		return err
	}
	return s.repo.Save(game)
}

func (s *GameService) PlaceShips(code string, playerIndex int, ships []*domain.Ship) error {
	game, err := s.repo.FindByID(code)
	if err != nil {
		return err
	}
	if err := game.PlaceShips(playerIndex, ships); err != nil {
		return err
	}
	return s.repo.Save(game)
}

func (s *GameService) Fire(code string, playerIndex int, target domain.Point) (*domain.ShotResult, error) {
	game, err := s.repo.FindByID(code)
	if err != nil {
		return nil, err
	}
	result, err := game.Fire(playerIndex, target)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(game); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GameService) GetGame(code string) (*domain.Game, error) {
	return s.repo.FindByID(code)
}

func (s *GameService) HandleDisconnect(code string, playerIndex int) error {
	game, err := s.repo.FindByID(code)
	if err != nil {
		return err
	}
	game.Players[playerIndex].Connected = false
	return s.repo.Save(game)
}

func (s *GameService) HandleReconnect(code string, playerID string) (*domain.Game, int, error) {
	game, err := s.repo.FindByID(code)
	if err != nil {
		return nil, -1, err
	}
	for i, p := range game.Players {
		if p != nil && p.ID == playerID {
			p.Connected = true
			if err := s.repo.Save(game); err != nil {
				return nil, -1, err
			}
			return game, i, nil
		}
	}
	return nil, -1, ErrPlayerNotFound
}

func (s *GameService) ActiveRoomCount() int {
	return s.rooms.ActiveRoomCount()
}
