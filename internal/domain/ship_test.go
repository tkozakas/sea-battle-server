package domain

import (
	"sort"
	"testing"
)

func TestNewShip(t *testing.T) {
	tests := []struct {
		name        string
		shipType    ShipType
		origin      Point
		orientation Orientation
		wantSize    int
		wantCells   []Point
	}{
		{
			name:     "Patrol horizontal",
			shipType: Patrol, origin: Point{0, 0}, orientation: Horizontal,
			wantSize:  1,
			wantCells: []Point{{0, 0}},
		},
		{
			name:     "Patrol vertical",
			shipType: Patrol, origin: Point{0, 0}, orientation: Vertical,
			wantSize:  1,
			wantCells: []Point{{0, 0}},
		},
		{
			name:     "Frigate horizontal",
			shipType: Frigate, origin: Point{0, 0}, orientation: Horizontal,
			wantSize:  2,
			wantCells: []Point{{0, 0}, {1, 0}},
		},
		{
			name:     "Frigate vertical",
			shipType: Frigate, origin: Point{0, 0}, orientation: Vertical,
			wantSize:  2,
			wantCells: []Point{{0, 0}, {0, 1}},
		},
		{
			name:     "Cruiser horizontal",
			shipType: Cruiser, origin: Point{0, 0}, orientation: Horizontal,
			wantSize:  3,
			wantCells: []Point{{0, 0}, {1, 0}, {2, 0}},
		},
		{
			name:     "Cruiser vertical",
			shipType: Cruiser, origin: Point{0, 0}, orientation: Vertical,
			wantSize:  3,
			wantCells: []Point{{0, 0}, {0, 1}, {0, 2}},
		},
		{
			name:     "Battleship horizontal",
			shipType: Battleship, origin: Point{0, 0}, orientation: Horizontal,
			wantSize:  4,
			wantCells: []Point{{0, 0}, {1, 0}, {2, 0}, {3, 0}},
		},
		{
			name:     "Battleship vertical",
			shipType: Battleship, origin: Point{0, 0}, orientation: Vertical,
			wantSize:  4,
			wantCells: []Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ship, err := NewShip(tt.shipType, tt.origin, tt.orientation)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ship.Size != tt.wantSize {
				t.Errorf("size = %d, want %d", ship.Size, tt.wantSize)
			}
			cells := ship.Cells()
			if len(cells) != len(tt.wantCells) {
				t.Fatalf("cells length = %d, want %d", len(cells), len(tt.wantCells))
			}
			for i, c := range cells {
				if c != tt.wantCells[i] {
					t.Errorf("cell[%d] = %v, want %v", i, c, tt.wantCells[i])
				}
			}
		})
	}
}

func TestNewShipInvalidType(t *testing.T) {
	_, err := NewShip("InvalidType", Point{0, 0}, Horizontal)
	if err == nil {
		t.Fatal("expected error for invalid ship type, got nil")
	}
}

func TestShipHit(t *testing.T) {
	ship, err := NewShip(Frigate, Point{3, 3}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}
	target := Point{3, 3}
	if err := ship.Hit(target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ship.IsHit(target) {
		t.Error("expected cell to be marked as hit")
	}
}

func TestShipHitAlreadyHit(t *testing.T) {
	ship, err := NewShip(Frigate, Point{3, 3}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}
	target := Point{3, 3}
	_ = ship.Hit(target)
	err = ship.Hit(target)
	if err != ErrAlreadyFired {
		t.Errorf("expected ErrAlreadyFired, got %v", err)
	}
}

func TestShipHitNotOnShip(t *testing.T) {
	ship, err := NewShip(Frigate, Point{3, 3}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}
	err = ship.Hit(Point{9, 9})
	if err == nil {
		t.Fatal("expected error for cell not on ship, got nil")
	}
}

func TestShipIsSunk(t *testing.T) {
	ship, err := NewShip(Frigate, Point{0, 0}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}

	if ship.IsSunk() {
		t.Error("ship should not be sunk before any hits")
	}

	_ = ship.Hit(Point{0, 0})
	if ship.IsSunk() {
		t.Error("ship should not be sunk after one hit on size-2 ship")
	}

	_ = ship.Hit(Point{1, 0})
	if !ship.IsSunk() {
		t.Error("ship should be sunk after all cells hit")
	}
}

func TestShipAdjacentCells(t *testing.T) {
	ship, err := NewShip(Frigate, Point{2, 2}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}
	adjacent := ship.AdjacentCells()

	shipCells := map[Point]bool{
		{2, 2}: true,
		{3, 2}: true,
	}

	expected := map[Point]bool{
		{1, 1}: true, {2, 1}: true, {3, 1}: true, {4, 1}: true,
		{1, 2}: true, {4, 2}: true,
		{1, 3}: true, {2, 3}: true, {3, 3}: true, {4, 3}: true,
	}

	if len(adjacent) != len(expected) {
		t.Errorf("adjacent count = %d, want %d", len(adjacent), len(expected))
	}

	for _, p := range adjacent {
		if shipCells[p] {
			t.Errorf("adjacent cells should not include ship cell %v", p)
		}
		if !expected[p] {
			t.Errorf("unexpected adjacent cell %v", p)
		}
	}

	sortPoints := func(pts []Point) {
		sort.Slice(pts, func(i, j int) bool {
			if pts[i].X != pts[j].X {
				return pts[i].X < pts[j].X
			}
			return pts[i].Y < pts[j].Y
		})
	}

	sortPoints(adjacent)
	for _, p := range adjacent {
		if !expected[p] {
			t.Errorf("unexpected adjacent cell %v", p)
		}
	}
}

func TestShipDeepCopy(t *testing.T) {
	ship, err := NewShip(Frigate, Point{0, 0}, Horizontal)
	if err != nil {
		t.Fatal(err)
	}
	_ = ship.Hit(Point{0, 0})

	cp := ship.DeepCopy()

	if cp.Type != ship.Type {
		t.Errorf("Type mismatch: got %s, want %s", cp.Type, ship.Type)
	}
	if !cp.IsHit(Point{0, 0}) {
		t.Error("copy should preserve hit state")
	}

	_ = cp.Hit(Point{1, 0})
	if ship.IsHit(Point{1, 0}) {
		t.Error("hitting copy should not affect original")
	}
}
