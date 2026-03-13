package transport

import (
	"testing"

	"github.com/tkozakas/sea-battle-server/internal/domain"
)

func TestBuildShipsLowercaseShipType(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "patrol", X: 0, Y: 0, Orientation: "horizontal"},
	}

	ships, err := buildShips(placements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ships) != 1 {
		t.Fatalf("expected 1 ship, got %d", len(ships))
	}
	if ships[0].Type != domain.Patrol {
		t.Errorf("expected Patrol, got %s", ships[0].Type)
	}
}

func TestBuildShipsAllLowercaseShipTypes(t *testing.T) {
	cases := []struct {
		input    string
		expected domain.ShipType
	}{
		{"patrol", domain.Patrol},
		{"frigate", domain.Frigate},
		{"cruiser", domain.Cruiser},
		{"battleship", domain.Battleship},
	}

	for _, tc := range cases {
		placements := []ShipPlacement{
			{Type: tc.input, X: 0, Y: 0, Orientation: "horizontal"},
		}
		ships, err := buildShips(placements)
		if err != nil {
			t.Errorf("type %q: unexpected error: %v", tc.input, err)
			continue
		}
		if ships[0].Type != tc.expected {
			t.Errorf("type %q: expected %s, got %s", tc.input, tc.expected, ships[0].Type)
		}
	}
}

func TestBuildShipsCapitalizedShipTypeStillWorks(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "Patrol", X: 0, Y: 0, Orientation: "Horizontal"},
	}

	ships, err := buildShips(placements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ships[0].Type != domain.Patrol {
		t.Errorf("expected Patrol, got %s", ships[0].Type)
	}
}

func TestBuildShipsLowercaseHorizontalOrientation(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "frigate", X: 0, Y: 0, Orientation: "horizontal"},
	}

	ships, err := buildShips(placements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ships[0].Orientation != domain.Horizontal {
		t.Errorf("expected Horizontal, got %s", ships[0].Orientation)
	}
}

func TestBuildShipsLowercaseVerticalOrientation(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "frigate", X: 0, Y: 0, Orientation: "vertical"},
	}

	ships, err := buildShips(placements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ships[0].Orientation != domain.Vertical {
		t.Errorf("expected Vertical, got %s", ships[0].Orientation)
	}
}

func TestBuildShipsUnknownShipTypeReturnsError(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "dreadnought", X: 0, Y: 0, Orientation: "horizontal"},
	}

	_, err := buildShips(placements)
	if err == nil {
		t.Error("expected error for unknown ship type, got nil")
	}
}

func TestBuildShipsMultiplePlacements(t *testing.T) {
	placements := []ShipPlacement{
		{Type: "battleship", X: 0, Y: 0, Orientation: "horizontal"},
		{Type: "frigate", X: 0, Y: 9, Orientation: "vertical"},
	}

	ships, err := buildShips(placements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ships) != 2 {
		t.Fatalf("expected 2 ships, got %d", len(ships))
	}
	if ships[0].Type != domain.Battleship {
		t.Errorf("expected Battleship, got %s", ships[0].Type)
	}
	if ships[1].Type != domain.Frigate {
		t.Errorf("expected Frigate, got %s", ships[1].Type)
	}
}
