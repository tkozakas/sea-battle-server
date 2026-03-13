package transport

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
	"github.com/tkozakas/sea-battle-server/internal/config"
	"github.com/tkozakas/sea-battle-server/internal/domain"
	"github.com/tkozakas/sea-battle-server/internal/service"
)

type connEntry struct {
	conn *websocket.Conn
}

type Handler struct {
	service        *service.GameService
	mu             sync.RWMutex
	timerMu        sync.Mutex
	connections    map[string][2]*connEntry
	turnTimers     map[string]*time.Timer
	turnTimeout    time.Duration
	reconnectGrace time.Duration
	pingInterval   time.Duration
	writeTimeout   time.Duration
	allowedOrigins []string
}

func NewHandler(svc *service.GameService, cfg *config.Config) *Handler {
	return &Handler{
		service:        svc,
		connections:    make(map[string][2]*connEntry),
		turnTimers:     make(map[string]*time.Timer),
		turnTimeout:    cfg.TurnTimeout,
		reconnectGrace: cfg.ReconnectGrace,
		pingInterval:   cfg.PingInterval,
		writeTimeout:   cfg.WriteTimeout,
		allowedOrigins: cfg.AllowedOrigins,
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
		var err error
		creatorID, err = generatePlayerID()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
	if errors.Is(err, domain.ErrGameNotFound) {
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

	acceptOptions := &websocket.AcceptOptions{}
	if len(h.allowedOrigins) == 1 && h.allowedOrigins[0] == "*" {
		acceptOptions.OriginPatterns = []string{"*"}
	} else {
		acceptOptions.OriginPatterns = h.allowedOrigins
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		slog.Error("websocket accept failed", "error", err)
		return
	}
	defer func() {
		if err := conn.CloseNow(); err != nil {
			slog.Debug("websocket close failed", "error", err)
		}
	}()

	game, playerIndex, reconnected := h.resolvePlayer(code, playerID)
	if game == nil {
		if err := conn.Close(websocket.StatusNormalClosure, "game not found"); err != nil {
			slog.Debug("websocket close failed", "error", err)
		}
		return
	}

	newJoin := playerIndex == 1 && !reconnected

	h.registerConn(code, playerIndex, conn)
	defer h.unregisterConn(code, playerIndex, conn)

	if newJoin {
		h.broadcast(code, NewServerMessage("game_joined", GameJoinedMsg{
			PlayerID:          playerID,
			OpponentConnected: true,
		}))
	}

	if reconnected {
		reconnectedGame, idx, err := h.service.HandleReconnect(code, playerID)
		if err == nil {
			h.sendTo(code, idx, NewServerMessage("game_state", buildGameStateMsg(reconnectedGame, idx)))
			opponentIndex := 1 - idx
			h.sendTo(code, opponentIndex, NewServerMessage("opponent_reconnected", OpponentReconnectedMsg{}))
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.pingLoop(ctx, conn)

	h.readLoop(ctx, conn, code, playerIndex)
}

func buildGameStateMsg(game *domain.Game, playerIndex int) GameStateMsg {
	opponentIndex := 1 - playerIndex
	var currentTurnDeadline time.Time

	opponentReady := false
	if game.Players[opponentIndex] != nil {
		opponentReady = game.Players[opponentIndex].Ready
	}

	return GameStateMsg{
		State:         string(game.State),
		CurrentTurn:   game.CurrentTurn,
		TurnDeadline:  currentTurnDeadline,
		YourReady:     game.Players[playerIndex].Ready,
		OpponentReady: opponentReady,
	}
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

	game, err := h.service.GetGame(code)
	if err != nil || game.IsOver() {
		return
	}

	h.broadcast(code, NewServerMessage("opponent_disconnected", OpponentDisconnectedMsg{
		ReconnectDeadline: time.Now().Add(h.reconnectGrace),
	}))
}

func (h *Handler) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, h.writeTimeout)
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
	case "request_rematch":
		h.handleRequestRematch(ctx, conn, code, playerIndex)
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
		deadline := time.Now().Add(h.turnTimeout)
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

	target := domain.Point{X: payload.X, Y: payload.Y}
	result, err := h.service.Fire(code, playerIndex, target)
	if err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("error", ErrorMsg{
			Code:    "fire_error",
			Message: err.Error(),
		}))
		return
	}

	h.cancelTurnTimer(code)

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
		deadline = time.Now().Add(h.turnTimeout)
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

func (h *Handler) handleRequestRematch(ctx context.Context, conn *websocket.Conn, code string, playerIndex int) {
	bothReady, err := h.service.RequestRematch(code, playerIndex)
	if err != nil {
		_ = wsjson.Write(ctx, conn, NewServerMessage("error", ErrorMsg{
			Code:    "rematch_error",
			Message: err.Error(),
		}))
		return
	}

	if !bothReady {
		opponentIndex := 1 - playerIndex
		h.sendTo(code, opponentIndex, NewServerMessage("rematch_requested", RematchRequestedMsg{
			PlayerIndex: playerIndex,
		}))
		return
	}

	h.broadcast(code, NewServerMessage("rematch_started", RematchStartedMsg{}))
}

func (h *Handler) startTurnTimer(code string, playerIndex int, deadline time.Time) {
	h.timerMu.Lock()
	defer h.timerMu.Unlock()
	if t, ok := h.turnTimers[code]; ok {
		t.Stop()
		delete(h.turnTimers, code)
	}
	duration := time.Until(deadline)
	timer := time.AfterFunc(duration, func() {
		h.autoFire(code, playerIndex)
	})
	h.turnTimers[code] = timer
}

func (h *Handler) cancelTurnTimer(code string) {
	h.timerMu.Lock()
	defer h.timerMu.Unlock()
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
	validTargets := make([]domain.Point, 0)
	for y := 0; y < domain.BoardSize; y++ {
		for x := 0; x < domain.BoardSize; x++ {
			p := domain.Point{X: x, Y: y}
			if game.Players[opponentIndex].Board.IsValidTarget(p) {
				validTargets = append(validTargets, p)
			}
		}
	}

	if len(validTargets) == 0 {
		return
	}

	idx, err := cryptoRandInt(len(validTargets))
	if err != nil {
		return
	}
	target := validTargets[idx]

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
			ctx, cancel := context.WithTimeout(context.Background(), h.writeTimeout)
			defer cancel()
			if err := wsjson.Write(ctx, entry.conn, msg); err != nil {
				slog.Error("broadcast failed", "playerIndex", i, "error", err)
			}
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

	ctx, cancel := context.WithTimeout(context.Background(), h.writeTimeout)
	defer cancel()
	if err := wsjson.Write(ctx, entry.conn, msg); err != nil {
		slog.Error("sendTo failed", "playerIndex", playerIndex, "error", err)
	}
}

var shipTypeNorm = map[string]domain.ShipType{
	"carrier":    domain.Carrier,
	"battleship": domain.Battleship,
	"cruiser":    domain.Cruiser,
	"submarine":  domain.Submarine,
	"destroyer":  domain.Destroyer,
}

var orientationNorm = map[string]domain.Orientation{
	"horizontal": domain.Horizontal,
	"vertical":   domain.Vertical,
}

func buildShips(placements []ShipPlacement) ([]*domain.Ship, error) {
	ships := make([]*domain.Ship, 0, len(placements))
	for _, p := range placements {
		shipType, ok := shipTypeNorm[strings.ToLower(p.Type)]
		if !ok {
			shipType = domain.ShipType(p.Type)
		}
		orientation, ok := orientationNorm[strings.ToLower(p.Orientation)]
		if !ok {
			orientation = domain.Horizontal
		}
		ship, err := domain.NewShip(shipType, domain.Point{X: p.X, Y: p.Y}, orientation)
		if err != nil {
			return nil, err
		}
		ships = append(ships, ship)
	}
	return ships, nil
}

func generatePlayerID() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, domain.PlayerIDLength)
	for i := range b {
		n, err := cryptoRandInt(len(letters))
		if err != nil {
			return "", err
		}
		b[i] = letters[n]
	}
	return string(b), nil
}

func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
