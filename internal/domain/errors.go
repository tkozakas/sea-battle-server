package domain

import "errors"

var (
	ErrGameFull         = errors.New("game is full")
	ErrNotYourTurn      = errors.New("not your turn")
	ErrInvalidState     = errors.New("invalid game state")
	ErrInvalidPlacement = errors.New("invalid ship placement")
	ErrAlreadyFired     = errors.New("already fired at this cell")
	ErrOutOfBounds      = errors.New("target is out of bounds")
	ErrShipOverlap      = errors.New("ships overlap")
	ErrShipAdjacent     = errors.New("ships are adjacent")
	ErrAlreadyReady     = errors.New("player is already ready")
	ErrInvalidShipSet   = errors.New("invalid ship set")
	ErrGameNotFound     = errors.New("game not found")
	ErrAlreadyRequested = errors.New("rematch already requested")
)
