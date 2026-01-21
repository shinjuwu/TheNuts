package domain

import (
	"testing"
)

func TestSimpleBettingRound(t *testing.T) {
	// 1. 初始化桌子與玩家
	table := NewTable("test-table")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}

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
	table.Pot = 30

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
	if table.Pot != 60 {
		t.Errorf("Expected Pot 60, got %d", table.Pot)
	}
	// 驗證下注額重置
	if p1.CurrentBet != 0 {
		t.Error("Expected p1 bet reset to 0")
	}
}

func TestFoldLogic(t *testing.T) {
	table := NewTable("test-table")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}

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
