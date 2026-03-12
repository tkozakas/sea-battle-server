package domain

import (
	"testing"
)

func mustShip(t *testing.T, shipType ShipType, x, y int, o Orientation) *Ship {
	t.Helper()
	s, err := NewShip(shipType, Point{x, y}, o)
	if err != nil {
		t.Fatalf("NewShip(%s) failed: %v", shipType, err)
	}
	return s
}

func TestPlaceShip(t *testing.T) {
	b := NewBoard()
	ships := []*Ship{
		mustShip(t, Carrier, 0, 0, Horizontal),
		mustShip(t, Battleship, 0, 2, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Submarine, 0, 6, Horizontal),
		mustShip(t, Destroyer, 0, 8, Horizontal),
	}
	for _, s := range ships {
		if err := b.PlaceShip(s); err != nil {
			t.Errorf("PlaceShip(%s) unexpected error: %v", s.Type, err)
		}
	}
	if len(b.Ships) != 5 {
		t.Errorf("expected 5 ships, got %d", len(b.Ships))
	}
}

func TestPlaceShipOutOfBounds(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Carrier, 8, 0, Horizontal)
	err := b.PlaceShip(ship)
	if err == nil {
		t.Fatal("expected error for out-of-bounds placement")
	}
}

func TestPlaceShipOverlap(t *testing.T) {
	b := NewBoard()
	s1 := mustShip(t, Destroyer, 0, 0, Horizontal)
	s2 := mustShip(t, Destroyer, 0, 0, Horizontal)
	_ = b.PlaceShip(s1)
	err := b.PlaceShip(s2)
	if err == nil {
		t.Fatal("expected error for overlapping ships")
	}
}

func TestPlaceShipAdjacent(t *testing.T) {
	tests := []struct {
		name string
		s1   *Ship
		s2   *Ship
	}{
		{
			name: "adjacent horizontal",
			s1:   &Ship{Type: Destroyer, Size: 2, Origin: Point{0, 0}, Orientation: Horizontal, cells: []Point{{0, 0}, {1, 0}}, hitCells: map[Point]bool{}},
			s2:   &Ship{Type: Destroyer, Size: 2, Origin: Point{0, 1}, Orientation: Horizontal, cells: []Point{{0, 1}, {1, 1}}, hitCells: map[Point]bool{}},
		},
		{
			name: "adjacent diagonal",
			s1:   &Ship{Type: Destroyer, Size: 2, Origin: Point{0, 0}, Orientation: Horizontal, cells: []Point{{0, 0}, {1, 0}}, hitCells: map[Point]bool{}},
			s2:   &Ship{Type: Destroyer, Size: 2, Origin: Point{2, 1}, Orientation: Horizontal, cells: []Point{{2, 1}, {3, 1}}, hitCells: map[Point]bool{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBoard()
			_ = b.PlaceShip(tt.s1)
			err := b.PlaceShip(tt.s2)
			if err == nil {
				t.Fatal("expected error for adjacent ships")
			}
		})
	}
}

func TestReceiveShotMiss(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Destroyer, 0, 0, Horizontal)
	_ = b.PlaceShip(ship)

	result, err := b.ReceiveShot(Point{5, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Hit {
		t.Error("expected miss")
	}
	if b.Grid[5][5] != CellMiss {
		t.Error("expected cell to be marked as miss")
	}
}

func TestReceiveShotHit(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Destroyer, 3, 3, Horizontal)
	_ = b.PlaceShip(ship)

	result, err := b.ReceiveShot(Point{3, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Hit {
		t.Error("expected hit")
	}
	if result.Sunk {
		t.Error("ship should not be sunk yet")
	}
	if b.Grid[3][3] != CellHit {
		t.Error("expected cell to be marked as hit")
	}
}

func TestReceiveShotSunk(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Destroyer, 3, 3, Horizontal)
	_ = b.PlaceShip(ship)

	_, _ = b.ReceiveShot(Point{3, 3})
	result, err := b.ReceiveShot(Point{4, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Hit {
		t.Error("expected hit")
	}
	if !result.Sunk {
		t.Error("expected ship to be sunk")
	}
	if result.ShipType != Destroyer {
		t.Errorf("expected ShipType = Destroyer, got %s", result.ShipType)
	}
	if len(result.Cells) != 2 {
		t.Errorf("expected 2 sunk cells, got %d", len(result.Cells))
	}
	if b.Grid[3][3] != CellSunk || b.Grid[3][4] != CellSunk {
		t.Error("expected cells to be marked as sunk")
	}
}

func TestReceiveShotAlreadyFired(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Destroyer, 0, 0, Horizontal)
	_ = b.PlaceShip(ship)

	_, _ = b.ReceiveShot(Point{5, 5})
	_, err := b.ReceiveShot(Point{5, 5})
	if err != ErrAlreadyFired {
		t.Errorf("expected ErrAlreadyFired, got %v", err)
	}
}

func TestReceiveShotOutOfBounds(t *testing.T) {
	b := NewBoard()
	tests := []Point{{-1, 0}, {10, 5}, {0, -1}, {5, 10}}
	for _, p := range tests {
		_, err := b.ReceiveShot(p)
		if err != ErrOutOfBounds {
			t.Errorf("point %v: expected ErrOutOfBounds, got %v", p, err)
		}
	}
}

func TestAllShipsSunk(t *testing.T) {
	b := NewBoard()
	ships := []*Ship{
		mustShip(t, Carrier, 0, 0, Horizontal),
		mustShip(t, Battleship, 0, 2, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Submarine, 0, 6, Horizontal),
		mustShip(t, Destroyer, 0, 8, Horizontal),
	}
	for _, s := range ships {
		_ = b.PlaceShip(s)
	}

	if b.AllShipsSunk() {
		t.Error("ships should not be sunk initially")
	}

	for _, s := range ships {
		for _, c := range s.Cells() {
			_, err := b.ReceiveShot(c)
			if err != nil {
				t.Fatalf("ReceiveShot(%v) unexpected error: %v", c, err)
			}
		}
	}

	if !b.AllShipsSunk() {
		t.Error("all ships should be sunk")
	}
}

func TestBoardDeepCopy(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Destroyer, 0, 0, Horizontal)
	_ = b.PlaceShip(ship)
	_, _ = b.ReceiveShot(Point{0, 0})

	cp := b.DeepCopy()

	if cp.Grid[0][0] != b.Grid[0][0] {
		t.Error("copy grid should match original")
	}

	cp.Grid[0][0] = CellMiss
	if b.Grid[0][0] == CellMiss {
		t.Error("modifying copy grid should not affect original")
	}

	if len(cp.Ships) != len(b.Ships) {
		t.Errorf("copy ship count = %d, want %d", len(cp.Ships), len(b.Ships))
	}

	_ = cp.Ships[0].Hit(Point{1, 0})
	if b.Ships[0].IsHit(Point{1, 0}) {
		t.Error("hitting copy ship should not affect original")
	}
}
