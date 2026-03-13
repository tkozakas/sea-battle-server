package transport_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tkozakas/sea-battle-server/internal/transport"
)

func TestClientMessageFirePayloadMarshalUnmarshal(t *testing.T) {
	fire := transport.FirePayload{X: 3, Y: 7}
	rawPayload, err := json.Marshal(fire)
	if err != nil {
		t.Fatalf("failed to marshal fire payload: %v", err)
	}

	msg := transport.ClientMessage{
		Type:    "fire",
		Payload: rawPayload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal client message: %v", err)
	}

	var decoded transport.ClientMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal client message: %v", err)
	}

	if decoded.Type != "fire" {
		t.Errorf("expected type 'fire', got %s", decoded.Type)
	}

	var fp transport.FirePayload
	if err := json.Unmarshal(decoded.Payload, &fp); err != nil {
		t.Fatalf("failed to unmarshal fire payload: %v", err)
	}

	if fp.X != 3 || fp.Y != 7 {
		t.Errorf("expected X=3 Y=7, got X=%d Y=%d", fp.X, fp.Y)
	}
}

func TestServerMessageGameCreatedMarshalUnmarshal(t *testing.T) {
	msg := transport.NewServerMessage("game_created", transport.GameCreatedMsg{
		GameID:   "ABC123",
		PlayerID: "player-uuid",
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal server message: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	var msgType string
	if err := json.Unmarshal(raw["type"], &msgType); err != nil {
		t.Fatalf("failed to unmarshal type: %v", err)
	}
	if msgType != "game_created" {
		t.Errorf("expected type 'game_created', got %s", msgType)
	}

	var payload transport.GameCreatedMsg
	if err := json.Unmarshal(raw["payload"], &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.GameID != "ABC123" {
		t.Errorf("expected GameID 'ABC123', got %s", payload.GameID)
	}
	if payload.PlayerID != "player-uuid" {
		t.Errorf("expected PlayerID 'player-uuid', got %s", payload.PlayerID)
	}
}

func TestServerMessageFireResultWithSunkShip(t *testing.T) {
	deadline := time.Now().Add(30 * time.Second).UTC().Truncate(time.Second)
	msg := transport.NewServerMessage("fire_result", transport.FireResultMsg{
		X:      2,
		Y:      5,
		Result: "sunk",
		SunkShip: &transport.SunkShipInfo{
			Type: "Frigate",
			Cells: []transport.PointMsg{
				{X: 2, Y: 5},
				{X: 3, Y: 5},
			},
		},
		NextTurn:     "player1",
		TurnDeadline: deadline,
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	var payload transport.FireResultMsg
	if err := json.Unmarshal(raw["payload"], &payload); err != nil {
		t.Fatalf("failed to unmarshal fire result: %v", err)
	}

	if payload.Result != "sunk" {
		t.Errorf("expected result 'sunk', got %s", payload.Result)
	}
	if payload.SunkShip == nil {
		t.Fatal("expected sunk ship info, got nil")
	}
	if payload.SunkShip.Type != "Frigate" {
		t.Errorf("expected Frigate, got %s", payload.SunkShip.Type)
	}
	if len(payload.SunkShip.Cells) != 2 {
		t.Errorf("expected 2 cells, got %d", len(payload.SunkShip.Cells))
	}
}

func TestServerMessageFireResultWithoutSunkShip(t *testing.T) {
	msg := transport.NewServerMessage("fire_result", transport.FireResultMsg{
		X:        1,
		Y:        1,
		Result:   "miss",
		SunkShip: nil,
		NextTurn: "player2",
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	var payload transport.FireResultMsg
	if err := json.Unmarshal(raw["payload"], &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if payload.SunkShip != nil {
		t.Error("expected nil SunkShip for miss")
	}
	if payload.Result != "miss" {
		t.Errorf("expected 'miss', got %s", payload.Result)
	}
}

func TestGameStateMsgWithBoardDataSerializes(t *testing.T) {
	var yourBoard [10][10]int
	yourBoard[0][0] = 1
	yourBoard[0][1] = 1
	yourBoard[5][3] = 2
	yourBoard[7][8] = 3
	yourBoard[9][9] = 4

	var oppBoard [10][10]int
	oppBoard[2][4] = 1
	oppBoard[3][3] = 2
	oppBoard[6][6] = 3

	ships := []transport.ShipInfo{
		{Type: "patrol", X: 0, Y: 0, Orientation: "horizontal"},
		{Type: "frigate", X: 3, Y: 5, Orientation: "vertical"},
	}

	msg := transport.NewServerMessage("game_state", transport.GameStateMsg{
		State:         "playing",
		YourBoard:     yourBoard,
		OpponentBoard: oppBoard,
		YourShips:     ships,
		CurrentTurn:   0,
		YourReady:     true,
		OpponentReady: true,
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	var payload transport.GameStateMsg
	if err := json.Unmarshal(raw["payload"], &payload); err != nil {
		t.Fatalf("failed to unmarshal game state: %v", err)
	}

	if payload.State != "playing" {
		t.Errorf("expected state 'playing', got %s", payload.State)
	}
	if payload.YourBoard[0][0] != 1 {
		t.Errorf("expected your_board[0][0]=1, got %d", payload.YourBoard[0][0])
	}
	if payload.YourBoard[5][3] != 2 {
		t.Errorf("expected your_board[5][3]=2, got %d", payload.YourBoard[5][3])
	}
	if payload.YourBoard[9][9] != 4 {
		t.Errorf("expected your_board[9][9]=4, got %d", payload.YourBoard[9][9])
	}
	if payload.OpponentBoard[2][4] != 1 {
		t.Errorf("expected opponent_board[2][4]=1, got %d", payload.OpponentBoard[2][4])
	}
	if payload.OpponentBoard[3][3] != 2 {
		t.Errorf("expected opponent_board[3][3]=2, got %d", payload.OpponentBoard[3][3])
	}
	if len(payload.YourShips) != 2 {
		t.Fatalf("expected 2 ships, got %d", len(payload.YourShips))
	}
	if payload.YourShips[0].Type != "patrol" {
		t.Errorf("expected ship type 'patrol', got %s", payload.YourShips[0].Type)
	}
	if payload.YourShips[1].Orientation != "vertical" {
		t.Errorf("expected orientation 'vertical', got %s", payload.YourShips[1].Orientation)
	}
	if !payload.YourReady {
		t.Error("expected your_ready=true")
	}
	if !payload.OpponentReady {
		t.Error("expected opponent_ready=true")
	}
}

func TestGameStateMsgEmptyShipsSerializesAsEmptyArray(t *testing.T) {
	msg := transport.NewServerMessage("game_state", transport.GameStateMsg{
		State:     "placing",
		YourShips: []transport.ShipInfo{},
	})

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	var payload transport.GameStateMsg
	if err := json.Unmarshal(raw["payload"], &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if payload.YourShips == nil {
		t.Error("expected non-nil ships slice")
	}
	if len(payload.YourShips) != 0 {
		t.Errorf("expected 0 ships, got %d", len(payload.YourShips))
	}
}
