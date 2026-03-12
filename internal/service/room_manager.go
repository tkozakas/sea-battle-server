package service

import (
	"crypto/rand"
	"errors"
	"time"

	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/repository"
)

const codeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const codeLength = 6
const maxCodeGenerationAttempts = 1000

var ErrMaxRoomsExceeded = errors.New("max rooms exceeded")

type RoomManager struct {
	repo     repository.GameRepository
	maxRooms int
}

func NewRoomManager(repo repository.GameRepository, maxRooms int) *RoomManager {
	return &RoomManager{repo: repo, maxRooms: maxRooms}
}

func (rm *RoomManager) GenerateCode() (string, error) {
	for i := 0; i < maxCodeGenerationAttempts; i++ {
		code, err := generateRandomCode()
		if err != nil {
			return "", err
		}
		if _, err := rm.repo.FindByID(code); errors.Is(err, domain.ErrGameNotFound) {
			return code, nil
		}
	}
	return "", ErrMaxRoomsExceeded
}

func generateRandomCode() (string, error) {
	b := make([]byte, codeLength)
	charset := []byte(codeCharset)
	random := make([]byte, codeLength)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(random[i])%len(charset)]
	}
	return string(b), nil
}

func (rm *RoomManager) CreateRoom(creatorID string) (string, error) {
	if rm.maxRooms > 0 && rm.repo.Count() >= rm.maxRooms {
		return "", ErrMaxRoomsExceeded
	}
	code, err := rm.GenerateCode()
	if err != nil {
		return "", err
	}
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
		if game.State == domain.StateWaiting && now.Sub(game.CreatedAt) > waitingTimeout {
			toDelete = append(toDelete, code)
			continue
		}
		if game.State == domain.StateGameOver || game.State == domain.StateAbandoned {
			referenceTime := game.FinishedAt
			if referenceTime.IsZero() {
				referenceTime = game.CreatedAt
			}
			if now.Sub(referenceTime) > finishedTimeout {
				toDelete = append(toDelete, code)
			}
		}
	}

	for _, code := range toDelete {
		_ = rm.repo.Delete(code)
	}
}
