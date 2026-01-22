package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shinjuwu/TheNuts/internal/game/core"
	"github.com/shinjuwu/TheNuts/internal/game/poker"
)

// 示例: 如何使用多遊戲框架

func main() {
	// 1. 創建遊戲服務
	gameService := core.NewGameService()

	// 2. 註冊遊戲引擎 (德撲)
	gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})

	// 未來可以註冊其他遊戲:
	// gameService.RegisterGameEngine(core.GameTypeSlot, &slot.SlotEngineFactory{})
	// gameService.RegisterGameEngine(core.GameTypeBaccarat, &baccarat.BaccaratEngineFactory{})

	// 3. 創建一張德撲桌
	pokerConfig := core.GameConfig{
		GameID:     "poker_table_001",
		MaxPlayers: 9,
		MinBet:     10,
		MaxBet:     1000,
		CustomData: map[string]interface{}{
			"blinds": int64(20), // 大盲注
		},
	}

	tableID, err := gameService.CreateGame(core.GameTypePoker, pokerConfig)
	if err != nil {
		log.Fatalf("Failed to create poker table: %v", err)
	}
	fmt.Printf("✅ Poker table created: %s\n", tableID)

	// 4. 模擬玩家加入
	// 創建玩家會話
	sessionID1 := gameService.CreateSession("player_alice", 10000)
	sessionID2 := gameService.CreateSession("player_bob", 10000)

	// 玩家加入遊戲 (買入 1000)
	if err := gameService.JoinGame(sessionID1, tableID, 1000); err != nil {
		log.Fatalf("Alice failed to join: %v", err)
	}
	fmt.Println("✅ Alice joined the game")

	if err := gameService.JoinGame(sessionID2, tableID, 1000); err != nil {
		log.Fatalf("Bob failed to join: %v", err)
	}
	fmt.Println("✅ Bob joined the game")

	// 5. 模擬玩家動作
	ctx := context.Background()

	action1 := core.PlayerAction{
		PlayerID:  "player_alice",
		SessionID: sessionID1,
		GameID:    tableID,
		Type:      core.ActionRaise,
		Amount:    50,
	}

	result, err := gameService.HandlePlayerAction(ctx, action1)
	if err != nil {
		log.Printf("Action failed: %v", err)
	} else {
		fmt.Printf("✅ Action result: %+v\n", result)
	}

	// 6. 未來可以創建其他類型的遊戲
	// slotConfig := core.GameConfig{
	//     GameID:     "slot_machine_001",
	//     MaxPlayers: 1,
	//     CustomData: map[string]interface{}{
	//         "rtp": 0.96, // Return to Player
	//     },
	// }
	// slotTableID, _ := gameService.CreateGame(core.GameTypeSlot, slotConfig)

	fmt.Println("\n🎰 Multi-game framework is running!")
	fmt.Println("   - Poker engine: ✅")
	fmt.Println("   - Slot engine: ⏳ (待實現)")
	fmt.Println("   - Baccarat engine: ⏳ (待實現)")

	// 保持運行
	select {}
}
