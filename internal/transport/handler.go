package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/service"
)

const (
	turnTimeout  = 30 * time.Second
	pingInterval = 15 * time.Second
)

type connEntry struct {
	conn *websocket.Conn
}

type Handler struct {
	service     *service.GameService
	mu          sync.RWMutex
	connections map[string][2]*connEntry
	turnTimers  map[string]*time.Timer
}

func NewHandler(svc *service.GameService) *Handler {
	return &Handler{
		service:     svc,
		connections: make(map[string][2]*connEntry),
		turnTimers:  make(map[string]*time.Timer),
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	creatorID := r.URL.Query().Get("player_id")
	if creatorID == "" {
		creatorID = generatePlayerID()
	}

	code, err := h.service.CreateGame(creatorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(NewServerMessage("game_created", GameCreatedMsg{
		GameID:   code,
		PlayerID: creatorID,
	}))
}

func (h *Handler) HandleGetGame(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	game, err := h.service.GetGame(code)
	if err == domain.ErrGameNotFound {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":  game.ID,
		"state": game.State,
	})
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	playerID := r.URL.Query().Get("player_id")

	if code == "" || playerID == "" {
		http.Error(w, "code and player_id required", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("websocket accept failed", "error", err)
		return
	}
	defer conn.CloseNow()

	game, playerIndex, reconnected := h.resolvePlayer(code, playerID)
	if game == nil {
		_ = conn.Close(websocket.StatusNormalClosure, "game not found")
		return
	}

	h.registerConn(code, playerIndex, conn)
	defer h.unregisterConn(code, playerIndex, conn)

	if reconnected {
		_, _, _ = h.service.HandleReconnect(code, playerID)
		h.broadcast(code, NewServerMessage("opponent_reconnected", OpponentReconnectedMsg{}))
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.pingLoop(ctx, conn)

	h.readLoop(ctx, conn, code, playerIndex)
}

func (h *Handler) resolvePlayer(code, playerID string) (*domain.Game, int, bool) {
	game, err := h.service.GetGame(code)
	if err != nil {
		return nil, -1, false
	}

	for i, p := range game.Players {
		if p != nil && p.ID == playerID {
			return game, i, !p.Connected
		}
	}

	if game.Players[1] == nil {
		if err := h.service.JoinGame(code, playerID); err != nil {
			return nil, -1, false
		}
		game, _ = h.service.GetGame(code)
		return game, 1, false
	}

	return nil, -1, false
}

func (h *Handler) registerConn(code string, playerIndex int, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entries := h.connections[code]
	entries[playerIndex] = &connEntry{conn: conn}
	h.connections[code] = entries
}

func (h *Handler) unregisterConn(code string, playerIndex int, conn *websocket.Conn) {
	h.mu.Lock()
	entries := h.connections[code]
	if entries[playerIndex] != nil && entries[playerIndex].conn == conn {
		entries[playerIndex] = nil
	}
	h.connections[code] = entries
	h.mu.Unlock()

	_ = h.service.HandleDisconnect(code, playerIndex)
	h.broadcast(code, NewServerMessage("opponent_disconnected", OpponentDisconnectedMsg{
		ReconnectDeadline: time.Now().Add(60 * time.Second),
	}))
}

func (h *Handler) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Handler) readLoop(ctx context.Context, conn *websocket.Conn, code string, playerIndex int) {
	for {
		var msg ClientMessage
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return
		}
		h.handleClientMessage(ctx, conn, code, playerIndex, msg)
	}
}

func (h *Handler) handleClientMessage(ctx context.Context, conn *websocket.Conn, code string, playerIndex int, msg ClientMessage) {
	switch msg.Type {
	case "place_ships":
		h.handlePlaceShips(ctx, conn, code, playerIndex, msg.Payload)
	case "fire":
		h.handleFire(ctx, conn, code, playerIndex, msg.Payload)
	default:
		_ = wsjson.Write(ctx, conn, NewServerMessage("error", ErrorMsg{
			Code:    "unknown_message",
			Message: "unknown message type: " + msg.Type,
		}))
	}
}

func (h *Handler) handlePlaceShips(ctx context.Context, conn *websocket.Conn, code string, playerIndex int, raw json.RawMessage) {
	var payload PlaceShipsPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("ships_rejected", ShipsRejectedMsg{
			Valid:  false,
			Reason: "invalid payload",
		}))
		return
	}

	ships, err := buildShips(payload.Ships)
	if err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("ships_rejected", ShipsRejectedMsg{
			Valid:  false,
			Reason: err.Error(),
		}))
		return
	}

	if err := h.service.PlaceShips(code, playerIndex, ships); err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("ships_rejected", ShipsRejectedMsg{
			Valid:  false,
			Reason: err.Error(),
		}))
		return
	}

	_ = wsjson.Write(ctx, conn, NewServerMessage("ships_accepted", ShipsAcceptedMsg{Valid: true}))

	game, _ := h.service.GetGame(code)
	if game != nil && game.BothReady() {
		deadline := time.Now().Add(turnTimeout)
		h.broadcast(code, NewServerMessage("both_ready", BothReadyMsg{
			FirstTurn:    game.Players[game.CurrentTurn].ID,
			TurnDeadline: deadline,
		}))
		h.startTurnTimer(code, game.CurrentTurn, deadline)
	}
}

func (h *Handler) handleFire(ctx context.Context, conn *websocket.Conn, code string, playerIndex int, raw json.RawMessage) {
	var payload FirePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("error", ErrorMsg{
			Code:    "invalid_payload",
			Message: "invalid fire payload",
		}))
		return
	}

	h.cancelTurnTimer(code)

	target := domain.Point{X: payload.X, Y: payload.Y}
	result, err := h.service.Fire(code, playerIndex, target)
	if err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("error", ErrorMsg{
			Code:    "fire_error",
			Message: err.Error(),
		}))
		return
	}

	game, _ := h.service.GetGame(code)
	h.broadcastFireResult(code, playerIndex, target, result, game)
}

func (h *Handler) broadcastFireResult(code string, playerIndex int, target domain.Point, result *domain.ShotResult, game *domain.Game) {
	resultStr := "miss"
	if result.Hit {
		resultStr = "hit"
	}
	if result.Sunk {
		resultStr = "sunk"
	}

	var sunkInfo *SunkShipInfo
	if result.Sunk {
		cells := make([]PointMsg, len(result.Cells))
		for i, c := range result.Cells {
			cells[i] = PointMsg{X: c.X, Y: c.Y}
		}
		sunkInfo = &SunkShipInfo{
			Type:  string(result.ShipType),
			Cells: cells,
		}
	}

	if game != nil && game.IsOver() {
		winnerID := ""
		if game.Winner >= 0 && game.Players[game.Winner] != nil {
			winnerID = game.Players[game.Winner].ID
		}
		h.broadcast(code, NewServerMessage("fire_result", FireResultMsg{
			X:            target.X,
			Y:            target.Y,
			Result:       resultStr,
			SunkShip:     sunkInfo,
			NextTurn:     "",
			TurnDeadline: time.Time{},
		}))
		h.broadcast(code, NewServerMessage("game_over", GameOverMsg{
			Winner: winnerID,
			Reason: "all_ships_sunk",
		}))
		return
	}

	nextPlayerID := ""
	deadline := time.Time{}
	if game != nil && game.Players[game.CurrentTurn] != nil {
		nextPlayerID = game.Players[game.CurrentTurn].ID
		deadline = time.Now().Add(turnTimeout)
	}

	h.broadcast(code, NewServerMessage("fire_result", FireResultMsg{
		X:            target.X,
		Y:            target.Y,
		Result:       resultStr,
		SunkShip:     sunkInfo,
		NextTurn:     nextPlayerID,
		TurnDeadline: deadline,
	}))

	if game != nil && !game.IsOver() {
		h.startTurnTimer(code, game.CurrentTurn, deadline)
	}
}

func (h *Handler) startTurnTimer(code string, playerIndex int, deadline time.Time) {
	h.cancelTurnTimer(code)
	duration := time.Until(deadline)
	timer := time.AfterFunc(duration, func() {
		h.autoFire(code, playerIndex)
	})
	h.mu.Lock()
	h.turnTimers[code] = timer
	h.mu.Unlock()
}

func (h *Handler) cancelTurnTimer(code string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.turnTimers[code]; ok {
		t.Stop()
		delete(h.turnTimers, code)
	}
}

func (h *Handler) autoFire(code string, playerIndex int) {
	game, err := h.service.GetGame(code)
	if err != nil || game.IsOver() || game.CurrentTurn != playerIndex {
		return
	}

	opponentIndex := 1 - playerIndex
	var target domain.Point
	found := false
	for y := 0; y < 10 && !found; y++ {
		for x := 0; x < 10 && !found; x++ {
			p := domain.Point{X: x, Y: y}
			if game.Players[opponentIndex].Board.IsValidTarget(p) {
				target = p
				found = true
			}
		}
	}

	if !found {
		return
	}

	validTargets := make([]domain.Point, 0)
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			p := domain.Point{X: x, Y: y}
			if game.Players[opponentIndex].Board.IsValidTarget(p) {
				validTargets = append(validTargets, p)
			}
		}
	}
	target = validTargets[rand.Intn(len(validTargets))]

	result, err := h.service.Fire(code, playerIndex, target)
	if err != nil {
		return
	}

	updatedGame, _ := h.service.GetGame(code)
	h.broadcastFireResult(code, playerIndex, target, result, updatedGame)
}

func (h *Handler) broadcast(code string, msg ServerMessage) {
	h.mu.RLock()
	entries := h.connections[code]
	h.mu.RUnlock()

	for i, entry := range entries {
		if entry != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := wsjson.Write(ctx, entry.conn, msg); err != nil {
				slog.Error("broadcast failed", "playerIndex", i, "error", err)
			}
			cancel()
		}
	}
}

func (h *Handler) sendTo(code string, playerIndex int, msg ServerMessage) {
	h.mu.RLock()
	entries := h.connections[code]
	h.mu.RUnlock()

	entry := entries[playerIndex]
	if entry == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, entry.conn, msg); err != nil {
		slog.Error("sendTo failed", "playerIndex", playerIndex, "error", err)
	}
}

func buildShips(placements []ShipPlacement) ([]*domain.Ship, error) {
	ships := make([]*domain.Ship, 0, len(placements))
	for _, p := range placements {
		orientation := domain.Horizontal
		if p.Orientation == string(domain.Vertical) {
			orientation = domain.Vertical
		}
		ship, err := domain.NewShip(domain.ShipType(p.Type), domain.Point{X: p.X, Y: p.Y}, orientation)
		if err != nil {
			return nil, err
		}
		ships = append(ships, ship)
	}
	return ships, nil
}

func generatePlayerID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
