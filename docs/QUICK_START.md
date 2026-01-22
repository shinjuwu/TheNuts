# 快速開始 - 多遊戲框架

## 🚀 5 分鐘快速體驗

### 1. 運行示例程序

```bash
# 運行多遊戲框架示例
go run examples/multi_game_example.go
```

你會看到：
```
✅ Poker table created: poker_table_001
✅ Alice joined the game
✅ Bob joined the game
✅ Action result: {Success:true Message:action queued}

🎰 Multi-game framework is running!
   - Poker engine: ✅
   - Slot engine: ⏳ (待實現)
   - Baccarat engine: ⏳ (待實現)
```

### 2. 創建你的第一個遊戲桌

```go
package main

import (
    "github.com/shinjuwu/TheNuts/internal/game/core"
    "github.com/shinjuwu/TheNuts/internal/game/poker"
)

func main() {
    // 1. 初始化服務
    gameService := core.NewGameService()
    
    // 2. 註冊德撲引擎
    gameService.RegisterGameEngine(
        core.GameTypePoker,
        &poker.PokerEngineFactory{},
    )
    
    // 3. 創建德撲桌
    config := core.GameConfig{
        GameID:     "my_first_table",
        MaxPlayers: 9,
        MinBet:     10,
        MaxBet:     1000,
        CustomData: map[string]interface{}{
            "blinds": int64(20),
        },
    }
    
    tableID, _ := gameService.CreateGame(core.GameTypePoker, config)
    println("Table created:", tableID)
}
```

### 3. 讓玩家加入遊戲

```go
// 創建玩家會話
sessionID := gameService.CreateSession("player123", 10000)

// 加入遊戲 (買入 1000 籌碼)
err := gameService.JoinGame(sessionID, tableID, 1000)
if err != nil {
    log.Fatal(err)
}
```

### 4. 處理玩家動作

```go
ctx := context.Background()

action := core.PlayerAction{
    PlayerID:  "player123",
    SessionID: sessionID,
    GameID:    tableID,
    Type:      core.ActionRaise,
    Amount:    50,
}

result, err := gameService.HandlePlayerAction(ctx, action)
fmt.Printf("Result: %+v\n", result)
```

## 📚 進階用法

### 監聽遊戲事件

```go
// 獲取遊戲引擎
engine, _ := gameService.GetTable(tableID)

// 訂閱事件 (如果引擎支援)
if pokerEngine, ok := engine.(*poker.PokerEngine); ok {
    go func() {
        for event := range pokerEngine.EventCh() {
            switch event.EventType {
            case core.EventPlayerJoin:
                fmt.Println("Player joined!")
            case core.EventBetPlaced:
                fmt.Println("Bet placed!")
            }
        }
    }()
}
```

### 自定義遊戲配置

```go
// 德撲錦標賽配置
tournamentConfig := core.GameConfig{
    GameID:     "tournament_001",
    MaxPlayers: 9,
    MinBet:     10,
    CustomData: map[string]interface{}{
        "blinds":          int64(20),
        "blind_structure": []int64{10, 20, 30, 50, 100}, // 盲注遞增
        "level_duration":  15 * 60, // 每級 15 分鐘
        "tournament_type": "MTT",
    },
}
```

### 獲取遊戲狀態（斷線重連）

```go
// 玩家斷線後重連
session, _ := gameService.GetSession(sessionID)
if session.CurrentGameID != "" {
    // 獲取當前遊戲狀態
    engine, _ := gameService.GetTable(session.CurrentGameID)
    state := engine.GetState()
    
    // 發送狀態快照給前端
    snapshot := map[string]interface{}{
        "game_id": state.GetID(),
        "phase":   state.GetPhase(),
        "players": state.GetPlayers(),
    }
}
```

## 🧪 運行測試

```bash
# 運行所有測試
go test ./...

# 只測試核心框架
go test ./internal/game/core/...

# 只測試德撲引擎
go test ./internal/game/poker/...

# 運行現有的 domain 測試（驗證兼容性）
go test ./internal/game/domain/...
```

## 🔧 開發新遊戲引擎

### 步驟 1: 定義遊戲類型

```go
// internal/game/core/game_engine.go
const (
    GameTypeSlot GameType = "slot"  // 新增
)
```

### 步驟 2: 實現引擎

```go
// internal/game/slot/slot_engine.go
package slot

import "github.com/shinjuwu/TheNuts/internal/game/core"

type SlotEngine struct {
    config core.GameConfig
    // ... 遊戲狀態
}

func (e *SlotEngine) GetType() core.GameType {
    return core.GameTypeSlot
}

func (e *SlotEngine) HandleAction(ctx context.Context, action core.PlayerAction) (*core.ActionResult, error) {
    if action.Type == core.ActionSpin {
        // 處理旋轉邏輯
        result := e.spin()
        return &core.ActionResult{
            Success: true,
            Data:    result,
        }, nil
    }
    return nil, fmt.Errorf("invalid action")
}

// 實現其他必需方法...
```

### 步驟 3: 創建工廠

```go
type SlotEngineFactory struct{}

func (f *SlotEngineFactory) Create(config core.GameConfig) (core.GameEngine, error) {
    return &SlotEngine{config: config}, nil
}
```

### 步驟 4: 註冊並使用

```go
gameService.RegisterGameEngine(core.GameTypeSlot, &slot.SlotEngineFactory{})

slotConfig := core.GameConfig{
    GameID: "slot_001",
    CustomData: map[string]interface{}{
        "rtp":      0.96,  // Return to Player
        "paylines": 20,
    },
}

tableID, _ := gameService.CreateGame(core.GameTypeSlot, slotConfig)
```

## 🐛 常見錯誤

### 錯誤 1: "unsupported game type"

```
原因: 忘記註冊遊戲引擎
解決: gameService.RegisterGameEngine(gameType, factory)
```

### 錯誤 2: "session not found"

```
原因: 玩家會話過期或未創建
解決: sessionID := gameService.CreateSession(playerID, balance)
```

### 錯誤 3: "action queue full"

```
原因: 遊戲引擎處理速度跟不上
解決: 增加 ActionCh 緩衝區大小或優化遊戲邏輯
```

## 📖 相關文檔

- [架構設計](./ARCHITECTURE.md) - 深入理解框架設計
- [遷移指南](./MIGRATION_GUIDE.md) - 從舊代碼遷移
- [API 參考](./API_REFERENCE.md) - 完整 API 文檔（待建立）

## 💡 最佳實踐

1. **永遠使用 context.Context** - 支援超時和取消
2. **視角過濾** - 廣播事件時檢查 `TargetPlayerID`
3. **錯誤處理** - 返回有意義的錯誤訊息
4. **日誌記錄** - 關鍵動作記錄到審計日誌
5. **測試先行** - 新功能先寫測試

## 🎯 下一步

1. ✅ 完成快速開始教程
2. [ ] 閱讀 [架構設計文檔](./ARCHITECTURE.md)
3. [ ] 實現你的第一個遊戲引擎
4. [ ] 添加 WebSocket 整合
5. [ ] 部署到生產環境

祝你開發順利！🚀
