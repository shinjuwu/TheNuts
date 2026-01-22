# TheNuts 多遊戲博弈框架 - 架構設計文檔

## 🎯 設計目標

構建一個**可擴展的商業化博弈遊戲框架**，支援：
- 棋牌類遊戲（德州撲克、百家樂、21點等）
- 電子遊戲（老虎機、輪盤等）
- 視訊遊戲（真人荷官）

## 🏗️ 核心架構

### 1. 分層架構

```
┌─────────────────────────────────────────────────┐
│  Presentation Layer (表現層)                     │
│  - WebSocket Gateway                            │
│  - HTTP API                                     │
│  - Admin Dashboard                              │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  Application Layer (應用層)                      │
│  - GameService (遊戲服務統一入口)                │
│  - PlayerSession (玩家會話管理)                  │
│  - TableManager (桌子管理器)                     │
└─────────────────┬───────────────────────────────┘
                  │
        ┌─────────┼─────────┐
        │         │         │
┌───────▼──┐ ┌────▼────┐ ┌─▼────────┐
│ Poker    │ │ Slot    │ │ Baccarat │
│ Engine   │ │ Engine  │ │ Engine   │
└──────────┘ └─────────┘ └──────────┘
        │         │         │
        └─────────┼─────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  Domain Layer (領域層)                           │
│  - GameEngine Interface                         │
│  - Common Game Logic                            │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  Infrastructure Layer (基礎設施層)               │
│  - Database (玩家資料、遊戲記錄)                 │
│  - RNG Service (隨機數生成器)                    │
│  - Wallet Service (錢包服務)                     │
│  - Audit Log (審計日誌)                          │
└─────────────────────────────────────────────────┘
```

### 2. 核心介面設計

#### GameEngine 介面
所有遊戲引擎必須實現的契約：

```go
type GameEngine interface {
    GetType() GameType
    Initialize(config GameConfig) error
    Start(ctx context.Context) error
    Stop() error
    HandleAction(ctx context.Context, action PlayerAction) (*ActionResult, error)
    GetState() GameState
    AddPlayer(player *Player) error
    RemovePlayer(playerID string) error
    BroadcastEvent(event GameEvent)
}
```

**設計思想**：
- 通過介面隔離，每種遊戲可以獨立開發和測試
- 統一的動作處理流程，方便監控和審計
- 支援水平擴展（不同遊戲可部署在不同服務器）

#### GameConfig 配置
```go
type GameConfig struct {
    GameID      string
    MaxPlayers  int
    MinBet      int64
    MaxBet      int64
    RakePercent float64
    Timeout     time.Duration
    CustomData  map[string]interface{} // 遊戲專屬配置
}
```

**擴展性**：
- `CustomData` 允許每種遊戲添加專屬配置
- 例如：德撲的盲注結構、老虎機的RTP、百家樂的佣金比例

### 3. 遊戲註冊機制（工廠模式）

```go
// 啟動時註冊遊戲引擎
gameService := core.NewGameService()
gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})
gameService.RegisterGameEngine(core.GameTypeSlot, &slot.SlotEngineFactory{})
gameService.RegisterGameEngine(core.GameTypeBaccarat, &baccarat.BaccaratEngineFactory{})
```

**優點**：
- 插件式架構，新增遊戲無需修改核心代碼
- 支援動態載入（未來可實現熱更新）
- 方便 A/B 測試（同一遊戲類型的不同版本）

### 4. 玩家會話管理

```go
type PlayerSession struct {
    SessionID     string
    PlayerID      string
    CurrentGameID string  // 當前所在遊戲桌
    GameType      GameType
    Balance       int64
    SendCh        chan []byte // WebSocket 發送通道
}
```

**關鍵特性**：
- 跨遊戲共用的會話層（玩家可以在不同遊戲間切換）
- 統一的餘額管理（避免重複扣款）
- 斷線重連支援（通過 SessionID 恢復）

### 5. 事件驅動架構

```go
type GameEvent struct {
    EventType      EventType
    GameID         string
    Timestamp      time.Time
    Data           interface{}
    TargetPlayerID string  // 視角過濾
}
```

**事件類型**：
- `EventGameStart` - 遊戲開始
- `EventPlayerJoin` - 玩家加入
- `EventBetPlaced` - 下注
- `EventCardDealt` - 發牌
- `EventWinner` - 結算

**視角過濾**：
- `TargetPlayerID` 為空 = 廣播給所有人
- `TargetPlayerID` 有值 = 只發給特定玩家（例如私人手牌）

## 🔌 如何添加新遊戲

### 步驟 1: 實現 GameEngine 介面

```go
package slot

type SlotEngine struct {
    config core.GameConfig
    reels  [][]string  // 老虎機滾輪
    // ... 其他遊戲專屬邏輯
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
            Data: result,
        }, nil
    }
    return nil, fmt.Errorf("invalid action")
}

// ... 實現其他介面方法
```

### 步驟 2: 創建工廠

```go
type SlotEngineFactory struct{}

func (f *SlotEngineFactory) Create(config core.GameConfig) (core.GameEngine, error) {
    return &SlotEngine{
        config: config,
        reels:  loadReelsFromConfig(config),
    }, nil
}
```

### 步驟 3: 註冊引擎

```go
gameService.RegisterGameEngine(core.GameTypeSlot, &slot.SlotEngineFactory{})
```

### 步驟 4: 創建遊戲實例

```go
slotConfig := core.GameConfig{
    GameID: "slot_001",
    CustomData: map[string]interface{}{
        "rtp": 0.96,
        "paylines": 20,
    },
}
gameService.CreateGame(core.GameTypeSlot, slotConfig)
```

## 🔐 安全考量

### 1. 視角隔離
- 廣播事件時，根據 `TargetPlayerID` 過濾數據
- 德撲中其他玩家的手牌不得洩露

### 2. 動作驗證
```go
// 每個引擎必須驗證動作合法性
func (e *PokerEngine) HandleAction(action core.PlayerAction) {
    // 1. 驗證是否輪到該玩家
    // 2. 驗證動作是否合法（不能在 PreFlop 棄牌時下注）
    // 3. 驗證金額是否足夠
}
```

### 3. 隨機數安全
- 使用 `crypto/rand` 而非 `math/rand`
- 洗牌算法採用 Fisher-Yates

### 4. 審計日誌
- 所有玩家動作記錄到 Audit Log
- 包含時間戳、玩家ID、動作類型、金額

## 📊 監控與可觀測性

### 關鍵指標 (待實現)
- 每秒動作數 (Actions per Second)
- 遊戲平均耗時
- 玩家勝率分佈
- 異常動作檢測（作弊檢測）

### 日誌結構
```go
{
    "timestamp": "2026-01-22T18:00:00Z",
    "game_id": "poker_001",
    "player_id": "alice",
    "action": "raise",
    "amount": 100,
    "result": "success"
}
```

## 🚀 未來擴展

### 1. 多桌錦標賽 (MTT)
- 需要 TournamentManager
- 實現併桌邏輯
- 盲注結構管理

### 2. 分散式部署
- 使用 Redis Pub/Sub 處理跨服務器廣播
- 使用 etcd 進行服務發現

### 3. 遊戲回放
- 記錄完整的 GameEvent 流
- 支援回放和爭議處理

### 4. AI 對手
- 實現 BotPlayer
- 支援不同難度等級

## 📝 檔案結構

```
internal/game/
├── core/                    # 核心抽象層
│   ├── game_engine.go       # GameEngine 介面定義
│   ├── table_manager.go     # 桌子管理器
│   └── service.go           # 遊戲服務統一入口
│
├── poker/                   # 德州撲克引擎
│   ├── poker_engine.go      # 實現 GameEngine
│   └── ...
│
├── slot/                    # 老虎機引擎 (待實現)
│   └── slot_engine.go
│
├── baccarat/                # 百家樂引擎 (待實現)
│   └── baccarat_engine.go
│
├── domain/                  # 領域層 (現有德撲邏輯)
│   ├── table.go
│   ├── player.go
│   └── ...
│
└── adapter/                 # 適配器層
    └── ws/                  # WebSocket 適配器
```

## 🎓 設計模式應用

1. **工廠模式** - GameEngineFactory
2. **策略模式** - 不同遊戲引擎實現同一介面
3. **觀察者模式** - GameEvent 廣播機制
4. **適配器模式** - 將現有 domain.Table 適配到 GameEngine
5. **單例模式** - GameService 全局唯一

## ✅ 下一步建議

1. **完善德撲引擎**
   - 實現盲注邏輯（已在 CODE_REVIEW.md 指出）
   - 實現錦標賽管理器
   
2. **實現第二個遊戲引擎**
   - 建議從簡單的老虎機開始
   - 驗證框架的通用性
   
3. **添加持久化層**
   - 玩家餘額持久化
   - 遊戲歷史記錄
   
4. **添加認證授權**
   - JWT Token 驗證
   - 防作弊機制

5. **性能優化**
   - 壓力測試（模擬 1000+ 並發桌）
   - 內存優化
   - 使用 Protobuf 替代 JSON
