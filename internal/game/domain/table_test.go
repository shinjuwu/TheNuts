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

// TestDealerRotationWithFoldedPlayer 測試 Dealer Button 應該移動到下一個有籌碼的玩家
// 即使該玩家當前狀態是 StatusFolded
func TestDealerRotationWithFoldedPlayer(t *testing.T) {
	table := NewTable("dealer-rotation-test")

	// 設置 3 個玩家在座位 0, 1, 2
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusFolded} // Folded 但有籌碼
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	// 設置當前 Dealer 在座位 0
	table.DealerPos = 0

	// 調用 rotateDealerButton
	table.rotateDealerButton()

	// 預期結果：Dealer 應該移動到座位 1（即使 p2 是 Folded）
	// 因為 p2 有籌碼且不是 SittingOut，下一手牌時 p2 會重置為 Playing
	if table.DealerPos != 1 {
		t.Errorf("Expected dealer at seat 1, got seat %d. "+
			"Dealer should rotate to next player with chips, regardless of Folded status",
			table.DealerPos)
	}
}

// TestDealerRotationSkipsSittingOut 測試 Dealer Button 應該跳過 SittingOut 的玩家
func TestDealerRotationSkipsSittingOut(t *testing.T) {
	table := NewTable("dealer-rotation-test-2")

	// 設置 3 個玩家在座位 0, 1, 2
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusSittingOut} // 暫離
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	// 設置當前 Dealer 在座位 0
	table.DealerPos = 0

	// 調用 rotateDealerButton
	table.rotateDealerButton()

	// 預期結果：Dealer 應該跳過座位 1（SittingOut），移動到座位 2
	if table.DealerPos != 2 {
		t.Errorf("Expected dealer at seat 2 (skipping SittingOut player), got seat %d",
			table.DealerPos)
	}
}

// TestDealerRotationSkipsNoChips 測試 Dealer Button 應該跳過沒有籌碼的玩家
func TestDealerRotationSkipsNoChips(t *testing.T) {
	table := NewTable("dealer-rotation-test-3")

	// 設置 3 個玩家在座位 0, 1, 2
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 0, Status: StatusPlaying} // 沒籌碼
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	// 設置當前 Dealer 在座位 0
	table.DealerPos = 0

	// 調用 rotateDealerButton
	table.rotateDealerButton()

	// 預期結果：Dealer 應該跳過座位 1（沒籌碼），移動到座位 2
	if table.DealerPos != 2 {
		t.Errorf("Expected dealer at seat 2 (skipping no-chips player), got seat %d",
			table.DealerPos)
	}
}

// TestProcessCommand_JoinTable 測試透過 processCommand 加入玩家
func TestProcessCommand_JoinTable(t *testing.T) {
	table := NewTable("cmd-test")

	player := &Player{
		ID:      "p1",
		SeatIdx: 0,
		Chips:   1000,
		Status:  StatusSittingOut,
	}

	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionJoinTable,
		Player:   player,
		SeatIdx:  0,
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}

	if table.Players["p1"] == nil {
		t.Error("Expected player to be added to Players map")
	}
	if table.Seats[0] == nil {
		t.Error("Expected player to be added to Seats[0]")
	}
}

// TestProcessCommand_JoinTable_SeatOccupied 測試座位被佔用時的錯誤處理
func TestProcessCommand_JoinTable_SeatOccupied(t *testing.T) {
	table := NewTable("cmd-test")

	// 先加入一個玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 嘗試加入另一個玩家到同一座位
	p2 := &Player{ID: "p2", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionJoinTable,
		Player:   p2,
		SeatIdx:  0,
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err == nil {
		t.Error("Expected error for occupied seat")
	}
	if table.Players["p2"] != nil {
		t.Error("p2 should not be added when seat is occupied")
	}
}

// TestProcessCommand_JoinTable_DuplicatePlayer 測試玩家重複加入的錯誤處理
func TestProcessCommand_JoinTable_DuplicatePlayer(t *testing.T) {
	table := NewTable("cmd-test")

	// 先加入玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 嘗試再次加入同一個玩家ID（不同座位）
	p1dup := &Player{ID: "p1", SeatIdx: 1, Chips: 1000, Status: StatusSittingOut}
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionJoinTable,
		Player:   p1dup,
		SeatIdx:  1,
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err == nil {
		t.Error("Expected error for duplicate player")
	}
}

// TestProcessCommand_LeaveTable 測試透過 processCommand 離開桌子
func TestProcessCommand_LeaveTable(t *testing.T) {
	table := NewTable("cmd-test")

	// 先加入玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 離開桌子
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionLeaveTable,
		PlayerID: "p1",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}

	if table.Players["p1"] != nil {
		t.Error("Expected player to be removed from Players map")
	}
	if table.Seats[0] != nil {
		t.Error("Expected player to be removed from Seats[0]")
	}
}

// TestProcessCommand_LeaveTable_WhilePlaying 測試 Playing 狀態離桌（自動 Fold）
func TestProcessCommand_LeaveTable_WhilePlaying(t *testing.T) {
	table := NewTable("cmd-test")

	// 加入 Playing 狀態的玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 離開桌子
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionLeaveTable,
		PlayerID: "p1",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != nil {
		t.Errorf("Expected no error (auto fold), got %v", result.Err)
	}

	if table.Players["p1"] != nil {
		t.Error("Expected player to be removed after auto-fold")
	}
}

// TestProcessCommand_LeaveTable_WhileAllIn 測試 AllIn 狀態離桌（應該被拒絕）
func TestProcessCommand_LeaveTable_WhileAllIn(t *testing.T) {
	table := NewTable("cmd-test")

	// 加入 AllIn 狀態的玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 0, Status: StatusAllIn}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 嘗試離開桌子
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionLeaveTable,
		PlayerID: "p1",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err == nil {
		t.Error("Expected error for AllIn player trying to leave")
	}

	if table.Players["p1"] == nil {
		t.Error("AllIn player should not be removed")
	}
}

// TestProcessCommand_LeaveTable_NotFound 測試玩家不在桌上時的錯誤處理
func TestProcessCommand_LeaveTable_NotFound(t *testing.T) {
	table := NewTable("cmd-test")

	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionLeaveTable,
		PlayerID: "nonexistent",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != ErrPlayerNotFound {
		t.Errorf("Expected ErrPlayerNotFound, got %v", result.Err)
	}
}

// TestProcessCommand_SitDown 測試透過 processCommand 坐下
func TestProcessCommand_SitDown(t *testing.T) {
	table := NewTable("cmd-test")

	// 加入 SittingOut 狀態的玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 坐下
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionSitDown,
		PlayerID: "p1",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}

	if p1.Status != StatusPlaying {
		t.Errorf("Expected status Playing, got %v", p1.Status)
	}
}

// TestProcessCommand_StandUp 測試透過 processCommand 站起
func TestProcessCommand_StandUp(t *testing.T) {
	table := NewTable("cmd-test")

	// 加入 Playing 狀態的玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p1.HoleCards = []Card{NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade)}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 站起
	resultCh := make(chan ActionResult, 1)
	table.processCommand(PlayerAction{
		Type:     ActionStandUp,
		PlayerID: "p1",
		ResultCh: resultCh,
	})

	result := <-resultCh
	if result.Err != nil {
		t.Errorf("Expected no error, got %v", result.Err)
	}

	if !result.WasInHand {
		t.Error("Expected WasInHand to be true for Playing status")
	}

	if p1.Status != StatusSittingOut {
		t.Errorf("Expected status SittingOut, got %v", p1.Status)
	}
}
