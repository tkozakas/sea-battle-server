package domain

import "time"

type GameState string

const (
	StateWaiting   GameState = "waiting"
	StatePlacing   GameState = "placing"
	StatePlaying   GameState = "playing"
	StateGameOver  GameState = "game_over"
	StateAbandoned GameState = "abandoned"
)

type Move struct {
	PlayerIndex int
	Target      Point
	Result      ShotResult
	Timestamp   time.Time
}

type Game struct {
	ID          string
	State       GameState
	Players     [2]*Player
	CurrentTurn int
	Winner      int
	MoveHistory []Move
	CreatedAt   time.Time
}

func NewGame(id, creatorID string) *Game {
	g := &Game{
		ID:        id,
		State:     StateWaiting,
		Winner:    -1,
		CreatedAt: time.Now(),
	}
	g.Players[0] = NewPlayer(creatorID)
	return g
}

func (g *Game) Join(playerID string) error {
	if g.Players[1] != nil {
		return ErrGameFull
	}
	g.Players[1] = NewPlayer(playerID)
	g.State = StatePlacing
	return nil
}

func (g *Game) PlaceShips(playerIndex int, ships []*Ship) error {
	if g.State != StatePlacing {
		return ErrInvalidState
	}
	if err := validateShipSet(ships); err != nil {
		return err
	}
	player := g.Players[playerIndex]
	if player.Ready {
		return ErrAlreadyReady
	}
	for _, ship := range ships {
		if err := player.Board.PlaceShip(ship); err != nil {
			return err
		}
	}
	player.Ready = true
	if g.BothReady() {
		g.State = StatePlaying
	}
	return nil
}

func validateShipSet(ships []*Ship) error {
	if len(ships) != len(RequiredShips) {
		return ErrInvalidShipSet
	}
	counts := make(map[ShipType]int)
	for _, s := range ships {
		counts[s.Type]++
	}
	for _, required := range RequiredShips {
		if counts[required] != 1 {
			return ErrInvalidShipSet
		}
	}
	return nil
}

func (g *Game) BothReady() bool {
	return g.Players[0] != nil && g.Players[1] != nil &&
		g.Players[0].Ready && g.Players[1].Ready
}

func (g *Game) Fire(playerIndex int, target Point) (*ShotResult, error) {
	if g.State != StatePlaying {
		return nil, ErrInvalidState
	}
	if g.CurrentTurn != playerIndex {
		return nil, ErrNotYourTurn
	}

	opponentIndex := 1 - playerIndex
	result, err := g.Players[opponentIndex].Board.ReceiveShot(target)
	if err != nil {
		return nil, err
	}

	g.MoveHistory = append(g.MoveHistory, Move{
		PlayerIndex: playerIndex,
		Target:      target,
		Result:      result,
		Timestamp:   time.Now(),
	})

	if g.Players[opponentIndex].Board.AllShipsSunk() {
		g.State = StateGameOver
		g.Winner = playerIndex
		return &result, nil
	}

	if !result.Hit {
		g.CurrentTurn = 1 - g.CurrentTurn
	}

	return &result, nil
}

func (g *Game) IsOver() bool {
	return g.State == StateGameOver || g.State == StateAbandoned
}
