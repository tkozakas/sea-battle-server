package domain

type ExplosionCell struct {
	Point    Point
	Hit      bool
	Sunk     bool
	ShipType ShipType
	Cells    []Point
}

type ShotResult struct {
	Hit        bool
	Sunk       bool
	ShipType   ShipType
	Cells      []Point
	Explosions []ExplosionCell
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
				explosions := explode(b, ship)
				return ShotResult{
					Hit:        true,
					Sunk:       true,
					ShipType:   ship.Type,
					Cells:      ship.Cells(),
					Explosions: explosions,
				}, nil
			}
			b.Grid[p.Y][p.X] = CellHit
			return ShotResult{Hit: true}, nil
		}
	}

	b.Grid[p.Y][p.X] = CellMiss
	return ShotResult{Hit: false}, nil
}

func explode(b *Board, sunkShip *Ship) []ExplosionCell {
	var explosions []ExplosionCell
	queue := []*Ship{sunkShip}
	exploded := make(map[*Ship]bool)
	exploded[sunkShip] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, adj := range current.AdjacentCells() {
			if !inBounds(adj) {
				continue
			}
			cell := b.Grid[adj.Y][adj.X]
			if cell == CellHit || cell == CellMiss || cell == CellSunk {
				continue
			}

			target := b.shipAt(adj)
			if target != nil {
				_ = target.Hit(adj)
				if target.IsSunk() {
					markSunk(b, target)
					explosions = append(explosions, ExplosionCell{
						Point:    adj,
						Hit:      true,
						Sunk:     true,
						ShipType: target.Type,
						Cells:    target.Cells(),
					})
					if !exploded[target] {
						exploded[target] = true
						queue = append(queue, target)
					}
				} else {
					b.Grid[adj.Y][adj.X] = CellHit
					explosions = append(explosions, ExplosionCell{
						Point: adj,
						Hit:   true,
					})
				}
			} else {
				b.Grid[adj.Y][adj.X] = CellMiss
				explosions = append(explosions, ExplosionCell{
					Point: adj,
					Hit:   false,
				})
			}
		}
	}
	return explosions
}

func (b *Board) shipAt(p Point) *Ship {
	for _, ship := range b.Ships {
		if ship.containsCell(p) {
			return ship
		}
	}
	return nil
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
