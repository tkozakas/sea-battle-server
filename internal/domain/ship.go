package domain

type ShipType string

const (
	Patrol     ShipType = "Patrol"
	Frigate    ShipType = "Frigate"
	Cruiser    ShipType = "Cruiser"
	Battleship ShipType = "Battleship"
)

var ShipSize = map[ShipType]int{
	Patrol:     1,
	Frigate:    2,
	Cruiser:    3,
	Battleship: 4,
}

var RequiredShipCounts = map[ShipType]int{
	Patrol:     4,
	Frigate:    2,
	Cruiser:    2,
	Battleship: 1,
}

var TotalShipCount = 9

type Orientation string

const (
	Horizontal Orientation = "Horizontal"
	Vertical   Orientation = "Vertical"
)

type Ship struct {
	Type        ShipType
	Size        int
	Origin      Point
	Orientation Orientation
	cells       []Point
	hitCells    map[Point]bool
}

func NewShip(shipType ShipType, origin Point, orientation Orientation) (*Ship, error) {
	size, ok := ShipSize[shipType]
	if !ok {
		return nil, ErrInvalidPlacement
	}

	s := &Ship{
		Type:        shipType,
		Size:        size,
		Origin:      origin,
		Orientation: orientation,
		hitCells:    make(map[Point]bool),
	}
	s.cells = computeCells(origin, orientation, size)
	return s, nil
}

func computeCells(origin Point, orientation Orientation, size int) []Point {
	cells := make([]Point, size)
	for i := 0; i < size; i++ {
		if orientation == Horizontal {
			cells[i] = Point{X: origin.X + i, Y: origin.Y}
		} else {
			cells[i] = Point{X: origin.X, Y: origin.Y + i}
		}
	}
	return cells
}

func (s *Ship) Cells() []Point {
	result := make([]Point, len(s.cells))
	copy(result, s.cells)
	return result
}

func (s *Ship) Hit(p Point) error {
	if !s.containsCell(p) {
		return ErrOutOfBounds
	}
	if s.hitCells[p] {
		return ErrAlreadyFired
	}
	s.hitCells[p] = true
	return nil
}

func (s *Ship) containsCell(p Point) bool {
	for _, c := range s.cells {
		if c == p {
			return true
		}
	}
	return false
}

func (s *Ship) IsHit(p Point) bool {
	return s.hitCells[p]
}

func (s *Ship) IsSunk() bool {
	return len(s.hitCells) == s.Size
}

func (s *Ship) AdjacentCells() []Point {
	shipSet := make(map[Point]bool)
	for _, c := range s.cells {
		shipSet[c] = true
	}

	neighborSet := make(map[Point]bool)
	for _, c := range s.cells {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				neighbor := Point{X: c.X + dx, Y: c.Y + dy}
				if !shipSet[neighbor] {
					neighborSet[neighbor] = true
				}
			}
		}
	}

	result := make([]Point, 0, len(neighborSet))
	for p := range neighborSet {
		result = append(result, p)
	}
	return result
}

func (s *Ship) DeepCopy() *Ship {
	cells := make([]Point, len(s.cells))
	copy(cells, s.cells)

	hitCells := make(map[Point]bool, len(s.hitCells))
	for k, v := range s.hitCells {
		hitCells[k] = v
	}

	return &Ship{
		Type:        s.Type,
		Size:        s.Size,
		Origin:      s.Origin,
		Orientation: s.Orientation,
		cells:       cells,
		hitCells:    hitCells,
	}
}
