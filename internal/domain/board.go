package domain

type ShotResult struct {
	Hit      bool
	Sunk     bool
	ShipType ShipType
	Cells    []Point
}

type Board struct {
	Grid  [BoardSize][BoardSize]CellState
	Ships []*Ship
}

func NewBoard() *Board {
	return &Board{}
}

func (b *Board) PlaceShip(ship *Ship) error {
	if !b.CanPlaceShip(ship) {
		return ErrInvalidPlacement
	}
	for _, p := range ship.Cells() {
		b.Grid[p.Y][p.X] = CellShip
	}
	b.Ships = append(b.Ships, ship)
	return nil
}

func (b *Board) CanPlaceShip(ship *Ship) bool {
	for _, p := range ship.Cells() {
		if !inBounds(p) {
			return false
		}
		if b.Grid[p.Y][p.X] == CellShip {
			return false
		}
	}
	for _, adj := range ship.AdjacentCells() {
		if inBounds(adj) && b.Grid[adj.Y][adj.X] == CellShip {
			return false
		}
	}
	return true
}

func (b *Board) ReceiveShot(p Point) (ShotResult, error) {
	if !inBounds(p) {
		return ShotResult{}, ErrOutOfBounds
	}
	if !b.IsValidTarget(p) {
		return ShotResult{}, ErrAlreadyFired
	}

	for _, ship := range b.Ships {
		if ship.containsCell(p) {
			_ = ship.Hit(p)
			if ship.IsSunk() {
				markSunk(b, ship)
				return ShotResult{
					Hit:      true,
					Sunk:     true,
					ShipType: ship.Type,
					Cells:    ship.Cells(),
				}, nil
			}
			b.Grid[p.Y][p.X] = CellHit
			return ShotResult{Hit: true}, nil
		}
	}

	b.Grid[p.Y][p.X] = CellMiss
	return ShotResult{Hit: false}, nil
}

func markSunk(b *Board, ship *Ship) {
	for _, c := range ship.Cells() {
		b.Grid[c.Y][c.X] = CellSunk
	}
}

func (b *Board) AllShipsSunk() bool {
	for _, ship := range b.Ships {
		if !ship.IsSunk() {
			return false
		}
	}
	return len(b.Ships) > 0
}

func (b *Board) IsValidTarget(p Point) bool {
	if !inBounds(p) {
		return false
	}
	state := b.Grid[p.Y][p.X]
	return state != CellHit && state != CellMiss && state != CellSunk
}

func inBounds(p Point) bool {
	return p.X >= 0 && p.X < BoardSize && p.Y >= 0 && p.Y < BoardSize
}

func (b *Board) DeepCopy() *Board {
	nb := &Board{
		Grid: b.Grid,
	}
	nb.Ships = make([]*Ship, len(b.Ships))
	for i, s := range b.Ships {
		nb.Ships[i] = s.DeepCopy()
	}
	return nb
}
