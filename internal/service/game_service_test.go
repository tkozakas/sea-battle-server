package service_test

import (
	"testing"

	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/repository"
	"github.com/tkozakas/sea-battle-server/internal/service"
)

func newGameService() *service.GameService {
	repo := repository.NewMemoryGameRepository()
	rooms := service.NewRoomManager(repo, 1000)
	return service.NewGameService(repo, rooms)
}

func validShips(rowOffset int) []*domain.Ship {
	placements := []struct {
		t domain.ShipType
		x int
		y int
	}{
		{domain.Carrier, 0, rowOffset + 0},
		{domain.Battleship, 0, rowOffset + 2},
		{domain.Cruiser, 0, rowOffset + 4},
		{domain.Submarine, 0, rowOffset + 6},
		{domain.Destroyer, 0, rowOffset + 8},
	}
	ships := make([]*domain.Ship, 0, len(placements))
	for _, p := range placements {
		s, err := domain.NewShip(p.t, domain.Point{X: p.x, Y: p.y}, domain.Horizontal)
		if err != nil {
			panic(err)
		}
		ships = append(ships, s)
	}
	return ships
}

func TestCreateGame(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected code length 6, got %d", len(code))
	}
}

func TestJoinGame(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.JoinGame(code, "player2"); err != nil {
		t.Fatalf("unexpected error joining game: %v", err)
	}

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatalf("unexpected error getting game: %v", err)
	}
	if game.Players[1] == nil {
		t.Error("expected player 2 to be set")
	}
	if game.Players[1].ID != "player2" {
		t.Errorf("expected player2 ID, got %s", game.Players[1].ID)
	}
}

func TestJoinGameNotFound(t *testing.T) {
	svc := newGameService()

	err := svc.JoinGame("XXXXXX", "player2")
	if err != domain.ErrGameNotFound {
		t.Errorf("expected ErrGameNotFound, got %v", err)
	}
}

func TestPlaceShips(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.JoinGame(code, "player2")

	ships := validShips(0)
	if err := svc.PlaceShips(code, 0, ships); err != nil {
		t.Fatalf("unexpected error placing ships: %v", err)
	}

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if !game.Players[0].Ready {
		t.Error("expected player 0 to be ready")
	}
}

func TestFire(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.JoinGame(code, "player2")
	_ = svc.PlaceShips(code, 0, validShips(0))
	_ = svc.PlaceShips(code, 1, validShips(0))

	result, err := svc.Fire(code, 0, domain.Point{X: 9, Y: 9})
	if err != nil {
		t.Fatalf("unexpected error firing: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestFireMissSwitchesTurn(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.JoinGame(code, "player2")
	_ = svc.PlaceShips(code, 0, validShips(0))
	_ = svc.PlaceShips(code, 1, validShips(0))

	result, err := svc.Fire(code, 0, domain.Point{X: 9, Y: 9})
	if err != nil {
		t.Fatal(err)
	}
	if result.Hit {
		t.Fatal("expected a miss for this test")
	}

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if game.CurrentTurn != 1 {
		t.Errorf("expected turn to switch to player 1, got %d", game.CurrentTurn)
	}
}

func TestFireHitSameTurn(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.JoinGame(code, "player2")
	_ = svc.PlaceShips(code, 0, validShips(0))
	_ = svc.PlaceShips(code, 1, validShips(0))

	result, err := svc.Fire(code, 0, domain.Point{X: 0, Y: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Hit {
		t.Fatal("expected a hit at (0,0) where Carrier starts")
	}

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if game.CurrentTurn != 0 {
		t.Errorf("expected turn to stay at 0 after hit, got %d", game.CurrentTurn)
	}
}

func TestFullGameFlow(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.JoinGame(code, "player2")
	_ = svc.PlaceShips(code, 0, validShips(0))
	_ = svc.PlaceShips(code, 1, validShips(0))

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if game.State != domain.StatePlaying {
		t.Fatalf("expected playing state, got %s", game.State)
	}

	targets := allShipCells(validShips(0))
	playerTurn := 0

	for {
		g, err := svc.GetGame(code)
		if err != nil {
			t.Fatal(err)
		}
		if g.IsOver() {
			break
		}

		var target domain.Point
		found := false
		for _, pt := range targets {
			if g.Players[1-playerTurn].Board.IsValidTarget(pt) {
				target = pt
				found = true
				break
			}
		}
		if !found {
			t.Fatal("no valid targets remaining but game not over")
		}

		result, err := svc.Fire(code, playerTurn, target)
		if err != nil {
			t.Fatalf("error firing: %v", err)
		}
		if !result.Hit {
			playerTurn = 1 - playerTurn
		}
	}

	final, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if final.State != domain.StateGameOver {
		t.Errorf("expected game over, got %s", final.State)
	}
}

func allShipCells(ships []*domain.Ship) []domain.Point {
	var pts []domain.Point
	for _, s := range ships {
		pts = append(pts, s.Cells()...)
	}
	return pts
}

func TestHandleDisconnect(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.HandleDisconnect(code, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	game, err := svc.GetGame(code)
	if err != nil {
		t.Fatal(err)
	}
	if game.Players[0].Connected {
		t.Error("expected player 0 to be disconnected")
	}
}

func TestHandleReconnect(t *testing.T) {
	svc := newGameService()
	code, err := svc.CreateGame("player1")
	if err != nil {
		t.Fatal(err)
	}
	_ = svc.HandleDisconnect(code, 0)

	game, idx, err := svc.HandleReconnect(code, "player1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 0 {
		t.Errorf("expected player index 0, got %d", idx)
	}
	if !game.Players[0].Connected {
		t.Error("expected player 0 to be connected after reconnect")
	}
}
