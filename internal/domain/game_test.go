package domain

import (
	"testing"
)

func validShipSet(t *testing.T) []*Ship {
	t.Helper()
	return []*Ship{
		mustShip(t, Carrier, 0, 0, Horizontal),
		mustShip(t, Battleship, 0, 2, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Submarine, 0, 6, Horizontal),
		mustShip(t, Destroyer, 0, 8, Horizontal),
	}
}

func setupPlayingGame(t *testing.T) *Game {
	t.Helper()
	g := NewGame("game-1", "player-0")
	if err := g.Join("player-1"); err != nil {
		t.Fatalf("Join failed: %v", err)
	}
	if err := g.PlaceShips(0, validShipSet(t)); err != nil {
		t.Fatalf("PlaceShips(0) failed: %v", err)
	}
	if err := g.PlaceShips(1, validShipSet(t)); err != nil {
		t.Fatalf("PlaceShips(1) failed: %v", err)
	}
	if g.State != StatePlaying {
		t.Fatalf("expected StatePlaying, got %s", g.State)
	}
	return g
}

func TestNewGame(t *testing.T) {
	g := NewGame("g1", "creator")

	if g.ID != "g1" {
		t.Errorf("ID = %q, want %q", g.ID, "g1")
	}
	if g.State != StateWaiting {
		t.Errorf("State = %q, want %q", g.State, StateWaiting)
	}
	if g.Players[0] == nil {
		t.Error("Players[0] should not be nil")
	}
	if g.Players[1] != nil {
		t.Error("Players[1] should be nil")
	}
	if g.Winner != -1 {
		t.Errorf("Winner = %d, want -1", g.Winner)
	}
	if g.CurrentTurn != 0 {
		t.Errorf("CurrentTurn = %d, want 0", g.CurrentTurn)
	}
}

func TestGameJoin(t *testing.T) {
	g := NewGame("g1", "creator")
	err := g.Join("joiner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.State != StatePlacing {
		t.Errorf("State = %q, want %q", g.State, StatePlacing)
	}
	if g.Players[1] == nil {
		t.Error("Players[1] should not be nil after join")
	}
	if g.Players[1].ID != "joiner" {
		t.Errorf("Players[1].ID = %q, want %q", g.Players[1].ID, "joiner")
	}
}

func TestGameJoinFull(t *testing.T) {
	g := NewGame("g1", "creator")
	_ = g.Join("joiner")
	err := g.Join("extra")
	if err != ErrGameFull {
		t.Errorf("expected ErrGameFull, got %v", err)
	}
}

func TestGameJoinWrongState(t *testing.T) {
	g := NewGame("g1", "creator")
	g.State = StatePlaying
	err := g.Join("joiner")
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState when joining non-waiting game, got %v", err)
	}
}

func TestGamePlaceShips(t *testing.T) {
	g := NewGame("g1", "p0")
	_ = g.Join("p1")

	if err := g.PlaceShips(0, validShipSet(t)); err != nil {
		t.Fatalf("PlaceShips(0) unexpected error: %v", err)
	}
	if g.Players[0].Ready != true {
		t.Error("player 0 should be ready")
	}
	if g.BothReady() {
		t.Error("should not be both ready yet")
	}

	if err := g.PlaceShips(1, validShipSet(t)); err != nil {
		t.Fatalf("PlaceShips(1) unexpected error: %v", err)
	}
	if !g.BothReady() {
		t.Error("should be both ready")
	}
	if g.State != StatePlaying {
		t.Errorf("State = %q, want %q", g.State, StatePlaying)
	}
}

func TestGamePlaceShipsWrongState(t *testing.T) {
	g := NewGame("g1", "p0")
	err := g.PlaceShips(0, validShipSet(t))
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestGamePlaceShipsRollbackOnInvalidPlacement(t *testing.T) {
	g := NewGame("g1", "p0")
	_ = g.Join("p1")

	overlappingShips := []*Ship{
		mustShip(t, Carrier, 0, 0, Horizontal),
		mustShip(t, Battleship, 0, 0, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Submarine, 0, 6, Horizontal),
		mustShip(t, Destroyer, 0, 8, Horizontal),
	}

	err := g.PlaceShips(0, overlappingShips)
	if err == nil {
		t.Fatal("expected error for overlapping ships, got nil")
	}

	err = g.PlaceShips(0, validShipSet(t))
	if err != nil {
		t.Fatalf("expected valid placement to succeed after failed attempt, got: %v", err)
	}
}

func TestGameFire(t *testing.T) {
	g := setupPlayingGame(t)
	target := Point{9, 9}
	result, err := g.Fire(0, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Hit {
		t.Error("expected miss at (9,9)")
	}
	if g.CurrentTurn != 1 {
		t.Errorf("CurrentTurn = %d, want 1 after miss", g.CurrentTurn)
	}
}

func TestGameFireHit(t *testing.T) {
	g := setupPlayingGame(t)
	target := Point{0, 0}
	result, err := g.Fire(0, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Hit {
		t.Error("expected hit at (0,0)")
	}
	if g.CurrentTurn != 0 {
		t.Errorf("CurrentTurn = %d, want 0 after hit", g.CurrentTurn)
	}
}

func TestGameFireSunk(t *testing.T) {
	g := setupPlayingGame(t)

	if _, err := g.Fire(0, Point{0, 8}); err != nil {
		t.Fatal(err)
	}
	result, err := g.Fire(0, Point{1, 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Sunk {
		t.Error("expected ship to be sunk")
	}
	if result.ShipType != Destroyer {
		t.Errorf("expected Destroyer sunk, got %s", result.ShipType)
	}
}

func TestGameFireNotYourTurn(t *testing.T) {
	g := setupPlayingGame(t)
	_, err := g.Fire(1, Point{5, 5})
	if err != ErrNotYourTurn {
		t.Errorf("expected ErrNotYourTurn, got %v", err)
	}
}

func TestGameFireWrongState(t *testing.T) {
	g := NewGame("g1", "p0")
	_ = g.Join("p1")
	_, err := g.Fire(0, Point{0, 0})
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestGameFullGame(t *testing.T) {
	g := setupPlayingGame(t)

	allShipCells := []Point{}
	shipPlacements := []struct {
		shipType ShipType
		x, y     int
	}{
		{Carrier, 0, 0},
		{Battleship, 0, 2},
		{Cruiser, 0, 4},
		{Submarine, 0, 6},
		{Destroyer, 0, 8},
	}
	for _, sp := range shipPlacements {
		s, err := NewShip(sp.shipType, Point{sp.x, sp.y}, Horizontal)
		if err != nil {
			t.Fatal(err)
		}
		allShipCells = append(allShipCells, s.Cells()...)
	}

	for i, cell := range allShipCells {
		if g.IsOver() {
			break
		}
		result, err := g.Fire(0, cell)
		if err != nil {
			t.Fatalf("Fire[%d] at %v failed: %v", i, cell, err)
		}
		_ = result
	}

	if !g.IsOver() {
		t.Error("game should be over after sinking all ships")
	}
	if g.State != StateGameOver {
		t.Errorf("State = %q, want %q", g.State, StateGameOver)
	}
	if g.Winner != 0 {
		t.Errorf("Winner = %d, want 0", g.Winner)
	}
	if g.FinishedAt.IsZero() {
		t.Error("FinishedAt should be set when game is over")
	}
}

func setupGameOver(t *testing.T) *Game {
	t.Helper()
	g := setupPlayingGame(t)
	for _, cell := range validShipSet(t) {
		for _, pt := range cell.Cells() {
			if g.IsOver() {
				break
			}
			if _, err := g.Fire(0, pt); err != nil {
				t.Fatalf("Fire failed: %v", err)
			}
		}
	}
	if g.State != StateGameOver {
		t.Fatalf("expected StateGameOver, got %s", g.State)
	}
	return g
}

func TestRequestRematch(t *testing.T) {
	g := setupGameOver(t)

	bothReady, err := g.RequestRematch(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bothReady {
		t.Error("expected false when only one player requested")
	}

	bothReady, err = g.RequestRematch(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bothReady {
		t.Error("expected true when both players requested")
	}
}

func TestRequestRematchWrongState(t *testing.T) {
	g := setupPlayingGame(t)

	_, err := g.RequestRematch(0)
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestStartRematch(t *testing.T) {
	g := setupGameOver(t)
	_, _ = g.RequestRematch(0)
	_, _ = g.RequestRematch(1)

	g.StartRematch()

	if g.State != StatePlacing {
		t.Errorf("State = %q, want %q", g.State, StatePlacing)
	}
	if g.Winner != -1 {
		t.Errorf("Winner = %d, want -1", g.Winner)
	}
	if len(g.MoveHistory) != 0 {
		t.Errorf("MoveHistory should be empty, got %d entries", len(g.MoveHistory))
	}
	if g.RematchRequests[0] || g.RematchRequests[1] {
		t.Error("RematchRequests should be cleared")
	}
	for i, p := range g.Players {
		if p == nil {
			continue
		}
		if p.Ready {
			t.Errorf("Players[%d] should not be ready", i)
		}
		for y := range p.Board.Grid {
			for x, cell := range p.Board.Grid[y] {
				if cell != CellEmpty {
					t.Errorf("Players[%d].Board.Grid[%d][%d] = %v, want CellEmpty", i, y, x, cell)
				}
			}
		}
	}
}

func TestGameDeepCopy(t *testing.T) {
	g := setupPlayingGame(t)

	cp := g.DeepCopy()

	if cp.ID != g.ID {
		t.Errorf("ID mismatch: got %s, want %s", cp.ID, g.ID)
	}
	if cp.State != g.State {
		t.Errorf("State mismatch")
	}

	cp.Players[0].Board.Grid[0][0] = CellHit
	if g.Players[0].Board.Grid[0][0] == CellHit {
		t.Error("modifying copy's board should not affect original")
	}

	cp.State = StateGameOver
	if g.State == StateGameOver {
		t.Error("modifying copy's state should not affect original")
	}
}
