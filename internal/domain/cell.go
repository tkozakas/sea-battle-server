package domain

type CellState int

const (
	CellEmpty CellState = 0
	CellShip  CellState = 1
	CellHit   CellState = 2
	CellMiss  CellState = 3
	CellSunk  CellState = 4
)
