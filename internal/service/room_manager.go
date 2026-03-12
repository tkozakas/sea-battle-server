package service

import (
	"math/rand"
	"time"

	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/repository"
)

const codeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const codeLength = 6

type RoomManager struct {
	repo repository.GameRepository
}

func NewRoomManager(repo repository.GameRepository) *RoomManager {
	return &RoomManager{repo: repo}
}

func (rm *RoomManager) GenerateCode() string {
	for {
		code := generateRandomCode()
		if _, err := rm.repo.FindByID(code); err != nil {
			return code
		}
	}
}

func generateRandomCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeCharset[rand.Intn(len(codeCharset))]
	}
	return string(b)
}

func (rm *RoomManager) CreateRoom(creatorID string) (string, error) {
	code := rm.GenerateCode()
	game := domain.NewGame(code, creatorID)
	if err := rm.repo.Save(game); err != nil {
		return "", err
	}
	return code, nil
}

func (rm *RoomManager) GetRoom(code string) (*domain.Game, error) {
	return rm.repo.FindByID(code)
}

func (rm *RoomManager) RemoveRoom(code string) error {
	return rm.repo.Delete(code)
}

func (rm *RoomManager) ActiveRoomCount() int {
	return rm.repo.Count()
}

func (rm *RoomManager) CleanupStaleRooms(waitingTimeout, finishedTimeout time.Duration) {
	allCodes := rm.repo.AllIDs()
	now := time.Now()
	toDelete := make([]string, 0)

	for _, code := range allCodes {
		game, err := rm.repo.FindByID(code)
		if err != nil {
			continue
		}
		age := now.Sub(game.CreatedAt)
		if game.State == domain.StateWaiting && age > waitingTimeout {
			toDelete = append(toDelete, code)
			continue
		}
		if (game.State == domain.StateGameOver || game.State == domain.StateAbandoned) && age > finishedTimeout {
			toDelete = append(toDelete, code)
		}
	}

	for _, code := range toDelete {
		_ = rm.repo.Delete(code)
	}
}
