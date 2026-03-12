package domain

import "testing"

func TestNewPlayer(t *testing.T) {
	p := NewPlayer("player-1")

	if p.ID != "player-1" {
		t.Errorf("ID = %q, want %q", p.ID, "player-1")
	}
	if p.Board == nil {
		t.Error("Board should not be nil")
	}
	if p.Ready {
		t.Error("Ready should be false")
	}
	if !p.Connected {
		t.Error("Connected should be true")
	}
}
