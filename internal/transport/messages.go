package transport

import (
	"encoding/json"
	"time"
)

type ClientMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type PlaceShipsPayload struct {
	Ships []ShipPlacement `json:"ships"`
}

type ShipPlacement struct {
	Type        string `json:"type"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Orientation string `json:"orientation"`
}

type FirePayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

func NewServerMessage(msgType string, payload interface{}) ServerMessage {
	return ServerMessage{Type: msgType, Payload: payload}
}

type GameCreatedMsg struct {
	GameID   string `json:"game_id"`
	PlayerID string `json:"player_id"`
}

type GameJoinedMsg struct {
	PlayerID          string `json:"player_id"`
	OpponentConnected bool   `json:"opponent_connected"`
}

type GameStateMsg struct {
	State         string      `json:"state"`
	YourBoard     interface{} `json:"your_board"`
	OpponentBoard interface{} `json:"opponent_board"`
	YourShips     interface{} `json:"your_ships"`
	CurrentTurn   int         `json:"current_turn"`
	TurnDeadline  time.Time   `json:"turn_deadline"`
	YourReady     bool        `json:"your_ready"`
	OpponentReady bool        `json:"opponent_ready"`
}

type ShipsAcceptedMsg struct {
	Valid bool `json:"valid"`
}

type ShipsRejectedMsg struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

type BothReadyMsg struct {
	FirstTurn    string    `json:"first_turn"`
	TurnDeadline time.Time `json:"turn_deadline"`
}

type SunkShipInfo struct {
	Type  string     `json:"type"`
	Cells []PointMsg `json:"cells"`
}

type PointMsg struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type FireResultMsg struct {
	X            int           `json:"x"`
	Y            int           `json:"y"`
	Result       string        `json:"result"`
	SunkShip     *SunkShipInfo `json:"sunk_ship,omitempty"`
	NextTurn     string        `json:"next_turn"`
	TurnDeadline time.Time     `json:"turn_deadline"`
}

type GameOverMsg struct {
	Winner string `json:"winner"`
	Reason string `json:"reason"`
}

type OpponentDisconnectedMsg struct {
	ReconnectDeadline time.Time `json:"reconnect_deadline"`
}

type OpponentReconnectedMsg struct{}

type OpponentLeftMsg struct {
	Winner string `json:"winner"`
	Reason string `json:"reason"`
}

type ErrorMsg struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
