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

func TestPlayerDeepCopy(t *testing.T) {
	p := NewPlayer("player-1")
	p.Ready = true
	p.Connected = false

	cp := p.DeepCopy()

	if cp.ID != p.ID {
		t.Errorf("ID mismatch: got %s, want %s", cp.ID, p.ID)
	}
	if cp.Ready != p.Ready {
		t.Errorf("Ready mismatch: got %v, want %v", cp.Ready, p.Ready)
	}
	if cp.Connected != p.Connected {
		t.Errorf("Connected mismatch: got %v, want %v", cp.Connected, p.Connected)
	}
	if cp.Board == p.Board {
		t.Error("DeepCopy should create a new Board, not share the same pointer")
	}
}
