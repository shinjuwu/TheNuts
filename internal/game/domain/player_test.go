package domain

import (
	"errors"
	"testing"
)

// --- SitDown tests ---

func TestPlayerSitDown_Success(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusSittingOut}
	if err := p.SitDown(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.Status != StatusPlaying {
		t.Errorf("expected StatusPlaying, got %v", p.Status)
	}
}

func TestPlayerSitDown_AlreadyPlaying(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusPlaying}
	err := p.SitDown()
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestPlayerSitDown_FromFolded(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusFolded}
	err := p.SitDown()
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

// --- StandUp tests ---

func TestPlayerStandUp_FromPlaying(t *testing.T) {
	p := &Player{
		ID:         "p1",
		Status:     StatusPlaying,
		HoleCards:  []Card{NewCard(RankA, SuitSpade), NewCard(RankK, SuitHeart)},
		CurrentBet: 50,
		HasActed:   false,
	}

	wasInHand, err := p.StandUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !wasInHand {
		t.Error("expected wasInHand=true")
	}
	if p.Status != StatusSittingOut {
		t.Errorf("expected StatusSittingOut, got %v", p.Status)
	}
	if p.HoleCards != nil {
		t.Errorf("expected HoleCards nil, got %v", p.HoleCards)
	}
	if p.CurrentBet != 0 {
		t.Errorf("expected CurrentBet 0, got %d", p.CurrentBet)
	}
	if !p.HasActed {
		t.Error("expected HasActed=true")
	}
}

func TestPlayerStandUp_FromSittingOut(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusSittingOut}

	wasInHand, err := p.StandUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wasInHand {
		t.Error("expected wasInHand=false")
	}
	if p.Status != StatusSittingOut {
		t.Errorf("expected StatusSittingOut, got %v", p.Status)
	}
}

func TestPlayerStandUp_FromFolded(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusFolded}

	wasInHand, err := p.StandUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wasInHand {
		t.Error("expected wasInHand=false")
	}
	if p.Status != StatusSittingOut {
		t.Errorf("expected StatusSittingOut, got %v", p.Status)
	}
}

func TestPlayerStandUp_FromAllIn(t *testing.T) {
	p := &Player{ID: "p1", Status: StatusAllIn}

	_, err := p.StandUp()
	if !errors.Is(err, ErrCannotStandUpWhileAllIn) {
		t.Fatalf("expected ErrCannotStandUpWhileAllIn, got %v", err)
	}
}

// --- Table-level wrapper tests ---

func TestTablePlayerSitDown_NotFound(t *testing.T) {
	table := NewTable("test-table")
	err := table.PlayerSitDown("nonexistent")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}

func TestTablePlayerStandUp_NotFound(t *testing.T) {
	table := NewTable("test-table")
	_, err := table.PlayerStandUp("nonexistent")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}
