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
			Type: "Destroyer",
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
	if payload.SunkShip.Type != "Destroyer" {
		t.Errorf("expected Destroyer, got %s", payload.SunkShip.Type)
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
