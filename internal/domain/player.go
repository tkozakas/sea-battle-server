package domain

type Player struct {
	ID        string
	Board     *Board
	Ready     bool
	Connected bool
}

func NewPlayer(id string) *Player {
	return &Player{
		ID:        id,
		Board:     NewBoard(),
		Ready:     false,
		Connected: true,
	}
}
