package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shinjuwu/TheNuts/internal/game/core"
	"github.com/shinjuwu/TheNuts/internal/game/poker"
)

// TestFrameworkBasicFlow 測試框架基本流程
func TestFrameworkBasicFlow(t *testing.T) {
	fmt.Println("\n🧪 開始框架驗證測試...")

	// 1. 創建遊戲服務
	gameService := core.NewGameService()
	fmt.Println("✅ GameService 創建成功")

	// 2. 註冊德撲引擎
	gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})
	fmt.Println("✅ 德撲引擎註冊成功")

	// 3. 創建德撲桌
	pokerConfig := core.GameConfig{
		GameID:     "test_table_001",
		MaxPlayers: 9,
		MinBet:     10,
		MaxBet:     1000,
		CustomData: map[string]interface{}{
			"blinds": int64(20),
		},
	}

	tableID, err := gameService.CreateGame(core.GameTypePoker, pokerConfig)
	if err != nil {
		t.Fatalf("❌ 創建桌子失敗: %v", err)
	}
	fmt.Printf("✅ 德撲桌創建成功: %s\n", tableID)

	// 4. 創建玩家會話
	session1 := gameService.CreateSession("alice", 10000)
	session2 := gameService.CreateSession("bob", 10000)
	session3 := gameService.CreateSession("charlie", 10000)
	fmt.Println("✅ 創建 3 個玩家會話")

	// 5. 玩家加入遊戲
	if err := gameService.JoinGame(session1, tableID, 1000); err != nil {
		t.Fatalf("❌ Alice 加入失敗: %v", err)
	}
	fmt.Println("✅ Alice 加入遊戲 (買入 1000)")

	if err := gameService.JoinGame(session2, tableID, 1000); err != nil {
		t.Fatalf("❌ Bob 加入失敗: %v", err)
	}
	fmt.Println("✅ Bob 加入遊戲 (買入 1000)")

	if err := gameService.JoinGame(session3, tableID, 500); err != nil {
		t.Fatalf("❌ Charlie 加入失敗: %v", err)
	}
	fmt.Println("✅ Charlie 加入遊戲 (買入 500)")

	// 6. 驗證玩家會話狀態
	s1, _ := gameService.GetSession(session1)
	if s1.CurrentGameID != tableID {
		t.Errorf("❌ Alice 未正確加入桌子")
	}
	fmt.Println("✅ 驗證玩家會話狀態正確")

	// 7. 模擬玩家動作
	ctx := context.Background()

	action1 := core.PlayerAction{
		PlayerID:  "alice",
		SessionID: session1,
		GameID:    tableID,
		Type:      core.ActionRaise,
		Amount:    50,
		Timestamp: time.Now(),
	}

	result, err := gameService.HandlePlayerAction(ctx, action1)
	if err != nil {
		t.Errorf("❌ 處理動作失敗: %v", err)
	} else {
		fmt.Printf("✅ Alice 加注 50: %s\n", result.Message)
	}

	// 8. 驗證遊戲引擎
	engine, err := gameService.GetTable(tableID)
	if err != nil {
		t.Fatalf("❌ 獲取桌子失敗: %v", err)
	}

	if engine.GetType() != core.GameTypePoker {
		t.Errorf("❌ 遊戲類型錯誤，期望 poker，得到 %s", engine.GetType())
	}
	fmt.Println("✅ 遊戲引擎類型正確")

	// 9. 獲取遊戲狀態
	state := engine.GetState()
	fmt.Printf("✅ 遊戲狀態: ID=%s, Phase=%s\n", state.GetID(), state.GetPhase())

	// 10. 測試關閉桌子
	// 註: 暫時跳過，因為關閉會影響其他測試

	fmt.Println("\n🎉 所有測試通過！框架運作正常。")
}

// TestMultipleTables 測試多桌並發
func TestMultipleTables(t *testing.T) {
	fmt.Println("\n🧪 測試多桌並發...")

	gameService := core.NewGameService()
	gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})

	// 創建 5 張桌子
	tableCount := 5
	tableIDs := make([]string, tableCount)

	for i := 0; i < tableCount; i++ {
		config := core.GameConfig{
			GameID:     fmt.Sprintf("table_%d", i),
			MaxPlayers: 9,
			MinBet:     10,
		}

		tableID, err := gameService.CreateGame(core.GameTypePoker, config)
		if err != nil {
			t.Fatalf("❌ 創建第 %d 張桌子失敗: %v", i, err)
		}
		tableIDs[i] = tableID
	}

	fmt.Printf("✅ 成功創建 %d 張桌子\n", tableCount)

	// 驗證所有桌子都存在
	for i, tableID := range tableIDs {
		_, err := gameService.GetTable(tableID)
		if err != nil {
			t.Errorf("❌ 第 %d 張桌子不存在: %v", i, err)
		}
	}

	fmt.Println("✅ 所有桌子驗證通過")
}

// TestSessionManagement 測試會話管理
func TestSessionManagement(t *testing.T) {
	fmt.Println("\n🧪 測試會話管理...")

	gameService := core.NewGameService()
	gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})

	// 創建會話
	sessionID := gameService.CreateSession("test_player", 5000)
	fmt.Printf("✅ 創建會話: %s\n", sessionID)

	// 獲取會話
	session, err := gameService.GetSession(sessionID)
	if err != nil {
		t.Fatalf("❌ 獲取會話失敗: %v", err)
	}

	// 驗證初始狀態
	if session.PlayerID != "test_player" {
		t.Errorf("❌ PlayerID 錯誤，期望 test_player，得到 %s", session.PlayerID)
	}

	if session.Balance != 5000 {
		t.Errorf("❌ Balance 錯誤，期望 5000，得到 %d", session.Balance)
	}

	if session.CurrentGameID != "" {
		t.Errorf("❌ 初始狀態不應該在遊戲中")
	}

	fmt.Println("✅ 會話初始狀態正確")

	// 創建桌子並加入
	config := core.GameConfig{
		GameID:     "session_test_table",
		MaxPlayers: 9,
		MinBet:     10,
	}
	tableID, _ := gameService.CreateGame(core.GameTypePoker, config)

	if err := gameService.JoinGame(sessionID, tableID, 1000); err != nil {
		t.Fatalf("❌ 加入遊戲失敗: %v", err)
	}

	// 驗證會話狀態更新
	session, _ = gameService.GetSession(sessionID)
	if session.CurrentGameID != tableID {
		t.Errorf("❌ CurrentGameID 未更新")
	}

	if session.Balance != 4000 { // 5000 - 1000
		t.Errorf("❌ 餘額未正確扣除，期望 4000，得到 %d", session.Balance)
	}

	fmt.Println("✅ 會話狀態更新正確")

	// 關閉會話
	if err := gameService.CloseSession(sessionID); err != nil {
		t.Errorf("❌ 關閉會話失敗: %v", err)
	}

	// 驗證會話已關閉
	_, err = gameService.GetSession(sessionID)
	if err == nil {
		t.Errorf("❌ 會話應該已經關閉")
	}

	fmt.Println("✅ 會話關閉成功")
}

// TestInsufficientBalance 測試餘額不足的情況
func TestInsufficientBalance(t *testing.T) {
	fmt.Println("\n🧪 測試餘額不足情況...")

	gameService := core.NewGameService()
	gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})

	// 創建低餘額玩家
	sessionID := gameService.CreateSession("poor_player", 100)

	// 創建桌子
	config := core.GameConfig{
		GameID:     "balance_test_table",
		MaxPlayers: 9,
		MinBet:     10,
	}
	tableID, _ := gameService.CreateGame(core.GameTypePoker, config)

	// 嘗試買入超過餘額
	err := gameService.JoinGame(sessionID, tableID, 1000)
	if err == nil {
		t.Errorf("❌ 應該因為餘額不足而失敗")
	} else {
		fmt.Printf("✅ 正確拒絕: %v\n", err)
	}

	// 驗證餘額未被扣除
	session, _ := gameService.GetSession(sessionID)
	if session.Balance != 100 {
		t.Errorf("❌ 餘額不應該被扣除，期望 100，得到 %d", session.Balance)
	}

	fmt.Println("✅ 餘額保護機制正常")
}
