package domain

import (
	"testing"
)

func TestSimpleBettingRound(t *testing.T) {
	// 1. 初始化桌子與玩家
	table := NewTable("test-table")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	// 設定初始狀態: PreFlop, BTN=0, SB=1, BB=2
	table.State = StatePreFlop
	table.DealerPos = 0
	table.CurrentPos = 0 // PreFlop 第一個行動的是 BTN (3人桌)

	// 模擬盲注
	table.MinBet = 20
	p2.CurrentBet = 10 // SB
	p2.Chips -= 10
	p3.CurrentBet = 20 // BB
	p3.Chips -= 20
	// table.Pots will be updated when round ends, so initially it's 0 or we manually accumulate for setup?
	// For simple test, we rely on handleAction triggering nextStreet to accumulate.
	// But lines 26-29 set up initial bets. `handleAction` will add MORE bets.
	// We should just let the test flow naturally or update setup.
	// In the test: "BTN Call, SB Call, BB Check".
	// The blind bets are ALREADY on the table in this setup?
	// Yes, lines 26-30 simulate setup.
	// If we want `Pot` to reflect blinds, we should manually Accumulate them or just ignore initial pot check if not relevant.
	// Previous code: table.Pot = 30.
	// New code:
	// table.Pots.Accumulate(initialBlindBets) -> This sets up the pot.
	// But let's just remove direct table.Pot = 30.
	// We want to verify `StatePreFlop` -> `StateFlop` transition which triggers `nextStreet` and accumulation.
	// So we don't need to manually set Pot if we trust nextStreet.
	// But wait, the test validates `table.Pot == 60` after the round.
	// The initial 30 + 30 from calls = 60.
	// So YES, we need to correctly handle the initial blinds.
	// Let's just remove explicit `table.Pot = 30` line. The Accumulate in nextStreet will pick up ALL bets (including blinds) IF `CurrentBet` is set correctly on players.
	// Lines 26-29 set p2.CurrentBet=10, p3.CurrentBet=20. p1(BTN) will bet 20.
	// When nextStreet is called, it will sum: p1(20) + p2(20) + p3(20) = 60.
	// So removing `table.Pot = 30` is correct because `nextStreet` sums `CurrentBet`.

	// 2. 測試流程: BTN Call -> SB Call -> BB Check

	// BTN Call 20
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall})
	if p1.CurrentBet != 20 {
		t.Errorf("Expected p1 bet 20, got %d", p1.CurrentBet)
	}
	if table.CurrentPos != 1 {
		t.Errorf("Expected current pos 1 (SB), got %d", table.CurrentPos)
	}

	// SB Call 20
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCall})
	if p2.CurrentBet != 20 {
		t.Errorf("Expected p2 bet 20, got %d", p2.CurrentBet)
	}

	// BB Check
	table.handleAction(PlayerAction{PlayerID: "p3", Type: ActionCheck})

	// 3. 驗證是否進入 Flop
	if table.State != StateFlop {
		t.Errorf("Expected StateFlop, got %v", table.State)
	}
	if table.Pots.Total() != 60 {
		t.Errorf("Expected Pot 60, got %d", table.Pots.Total())
	}
	// 驗證下注額重置
	if p1.CurrentBet != 0 {
		t.Error("Expected p1 bet reset to 0")
	}
}

func TestFoldLogic(t *testing.T) {
	table := NewTable("test-table")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// P1 Fold
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionFold})
	if p1.Status != StatusFolded {
		t.Error("Expected p1 folded")
	}
	if table.CurrentPos != 1 {
		t.Errorf("Expected next pos 1, got %d", table.CurrentPos)
	}
}
