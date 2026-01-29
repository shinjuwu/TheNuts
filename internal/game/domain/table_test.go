package domain

import (
	"testing"
	"time"
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
	if err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall}); err != nil {
		t.Fatalf("Expected no error for p1 call, got %v", err)
	}
	if p1.CurrentBet != 20 {
		t.Errorf("Expected p1 bet 20, got %d", p1.CurrentBet)
	}
	if table.CurrentPos != 1 {
		t.Errorf("Expected current pos 1 (SB), got %d", table.CurrentPos)
	}

	// SB Call 20
	if err := table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCall}); err != nil {
		t.Fatalf("Expected no error for p2 call, got %v", err)
	}
	if p2.CurrentBet != 20 {
		t.Errorf("Expected p2 bet 20, got %d", p2.CurrentBet)
	}

	// BB Check
	if err := table.handleAction(PlayerAction{PlayerID: "p3", Type: ActionCheck}); err != nil {
		t.Fatalf("Expected no error for p3 check, got %v", err)
	}

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
	if err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionFold}); err != nil {
		t.Fatalf("Expected no error for p1 fold, got %v", err)
	}
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

// === 斷線/重連測試 ===

// TestDisconnect_RecordTime 送 ActionDisconnect → 驗證 DisconnectedAt 有記錄
func TestDisconnect_RecordTime(t *testing.T) {
	table := NewTable("dc-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	before := time.Now()
	table.processCommand(PlayerAction{
		Type:     ActionDisconnect,
		PlayerID: "p1",
	})

	dcTime, exists := table.DisconnectedAt["p1"]
	if !exists {
		t.Fatal("Expected DisconnectedAt entry for p1")
	}
	if dcTime.Before(before) {
		t.Error("DisconnectedAt should be >= test start time")
	}
}

// TestDisconnect_PlayerNotFound 不存在的玩家斷線 → 不 panic，無記錄
func TestDisconnect_PlayerNotFound(t *testing.T) {
	table := NewTable("dc-test")

	// Should not panic
	table.processCommand(PlayerAction{
		Type:     ActionDisconnect,
		PlayerID: "nonexistent",
	})

	if len(table.DisconnectedAt) != 0 {
		t.Error("Expected no DisconnectedAt entries for nonexistent player")
	}
}

// TestReconnect_ClearDisconnect 送 ActionReconnect → 驗證 DisconnectedAt 被清除
func TestReconnect_ClearDisconnect(t *testing.T) {
	table := NewTable("dc-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	// 先斷線
	table.processCommand(PlayerAction{
		Type:     ActionDisconnect,
		PlayerID: "p1",
	})
	if _, exists := table.DisconnectedAt["p1"]; !exists {
		t.Fatal("Expected DisconnectedAt entry after disconnect")
	}

	// 再重連
	table.processCommand(PlayerAction{
		Type:     ActionReconnect,
		PlayerID: "p1",
	})
	if _, exists := table.DisconnectedAt["p1"]; exists {
		t.Error("Expected DisconnectedAt cleared after reconnect")
	}
}

// TestDisconnectTimeout_AutoFoldCurrentPlayer 斷線玩家輪到行動 + 超時 → 自動 Fold + StandUp
func TestDisconnectTimeout_AutoFoldCurrentPlayer(t *testing.T) {
	table := NewTable("dc-test")
	table.DisconnectTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Seats[0] = p1
	table.Seats[1] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0 // p1's turn
	table.MinBet = 20

	// 記錄斷線
	table.DisconnectedAt["p1"] = time.Now().Add(-1 * time.Second) // 已過超時

	// 觸發超時檢查
	table.checkDisconnectTimeouts()

	// p1 應該被自動 Fold 並 StandUp
	if p1.Status != StatusSittingOut {
		t.Errorf("Expected p1 status SittingOut after timeout, got %v", p1.Status)
	}
	if _, exists := table.DisconnectedAt["p1"]; exists {
		t.Error("Expected DisconnectedAt cleared after timeout")
	}
}

// TestDisconnectTimeout_NotCurrentPlayer 斷線玩家不是當前行動者 → 超時後僅 StandUp
func TestDisconnectTimeout_NotCurrentPlayer(t *testing.T) {
	table := NewTable("dc-test")
	table.DisconnectTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Seats[0] = p1
	table.Seats[1] = p2

	table.State = StatePreFlop
	table.CurrentPos = 1 // p2's turn, NOT p1
	table.MinBet = 20

	// p1 斷線超時
	table.DisconnectedAt["p1"] = time.Now().Add(-1 * time.Second)

	table.checkDisconnectTimeouts()

	// p1 不是當前行動者，不應 Fold，但應 StandUp
	if p1.Status != StatusSittingOut {
		t.Errorf("Expected p1 status SittingOut after timeout, got %v", p1.Status)
	}
	// p2 不受影響
	if p2.Status != StatusPlaying {
		t.Errorf("Expected p2 status unchanged (Playing), got %v", p2.Status)
	}
}

// TestDisconnectTimeout_AllInPlayer AllIn 斷線玩家 → 超時後不做任何處理
func TestDisconnectTimeout_AllInPlayer(t *testing.T) {
	table := NewTable("dc-test")
	table.DisconnectTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 0, Status: StatusAllIn, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Seats[0] = p1
	table.Seats[1] = p2

	table.State = StatePreFlop
	table.CurrentPos = 1 // p2's turn
	table.MinBet = 20

	// p1 AllIn 且斷線超時
	table.DisconnectedAt["p1"] = time.Now().Add(-1 * time.Second)

	table.checkDisconnectTimeouts()

	// AllIn 玩家不應被 StandUp（StandUp 會返回 error）
	if p1.Status != StatusAllIn {
		t.Errorf("Expected AllIn player status unchanged, got %v", p1.Status)
	}
	// 斷線記錄應被清除
	if _, exists := table.DisconnectedAt["p1"]; exists {
		t.Error("Expected DisconnectedAt cleared even for AllIn player")
	}
}

// TestDisconnectTimeout_ReconnectBeforeTimeout 斷線後重連（未超時）→ 不自動 Fold
func TestDisconnectTimeout_ReconnectBeforeTimeout(t *testing.T) {
	table := NewTable("dc-test")
	table.DisconnectTimeout = 1 * time.Hour // 很長的超時

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Seats[0] = p1
	table.Seats[1] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0 // p1's turn
	table.MinBet = 20

	// 斷線
	table.processCommand(PlayerAction{
		Type:     ActionDisconnect,
		PlayerID: "p1",
	})

	// 觸發超時檢查（不應超時，因為 timeout 很長）
	table.checkDisconnectTimeouts()

	// p1 應該還在斷線狀態，未被強制動作
	if p1.Status != StatusPlaying {
		t.Errorf("Expected p1 still Playing (not timed out), got %v", p1.Status)
	}
	if _, exists := table.DisconnectedAt["p1"]; !exists {
		t.Error("Expected DisconnectedAt still present (not timed out)")
	}

	// 重連
	table.processCommand(PlayerAction{
		Type:     ActionReconnect,
		PlayerID: "p1",
	})

	// 再觸發超時檢查
	table.checkDisconnectTimeouts()

	// p1 依然正常
	if p1.Status != StatusPlaying {
		t.Errorf("Expected p1 still Playing after reconnect, got %v", p1.Status)
	}
}

// TestDisconnectTimeout_BeforeTryStartNewHand 超時 StandUp 在 tryStartNewHand 之前執行
// 驗證斷線超時玩家不被計入 readyPlayers
func TestDisconnectTimeout_BeforeTryStartNewHand(t *testing.T) {
	table := NewTable("dc-test")
	table.DisconnectTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Seats[0] = p1
	table.Seats[1] = p2

	table.State = StateIdle

	// p1 斷線超時
	table.DisconnectedAt["p1"] = time.Now().Add(-1 * time.Second)

	// 先執行 checkDisconnectTimeouts，再 tryStartNewHand
	// 模擬 Run() 中 ticker 的行為
	table.checkDisconnectTimeouts()

	// p1 應已 StandUp
	if p1.Status != StatusSittingOut {
		t.Errorf("Expected p1 SittingOut after disconnect timeout, got %v", p1.Status)
	}

	// 此時只剩 p2 一個 Playing 玩家，tryStartNewHand 不應開始
	table.tryStartNewHand()
	if table.State != StateIdle {
		t.Errorf("Expected table still Idle (only 1 ready player), got %v", table.State)
	}
}

// === removePlayer 手牌中離開測試 ===

// TestRemovePlayer_DuringIdle 閒置時移除，立即從 Players 和 Seats 清除
func TestRemovePlayer_DuringIdle(t *testing.T) {
	table := NewTable("remove-idle-test")

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusSittingOut}
	table.Players["p1"] = p1
	table.Seats[0] = p1

	table.State = StateIdle

	err := table.removePlayer("p1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, exists := table.Players["p1"]; exists {
		t.Error("Expected player removed from Players map during idle")
	}
	if table.Seats[0] != nil {
		t.Error("Expected seat 0 cleared during idle")
	}
}

// TestRemovePlayer_DuringHand_CurrentPlayer 移除當前行動者，遊戲推進到下一位
func TestRemovePlayer_DuringHand_CurrentPlayer(t *testing.T) {
	table := NewTable("remove-current-test")

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
	table.DealerPos = 0
	table.CurrentPos = 0 // p1 是當前行動者
	table.MinBet = 20

	err := table.removePlayer("p1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// p1 應該被標記為 Folded，保留在 Players map 中
	if p1.Status != StatusFolded {
		t.Errorf("Expected p1 status Folded, got %v", p1.Status)
	}
	if _, exists := table.Players["p1"]; !exists {
		t.Error("Expected p1 still in Players map during hand")
	}

	// p1 的座位應該被釋放
	if table.Seats[0] != nil {
		t.Error("Expected seat 0 cleared")
	}
	if p1.SeatIdx != -1 {
		t.Errorf("Expected p1 SeatIdx == -1, got %d", p1.SeatIdx)
	}

	// 遊戲應推進到下一位（p2 在座位 1）
	if table.CurrentPos != 1 {
		t.Errorf("Expected CurrentPos moved to 1 (p2), got %d", table.CurrentPos)
	}
}

// TestRemovePlayer_DuringHand_BetPreserved 離開玩家的下注被計入底池
func TestRemovePlayer_DuringHand_BetPreserved(t *testing.T) {
	table := NewTable("remove-bet-test")

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 980, Status: StatusPlaying, CurrentBet: 20, HoleCards: []Card{}, HasActed: true}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 980, Status: StatusPlaying, CurrentBet: 20, HoleCards: []Card{}, HasActed: true}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 980, Status: StatusPlaying, CurrentBet: 20, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	table.State = StatePreFlop
	table.DealerPos = 0
	table.CurrentPos = 2 // p3 是當前行動者
	table.MinBet = 20

	// p1 離開，但不是當前行動者
	err := table.removePlayer("p1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// p1 的 CurrentBet 應保留（不被清零）
	if p1.CurrentBet != 20 {
		t.Errorf("Expected p1 CurrentBet preserved at 20, got %d", p1.CurrentBet)
	}

	// p3 Check 完成，進入下一階段（此時 nextStreet 會收集所有玩家的 CurrentBet）
	if err := table.handleAction(PlayerAction{PlayerID: "p3", Type: ActionCheck}); err != nil {
		t.Fatalf("Expected no error for p3 check, got %v", err)
	}

	// 驗證底池包含了 p1 的下注（20 + 20 + 20 = 60）
	if table.Pots.Total() != 60 {
		t.Errorf("Expected pot total 60 (including departed player's bet), got %d", table.Pots.Total())
	}
}

// TestRemovePlayer_DuringHand_LastTwoPlayers 只剩一人時手牌正確結束
func TestRemovePlayer_DuringHand_LastTwoPlayers(t *testing.T) {
	table := NewTable("remove-last-test")

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 980, Status: StatusPlaying, CurrentBet: 20, HoleCards: []Card{}, HasActed: true}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 980, Status: StatusPlaying, CurrentBet: 20, HoleCards: []Card{}, HasActed: true}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.DealerPos = 0
	table.CurrentPos = 0 // p1 是當前行動者
	table.MinBet = 20

	err := table.removePlayer("p1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 只剩 p2，手牌應該結束
	if table.State != StateIdle {
		t.Errorf("Expected table state Idle (hand ended), got %v", table.State)
	}

	// p2 應該贏得底池（20 + 20 = 40）
	if p2.Chips != 1020 {
		t.Errorf("Expected p2 chips 1020 (980 + 40 pot), got %d", p2.Chips)
	}

	// p1 應該在 endHand() 清理中被移除
	if _, exists := table.Players["p1"]; exists {
		t.Error("Expected p1 removed from Players map after hand ended")
	}
}

// === handleAction 錯誤回傳測試 ===

// TestHandleAction_NotYourTurn 非當前玩家送動作 → ErrNotYourTurn
func TestHandleAction_NotYourTurn(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0 // p1's turn
	table.MinBet = 20

	// p2 嘗試行動（不是 p2 的回合）
	err := table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCall})
	if err != ErrNotYourTurn {
		t.Errorf("Expected ErrNotYourTurn, got %v", err)
	}
}

// TestHandleAction_CannotCheck 有下注時 Check → ErrCannotCheck
func TestHandleAction_CannotCheck(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20
	// p1 的 CurrentBet 為 0，低於 MinBet 20，所以不能 Check
	p1.CurrentBet = 0

	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCheck})
	if err != ErrCannotCheck {
		t.Errorf("Expected ErrCannotCheck, got %v", err)
	}
}

// TestHandleAction_BetTooLow 下注低於最低 → ErrBetTooLow
func TestHandleAction_BetTooLow(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// 嘗試 Bet 10（低於 MinBet 20）
	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionBet, Amount: 10})
	if err != ErrBetTooLow {
		t.Errorf("Expected ErrBetTooLow, got %v", err)
	}
}

// TestHandleAction_InsufficientChips 籌碼不足 → ErrInsufficientChips
func TestHandleAction_InsufficientChips(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 30, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// 嘗試 Raise 到 50，但 p1 只有 30 chips（diff = 50 - 0 = 50 > 30）
	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionRaise, Amount: 50})
	if err != ErrInsufficientChips {
		t.Errorf("Expected ErrInsufficientChips, got %v", err)
	}
}

// TestHandleAction_AlreadyAllIn 已全押再 AllIn → ErrAlreadyAllIn
func TestHandleAction_AlreadyAllIn(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 0, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// p1 籌碼為 0，嘗試 AllIn
	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionAllIn})
	if err != ErrAlreadyAllIn {
		t.Errorf("Expected ErrAlreadyAllIn, got %v", err)
	}
}

// TestHandleAction_ValidAction_ReturnsNil 合法動作 → nil
func TestHandleAction_ValidAction_ReturnsNil(t *testing.T) {
	table := NewTable("err-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// p1 Call（合法動作）
	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall})
	if err != nil {
		t.Errorf("Expected nil error for valid action, got %v", err)
	}

	// 驗證動作已生效
	if p1.CurrentBet != 20 {
		t.Errorf("Expected p1 CurrentBet 20, got %d", p1.CurrentBet)
	}
}

// === 行動計時器測試 ===

// TestActionTimeout_AutoFold 超時且有下注 → 自動 Fold
func TestActionTimeout_AutoFold(t *testing.T) {
	table := NewTable("timeout-test")
	table.ActionTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0 // p1's turn
	table.MinBet = 20
	p1.CurrentBet = 0 // p1 hasn't matched the MinBet → cannot Check → auto Fold

	// 設定已過期的 deadline
	table.ActionDeadline = time.Now().Add(-1 * time.Second)

	table.checkActionTimeout()

	// p1 應該被自動 Fold
	if p1.Status != StatusFolded {
		t.Errorf("Expected p1 status Folded after action timeout, got %v", p1.Status)
	}
	if !p1.HasActed {
		t.Error("Expected p1 HasActed true after auto-fold")
	}
}

// TestActionTimeout_AutoCheck 超時且可 Check → 自動 Check
func TestActionTimeout_AutoCheck(t *testing.T) {
	table := NewTable("timeout-test")
	table.ActionTimeout = 1 * time.Millisecond

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StateFlop
	table.CurrentPos = 0 // p1's turn
	table.MinBet = 0     // Flop 階段 MinBet 重置為 0
	p1.CurrentBet = 0    // p1.CurrentBet >= MinBet (0 >= 0) → can Check

	// 設定已過期的 deadline
	table.ActionDeadline = time.Now().Add(-1 * time.Second)

	table.checkActionTimeout()

	// p1 應該被自動 Check（不是 Fold）
	if p1.Status == StatusFolded {
		t.Error("Expected p1 NOT folded (should auto-check when MinBet == 0)")
	}
	if !p1.HasActed {
		t.Error("Expected p1 HasActed true after auto-check")
	}
}

// TestActionTimeout_NoTimerWhenIdle Idle 狀態不觸發超時
func TestActionTimeout_NoTimerWhenIdle(t *testing.T) {
	table := NewTable("timeout-test")

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	table.Seats[0] = p1
	table.Players["p1"] = p1

	table.State = StateIdle
	table.CurrentPos = 0
	// 設定已過期的 deadline（但 State 為 Idle）
	table.ActionDeadline = time.Now().Add(-1 * time.Second)

	table.checkActionTimeout()

	// Idle 狀態下不應觸發任何動作
	if p1.Status != StatusPlaying {
		t.Errorf("Expected p1 status unchanged in Idle state, got %v", p1.Status)
	}
}

// TestActionTimeout_ResetOnAction 合法動作後 deadline 被重設（moveToNextPlayer 設定新的）
func TestActionTimeout_ResetOnAction(t *testing.T) {
	table := NewTable("timeout-test")
	table.ActionTimeout = 30 * time.Second

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0
	table.MinBet = 20

	// 設定一個即將過期的 deadline
	oldDeadline := time.Now().Add(1 * time.Second)
	table.ActionDeadline = oldDeadline

	// p1 做合法動作（Call）→ moveToNextPlayer 會設定新的 deadline
	err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// deadline 應該被更新為新的值（大約 30 秒後）
	if table.ActionDeadline.Equal(oldDeadline) {
		t.Error("Expected ActionDeadline to be reset after action")
	}
	if table.ActionDeadline.Before(time.Now()) {
		t.Error("Expected new ActionDeadline to be in the future")
	}
}

// TestActionTimeout_DeadlineInYourTurnEvent YOUR_TURN 事件包含 deadline
func TestActionTimeout_DeadlineInYourTurnEvent(t *testing.T) {
	table := NewTable("timeout-test")
	table.ActionTimeout = 30 * time.Second

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.CurrentPos = 0 // 將從 0 開始找下一位
	table.MinBet = 20

	var capturedEvent TableEvent
	table.AddOnEvent(func(event TableEvent) {
		if event.Type == EventYourTurn {
			capturedEvent = event
		}
	})

	// moveToNextPlayer 會觸發 YOUR_TURN 事件
	table.moveToNextPlayer()

	if capturedEvent.Type != EventYourTurn {
		t.Fatal("Expected YOUR_TURN event to be fired")
	}

	deadline, ok := capturedEvent.Data["deadline"]
	if !ok {
		t.Fatal("Expected 'deadline' field in YOUR_TURN event data")
	}

	deadlineUnix, ok := deadline.(int64)
	if !ok {
		t.Fatalf("Expected deadline to be int64, got %T", deadline)
	}

	// deadline 應該大約在 30 秒後
	now := time.Now().Unix()
	if deadlineUnix < now || deadlineUnix > now+35 {
		t.Errorf("Expected deadline around 30s from now, got %d (now=%d, diff=%d)",
			deadlineUnix, now, deadlineUnix-now)
	}
}

// TestActionTimeout_ClearedOnEndHand endHand 後 ActionDeadline 被清除
func TestActionTimeout_ClearedOnEndHand(t *testing.T) {
	table := NewTable("timeout-test")

	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	table.State = StatePreFlop
	table.DealerPos = 0
	// 設定一個未來的 deadline
	table.ActionDeadline = time.Now().Add(30 * time.Second)

	// 呼叫 endHand
	table.endHand()

	// ActionDeadline 應該被清除
	if !table.ActionDeadline.IsZero() {
		t.Error("Expected ActionDeadline to be cleared after endHand")
	}
}
