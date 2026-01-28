package domain

import (
	"testing"
	"time"
)

// TestAutoGameFlow 測試自動遊戲流程
func TestAutoGameFlow(t *testing.T) {
	// 1. 創建牌桌
	table := NewTable("auto-game")

	// 2. 添加 3 個玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}
	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3
	table.DealerPos = 0

	// 3. 在後台啟動 Table.Run()
	go table.Run()

	// 4. 等待自動開局（應該在 1 秒內觸發）
	time.Sleep(1500 * time.Millisecond)

	// 5. 驗證遊戲已經開始
	if table.State == StateIdle {
		t.Error("Expected game to auto-start, but still in StateIdle")
	}

	// 6. 驗證玩家已經收到手牌
	if len(p1.HoleCards) != 2 {
		t.Errorf("Expected p1 to have 2 hole cards, got %d", len(p1.HoleCards))
	}

	// 7. 驗證盲注已經收取
	totalBets := p1.CurrentBet + p2.CurrentBet + p3.CurrentBet
	if totalBets == 0 {
		t.Error("Expected blinds to be posted, but total bets is 0")
	}

	// 8. 清理
	close(table.CloseCh)
	time.Sleep(100 * time.Millisecond)

	t.Logf("Auto-game flow test passed. Game state: %v", table.State)
}

// TestDealerRotation 測試莊家位置推進
func TestDealerRotation(t *testing.T) {
	// 1. 創建牌桌
	table := NewTable("dealer-rotation")

	// 2. 添加 3 個玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying}
	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3
	table.DealerPos = 0

	// 3. 記錄初始位置
	initialDealer := table.DealerPos

	// 4. 手動開始第一手牌
	table.StartHand()

	// 5. 直接結束手牌（模擬所有人 Fold）
	table.endHand()

	// 6. 驗證 Dealer Button 已移動
	if table.DealerPos == initialDealer {
		t.Error("Expected dealer button to rotate after hand")
	}
	expectedDealer := 1 // 應該移到下一個座位
	if table.DealerPos != expectedDealer {
		t.Errorf("Expected dealer at seat %d, got %d", expectedDealer, table.DealerPos)
	}

	// 7. 驗證玩家狀態已重置
	if p1.Status != StatusPlaying {
		t.Errorf("Expected p1 status to reset to Playing, got %v", p1.Status)
	}
	if p1.CurrentBet != 0 {
		t.Errorf("Expected p1 CurrentBet to reset to 0, got %d", p1.CurrentBet)
	}
	if len(p1.HoleCards) != 0 {
		t.Errorf("Expected p1 HoleCards to be cleared, got %d cards", len(p1.HoleCards))
	}

	// 8. 驗證狀態回到 Idle
	if table.State != StateIdle {
		t.Errorf("Expected StateIdle, got %v", table.State)
	}

	t.Logf("Dealer rotation test passed. New dealer position: %d", table.DealerPos)
}

// TestPlayerResetAfterHand 測試玩家狀態重置
func TestPlayerResetAfterHand(t *testing.T) {
	table := NewTable("reset-test")

	// 添加玩家
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 0, Status: StatusAllIn} // 沒籌碼的 All-in 玩家
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 500, Status: StatusFolded}
	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	// 給玩家一些狀態
	p1.HoleCards = []Card{NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade)}
	p1.CurrentBet = 100
	p1.HasActed = true

	// 執行重置
	table.resetPlayersForNextHand()

	// 驗證 p1 狀態重置
	if len(p1.HoleCards) != 0 {
		t.Error("Expected p1 HoleCards to be cleared")
	}
	if p1.CurrentBet != 0 {
		t.Error("Expected p1 CurrentBet to be 0")
	}
	if p1.HasActed {
		t.Error("Expected p1 HasActed to be false")
	}
	if p1.Status != StatusPlaying {
		t.Errorf("Expected p1 to remain Playing, got %v", p1.Status)
	}

	// 驗證 p2（沒籌碼）被設為 SittingOut
	if p2.Status != StatusSittingOut {
		t.Errorf("Expected p2 with 0 chips to be SittingOut, got %v", p2.Status)
	}

	// 驗證 p3（Folded → Playing）
	if p3.Status != StatusPlaying {
		t.Errorf("Expected p3 to reset to Playing, got %v", p3.Status)
	}

	t.Log("Player reset test passed")
}
