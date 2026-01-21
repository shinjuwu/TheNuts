package domain

import (
	"testing"
)

func TestPotManager_Accumulate_MainPotOnly(t *testing.T) {
	pm := NewPotManager()
	bets := map[string]int64{
		"p1": 50,
		"p2": 50,
		"p3": 50,
	}

	pm.Accumulate(bets)

	if len(pm.Pots) != 1 {
		t.Fatalf("Expected 1 pot, got %d", len(pm.Pots))
	}
	if pm.Pots[0].Amount != 150 {
		t.Errorf("Expected pot amount 150, got %d", pm.Pots[0].Amount)
	}
	if len(pm.Pots[0].Contributors) != 3 {
		t.Errorf("Expected 3 contributors, got %d", len(pm.Pots[0].Contributors))
	}
}

func TestPotManager_Accumulate_SidePots(t *testing.T) {
	// Case: P1 All-in 100, P2 All-in 200, P3 Covers (Bet 500)
	// Expectation:
	// Pot 1 (Main): 300 (100 from each) -> Contributors: P1, P2, P3
	// Pot 2 (Side): 200 (100 from P2, 100 from P3) -> Contributors: P2, P3
	// Pot 3 (Side): 300 (300 from P3) -> Contributors: P3

	pm := NewPotManager()
	bets := map[string]int64{
		"p1": 100,
		"p2": 200,
		"p3": 500,
	}

	pm.Accumulate(bets)

	if len(pm.Pots) != 3 {
		t.Fatalf("Expected 3 pots, got %d", len(pm.Pots))
	}

	// Verify Pot 1
	if pm.Pots[0].Amount != 300 {
		t.Errorf("Pot 1 amount mismatch: want 300, got %d", pm.Pots[0].Amount)
	}
	if len(pm.Pots[0].Contributors) != 3 {
		t.Errorf("Pot 1 contributors mismatch: want 3, got %d", len(pm.Pots[0].Contributors))
	}

	// Verify Pot 2
	if pm.Pots[1].Amount != 200 {
		t.Errorf("Pot 2 amount mismatch: want 200, got %d", pm.Pots[1].Amount)
	}
	if len(pm.Pots[1].Contributors) != 2 {
		t.Errorf("Pot 2 contributors mismatch: want 2, got %d", len(pm.Pots[1].Contributors))
	}
	if pm.Pots[1].Contributors["p1"] {
		t.Error("P1 should not be in Pot 2")
	}

	// Verify Pot 3
	if pm.Pots[2].Amount != 300 {
		t.Errorf("Pot 3 amount mismatch: want 300, got %d", pm.Pots[2].Amount)
	}
	if len(pm.Pots[2].Contributors) != 1 {
		t.Errorf("Pot 3 contributors mismatch: want 1, got %d", len(pm.Pots[2].Contributors))
	}
	if !pm.Pots[2].Contributors["p3"] {
		t.Error("P3 should be in Pot 3")
	}
}

func TestPotManager_Accumulate_Merge(t *testing.T) {
	// 模擬多輪下注
	pm := NewPotManager()

	// Round 1: Everyone bets 10
	pm.Accumulate(map[string]int64{"p1": 10, "p2": 10})

	// Round 2: Everyone bets 20 more
	pm.Accumulate(map[string]int64{"p1": 20, "p2": 20})

	if len(pm.Pots) != 1 {
		t.Fatalf("Expected pots to merge into 1, got %d", len(pm.Pots))
	}
	if pm.Total() != 60 {
		t.Errorf("Expected total 60, got %d", pm.Total())
	}
}
