package transport_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkozakas/sea-battle-server/internal/repository"
	"github.com/tkozakas/sea-battle-server/internal/service"
	"github.com/tkozakas/sea-battle-server/internal/transport"
)

func newTestHandler() *transport.Handler {
	repo := repository.NewMemoryGameRepository()
	svc := service.NewGameService(repo)
	return transport.NewHandler(svc)
}

func TestHandleHealth(t *testing.T) {
	h := newTestHandler()
	router := transport.NewRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", body["status"])
	}
}

func TestHandleCreateGame(t *testing.T) {
	h := newTestHandler()
	router := transport.NewRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/games?player_id=player1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var msg transport.ServerMessage
	if err := json.NewDecoder(rr.Body).Decode(&msg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if msg.Type != "game_created" {
		t.Errorf("expected type 'game_created', got %s", msg.Type)
	}

	payloadBytes, _ := json.Marshal(msg.Payload)
	var payload transport.GameCreatedMsg
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	if len(payload.GameID) != 6 {
		t.Errorf("expected 6-char game ID, got %s", payload.GameID)
	}
	if payload.PlayerID != "player1" {
		t.Errorf("expected player_id 'player1', got %s", payload.PlayerID)
	}
}

func TestHandleGetGame(t *testing.T) {
	repo := repository.NewMemoryGameRepository()
	svc := service.NewGameService(repo)
	h := transport.NewHandler(svc)
	router := transport.NewRouter(h)

	code, _ := svc.CreateGame("player1")

	req := httptest.NewRequest(http.MethodGet, "/api/games/"+code, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["code"] != code {
		t.Errorf("expected code %s, got %v", code, body["code"])
	}
	if body["state"] != "waiting" {
		t.Errorf("expected state 'waiting', got %v", body["state"])
	}
}

func TestHandleGetGameNotFound(t *testing.T) {
	h := newTestHandler()
	router := transport.NewRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/games/XXXXXX", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}
