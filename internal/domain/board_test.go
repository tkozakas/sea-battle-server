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
		mustShip(t, Battleship, 0, 0, Horizontal),
		mustShip(t, Cruiser, 0, 2, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Frigate, 0, 6, Horizontal),
		mustShip(t, Frigate, 5, 6, Horizontal),
		mustShip(t, Patrol, 0, 8, Horizontal),
		mustShip(t, Patrol, 2, 8, Horizontal),
		mustShip(t, Patrol, 4, 8, Horizontal),
		mustShip(t, Patrol, 6, 8, Horizontal),
	}
	for _, s := range ships {
		if err := b.PlaceShip(s); err != nil {
			t.Errorf("PlaceShip(%s) unexpected error: %v", s.Type, err)
		}
	}
	if len(b.Ships) != 9 {
		t.Errorf("expected 9 ships, got %d", len(b.Ships))
	}
}

func TestPlaceShipOutOfBounds(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Battleship, 8, 0, Horizontal)
	err := b.PlaceShip(ship)
	if err == nil {
		t.Fatal("expected error for out-of-bounds placement")
	}
}

func TestPlaceShipOverlap(t *testing.T) {
	b := NewBoard()
	s1 := mustShip(t, Frigate, 0, 0, Horizontal)
	s2 := mustShip(t, Frigate, 0, 0, Horizontal)
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
			s1:   &Ship{Type: Frigate, Size: 2, Origin: Point{0, 0}, Orientation: Horizontal, cells: []Point{{0, 0}, {1, 0}}, hitCells: map[Point]bool{}},
			s2:   &Ship{Type: Frigate, Size: 2, Origin: Point{0, 1}, Orientation: Horizontal, cells: []Point{{0, 1}, {1, 1}}, hitCells: map[Point]bool{}},
		},
		{
			name: "adjacent diagonal",
			s1:   &Ship{Type: Frigate, Size: 2, Origin: Point{0, 0}, Orientation: Horizontal, cells: []Point{{0, 0}, {1, 0}}, hitCells: map[Point]bool{}},
			s2:   &Ship{Type: Frigate, Size: 2, Origin: Point{2, 1}, Orientation: Horizontal, cells: []Point{{2, 1}, {3, 1}}, hitCells: map[Point]bool{}},
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
	ship := mustShip(t, Frigate, 0, 0, Horizontal)
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
	ship := mustShip(t, Frigate, 3, 3, Horizontal)
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
	ship := mustShip(t, Frigate, 3, 3, Horizontal)
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
	if result.ShipType != Frigate {
		t.Errorf("expected ShipType = Frigate, got %s", result.ShipType)
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
	ship := mustShip(t, Frigate, 0, 0, Horizontal)
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
		mustShip(t, Battleship, 0, 0, Horizontal),
		mustShip(t, Cruiser, 0, 2, Horizontal),
		mustShip(t, Cruiser, 0, 4, Horizontal),
		mustShip(t, Frigate, 0, 6, Horizontal),
		mustShip(t, Frigate, 5, 6, Horizontal),
		mustShip(t, Patrol, 0, 8, Horizontal),
		mustShip(t, Patrol, 2, 8, Horizontal),
		mustShip(t, Patrol, 4, 8, Horizontal),
		mustShip(t, Patrol, 6, 8, Horizontal),
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

func containsExplosionPoint(explosions []ExplosionCell, p Point) bool {
	for _, e := range explosions {
		if e.Point == p {
			return true
		}
	}
	return false
}

func sinkShip(t *testing.T, b *Board, ship *Ship) ShotResult {
	t.Helper()
	var result ShotResult
	for _, c := range ship.Cells() {
		var err error
		result, err = b.ReceiveShot(c)
		if err != nil {
			t.Fatalf("ReceiveShot(%v) unexpected error: %v", c, err)
		}
	}
	return result
}

func placeShipDirectly(b *Board, ship *Ship) {
	for _, p := range ship.Cells() {
		b.Grid[p.Y][p.X] = CellShip
	}
	b.Ships = append(b.Ships, ship)
}

func TestExplosionRevealsAdjacentCells(t *testing.T) {
	b := NewBoard()
	patrol := mustShip(t, Patrol, 5, 5, Horizontal)
	_ = b.PlaceShip(patrol)

	result := sinkShip(t, b, patrol)

	if len(result.Explosions) == 0 {
		t.Fatal("expected explosions after sinking ship")
	}

	expectedMisses := []Point{
		{4, 4}, {5, 4}, {6, 4},
		{4, 5}, {6, 5},
		{4, 6}, {5, 6}, {6, 6},
	}

	for _, p := range expectedMisses {
		if !containsExplosionPoint(result.Explosions, p) {
			t.Errorf("expected explosion at %v", p)
		}
		if b.Grid[p.Y][p.X] != CellMiss {
			t.Errorf("expected cell %v to be CellMiss, got %v", p, b.Grid[p.Y][p.X])
		}
	}

	if len(result.Explosions) != len(expectedMisses) {
		t.Errorf("expected %d explosions, got %d", len(expectedMisses), len(result.Explosions))
	}
}

func TestExplosionAtCorner(t *testing.T) {
	b := NewBoard()
	patrol := mustShip(t, Patrol, 0, 0, Horizontal)
	_ = b.PlaceShip(patrol)

	result := sinkShip(t, b, patrol)

	expectedMisses := []Point{{0, 1}, {1, 0}, {1, 1}}
	if len(result.Explosions) != len(expectedMisses) {
		t.Errorf("expected %d explosions at corner, got %d", len(expectedMisses), len(result.Explosions))
	}

	for _, p := range expectedMisses {
		if !containsExplosionPoint(result.Explosions, p) {
			t.Errorf("expected explosion at %v", p)
		}
	}

	outOfBounds := []Point{{-1, -1}, {-1, 0}, {0, -1}, {1, -1}, {-1, 1}}
	for _, p := range outOfBounds {
		if containsExplosionPoint(result.Explosions, p) {
			t.Errorf("out-of-bounds point %v should not be in explosions", p)
		}
	}
}

func TestExplosionAtEdge(t *testing.T) {
	b := NewBoard()
	patrol := mustShip(t, Patrol, 0, 5, Horizontal)
	_ = b.PlaceShip(patrol)

	result := sinkShip(t, b, patrol)

	expectedMisses := []Point{{0, 4}, {1, 4}, {1, 5}, {0, 6}, {1, 6}}
	if len(result.Explosions) != len(expectedMisses) {
		t.Errorf("expected %d explosions at edge, got %d", len(expectedMisses), len(result.Explosions))
	}

	for _, p := range expectedMisses {
		if !containsExplosionPoint(result.Explosions, p) {
			t.Errorf("expected explosion at %v", p)
		}
	}
}

func TestExplosionSkipsAlreadyFiredCells(t *testing.T) {
	b := NewBoard()
	frigate := mustShip(t, Frigate, 3, 3, Horizontal)
	_ = b.PlaceShip(frigate)

	_, err := b.ReceiveShot(Point{2, 3})
	if err != nil {
		t.Fatalf("ReceiveShot unexpected error: %v", err)
	}

	result := sinkShip(t, b, frigate)

	if containsExplosionPoint(result.Explosions, Point{2, 3}) {
		t.Error("already-fired cell (2,3) should not appear in explosions")
	}
}

func TestExplosionOnLargerShip(t *testing.T) {
	b := NewBoard()
	battleship := mustShip(t, Battleship, 3, 3, Horizontal)
	_ = b.PlaceShip(battleship)

	result := sinkShip(t, b, battleship)

	expectedMisses := []Point{
		{2, 2}, {3, 2}, {4, 2}, {5, 2}, {6, 2}, {7, 2},
		{2, 3}, {7, 3},
		{2, 4}, {3, 4}, {4, 4}, {5, 4}, {6, 4}, {7, 4},
	}

	if len(result.Explosions) != len(expectedMisses) {
		t.Errorf("expected %d explosion cells around battleship, got %d", len(expectedMisses), len(result.Explosions))
	}

	for _, p := range expectedMisses {
		if !containsExplosionPoint(result.Explosions, p) {
			t.Errorf("expected explosion at %v", p)
		}
		if b.Grid[p.Y][p.X] != CellMiss {
			t.Errorf("expected cell %v to be CellMiss", p)
		}
	}
}

func TestExplosionChainReaction(t *testing.T) {
	b := NewBoard()

	frigate := &Ship{
		Type:        Frigate,
		Size:        2,
		Origin:      Point{3, 3},
		Orientation: Horizontal,
		cells:       []Point{{3, 3}, {4, 3}},
		hitCells:    make(map[Point]bool),
	}
	patrol := &Ship{
		Type:        Patrol,
		Size:        1,
		Origin:      Point{3, 4},
		Orientation: Horizontal,
		cells:       []Point{{3, 4}},
		hitCells:    make(map[Point]bool),
	}

	placeShipDirectly(b, frigate)
	placeShipDirectly(b, patrol)

	result := sinkShip(t, b, frigate)

	patrolSunkInExplosion := false
	for _, e := range result.Explosions {
		if e.Point == (Point{3, 4}) && e.Hit && e.Sunk && e.ShipType == Patrol {
			patrolSunkInExplosion = true
		}
	}
	if !patrolSunkInExplosion {
		t.Error("expected patrol at (3,4) to be sunk by chain explosion")
	}

	if b.Grid[3][4] != CellSunk {
		t.Errorf("expected patrol cell (3,4) to be CellSunk, got %v", b.Grid[3][4])
	}
}

func TestExplosionNoExplosionsOnHit(t *testing.T) {
	b := NewBoard()
	frigate := mustShip(t, Frigate, 3, 3, Horizontal)
	_ = b.PlaceShip(frigate)

	result, err := b.ReceiveShot(Point{3, 3})
	if err != nil {
		t.Fatalf("ReceiveShot unexpected error: %v", err)
	}

	if result.Explosions != nil {
		t.Errorf("expected nil explosions on hit-but-not-sunk, got %v", result.Explosions)
	}
}

func TestExplosionNoExplosionsOnMiss(t *testing.T) {
	b := NewBoard()

	result, err := b.ReceiveShot(Point{5, 5})
	if err != nil {
		t.Fatalf("ReceiveShot unexpected error: %v", err)
	}

	if result.Explosions != nil {
		t.Errorf("expected nil explosions on miss, got %v", result.Explosions)
	}
}

func TestShipAtFindsCorrectShip(t *testing.T) {
	b := NewBoard()
	frigate := mustShip(t, Frigate, 0, 0, Horizontal)
	patrol := mustShip(t, Patrol, 5, 5, Horizontal)
	_ = b.PlaceShip(frigate)
	_ = b.PlaceShip(patrol)

	found := b.shipAt(Point{0, 0})
	if found != frigate {
		t.Error("expected shipAt(0,0) to return frigate")
	}

	found = b.shipAt(Point{1, 0})
	if found != frigate {
		t.Error("expected shipAt(1,0) to return frigate")
	}

	found = b.shipAt(Point{5, 5})
	if found != patrol {
		t.Error("expected shipAt(5,5) to return patrol")
	}

	found = b.shipAt(Point{3, 3})
	if found != nil {
		t.Errorf("expected shipAt(3,3) to return nil, got %v", found)
	}
}

func TestExplosionAllShipsSunkAfterChain(t *testing.T) {
	b := NewBoard()

	frigate := &Ship{
		Type:        Frigate,
		Size:        2,
		Origin:      Point{4, 4},
		Orientation: Horizontal,
		cells:       []Point{{4, 4}, {5, 4}},
		hitCells:    make(map[Point]bool),
	}
	patrol1 := &Ship{
		Type:        Patrol,
		Size:        1,
		Origin:      Point{4, 5},
		Orientation: Horizontal,
		cells:       []Point{{4, 5}},
		hitCells:    make(map[Point]bool),
	}
	patrol2 := &Ship{
		Type:        Patrol,
		Size:        1,
		Origin:      Point{5, 5},
		Orientation: Horizontal,
		cells:       []Point{{5, 5}},
		hitCells:    make(map[Point]bool),
	}

	placeShipDirectly(b, frigate)
	placeShipDirectly(b, patrol1)
	placeShipDirectly(b, patrol2)

	if b.AllShipsSunk() {
		t.Fatal("ships should not all be sunk before any shots")
	}

	sinkShip(t, b, frigate)

	if !b.AllShipsSunk() {
		t.Error("expected all ships to be sunk after chain explosion")
	}
}

func TestBoardDeepCopy(t *testing.T) {
	b := NewBoard()
	ship := mustShip(t, Frigate, 0, 0, Horizontal)
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
