# 下一步行動計劃 (Next Steps)

## 🎯 目標
在不破壞現有代碼的情況下，驗證多遊戲框架的可行性。

---

## ⚡ 立即執行 (今天)

### 1. 驗證框架代碼能編譯

```bash
# 編譯核心框架
go build ./internal/game/core/...

# 編譯德撲適配器
go build ./internal/game/poker/...

# 編譯示例程序
go build examples/multi_game_example.go
```

**預期結果**: 無編譯錯誤

---

### 2. 運行示例程序

```bash
go run examples/multi_game_example.go
```

**預期輸出**:
```
✅ Poker table created: poker_table_001
✅ Alice joined the game
✅ Bob joined the game
✅ Action result: {Success:true Message:action queued}
```

**如果成功**: 恭喜！框架基礎已就緒 ✅  
**如果失敗**: 查看錯誤訊息，可能需要修復 import 路徑

---

### 3. 運行現有測試（確保兼容性）

```bash
# 運行 domain 層測試（不應該受影響）
go test ./internal/game/domain/... -v

# 預期: 所有測試通過 (PASS)
```

**如果測試失敗**: 檢查是否不小心修改了 domain 層代碼

---

## 📅 本週任務 (Week 1)

### 任務 1: 修復盲注邏輯 ⚠️ 緊急
**問題**: `internal/game/domain/table.go:69-71` 盲注未實現  
**修復步驟**:

1. 打開 `internal/game/domain/table.go`
2. 找到 `StartHand()` 函數
3. 在發牌後添加盲注邏輯：

```go
// 在 StartHand() 中添加
func (t *Table) postBlinds() {
    sbAmount := int64(10)  // 小盲
    bbAmount := int64(20)  // 大盲

    // 計算盲注位置
    sbPos := (t.DealerPos + 1) % 9
    bbPos := (t.DealerPos + 2) % 9

    // 扣除小盲
    if sb := t.Seats[sbPos]; sb != nil && sb.IsActive() {
        amount := min(sbAmount, sb.Chips)
        sb.Chips -= amount
        sb.CurrentBet = amount
        t.Pots.AddBet(sb.ID, amount)
    }

    // 扣除大盲
    if bb := t.Seats[bbPos]; bb != nil && bb.IsActive() {
        amount := min(bbAmount, bb.Chips)
        bb.Chips -= amount
        bb.CurrentBet = amount
        t.Pots.AddBet(bb.ID, amount)
        t.MinBet = bbAmount
    }
}

// 在 StartHand() 中調用
func (t *Table) StartHand() {
    // ... 現有代碼 ...
    
    t.postBlinds()  // 添加這一行
    
    // ... 現有代碼 ...
}
```

4. 運行測試驗證：
```bash
go test ./internal/game/domain/... -v -run TestTable
```

**預估時間**: 2-3 小時  
**優先級**: 🔴 P0

---

### 任務 2: 添加基礎認證

**目標**: 防止 PlayerID 偽造

1. 創建簡單的 Token 驗證：

```go
// internal/game/adapter/ws/auth.go
package ws

import (
    "errors"
    "net/http"
)

// 臨時解決方案: 簡單的 API Key 驗證
// 生產環境應該使用 JWT
func authenticateRequest(r *http.Request) (string, error) {
    apiKey := r.Header.Get("X-API-Key")
    if apiKey == "" {
        return "", errors.New("missing API key")
    }
    
    // TODO: 從數據庫驗證 API Key
    // 現在先返回固定的 PlayerID
    return "player_" + apiKey, nil
}
```

2. 在 WebSocket Handler 中使用：

```go
// internal/game/adapter/ws/handler.go
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
    // 認證
    playerID, err := authenticateRequest(r)
    if err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    // ... 繼續原有邏輯
}
```

**預估時間**: 2 小時  
**優先級**: 🔴 P0

---

### 任務 3: 整合測試

創建一個端到端測試：

```go
// internal/game/integration_test.go
package game_test

import (
    "testing"
    "github.com/shinjuwu/TheNuts/internal/game/core"
    "github.com/shinjuwu/TheNuts/internal/game/poker"
)

func TestMultiGameFramework_Integration(t *testing.T) {
    // 1. 初始化服務
    gameService := core.NewGameService()
    gameService.RegisterGameEngine(core.GameTypePoker, &poker.PokerEngineFactory{})
    
    // 2. 創建桌子
    config := core.GameConfig{
        GameID:     "test_table",
        MaxPlayers: 9,
        MinBet:     10,
    }
    tableID, err := gameService.CreateGame(core.GameTypePoker, config)
    if err != nil {
        t.Fatalf("Failed to create table: %v", err)
    }
    
    // 3. 玩家加入
    sessionID := gameService.CreateSession("player1", 10000)
    err = gameService.JoinGame(sessionID, tableID, 1000)
    if err != nil {
        t.Fatalf("Failed to join game: %v", err)
    }
    
    // 4. 驗證玩家已在遊戲中
    session, _ := gameService.GetSession(sessionID)
    if session.CurrentGameID != tableID {
        t.Errorf("Player not in game")
    }
    
    t.Log("✅ Integration test passed!")
}
```

運行測試：
```bash
go test ./internal/game/... -v -run TestMultiGameFramework
```

**預估時間**: 2 小時  
**優先級**: 🟡 P1

---

## 📅 下週任務 (Week 2)

### 任務 4: 改造 WebSocket Handler

將現有的 WebSocket Handler 改為使用新的 `GameService`。

**文件**: `internal/game/adapter/ws/handler.go`

**改動點**:
1. 將依賴從 `TableManager` 改為 `GameService`
2. 使用 `PlayerSession` 管理玩家狀態
3. 動作處理改用 `GameService.HandlePlayerAction()`

**參考**: `docs/MIGRATION_GUIDE.md` 第 2.1 節

**預估時間**: 6-8 小時  
**優先級**: 🟡 P1

---

### 任務 5: 實現 Wallet Service

**目標**: 統一的餘額管理

創建 `internal/game/wallet/service.go`:

```go
package wallet

type Service struct {
    // TODO: 連接數據庫
}

func (s *Service) GetBalance(playerID string) (int64, error) {
    // TODO: 從數據庫查詢
    return 0, nil
}

func (s *Service) Deduct(playerID string, amount int64) error {
    // TODO: 扣款，確保原子性
    return nil
}

func (s *Service) Credit(playerID string, amount int64) error {
    // TODO: 加款
    return nil
}
```

**預估時間**: 4-6 小時  
**優先級**: 🟡 P1

---

## 🎯 第一個里程碑 (2 週後)

### 目標狀態
- [x] 多遊戲框架代碼已合併
- [ ] 盲注邏輯修復 ✅
- [ ] 基礎認證實現 ✅
- [ ] 整合測試通過 ✅
- [ ] WebSocket 層使用新框架
- [ ] Wallet Service 實現

### 驗收標準
1. 能夠創建德撲桌並正常遊戲
2. 盲注正確扣除
3. 前端連接需要認證
4. 所有現有測試通過
5. 新增整合測試覆蓋率 > 60%

---

## 🚨 風險預警

### 風險 1: 現有功能退化
**緩解措施**: 每次改動後運行 `go test ./...`

### 風險 2: 性能下降
**緩解措施**: 添加基準測試
```bash
go test -bench=. ./internal/game/...
```

### 風險 3: 時間不足
**緩解措施**: 優先完成 P0 任務，P1/P2 可延後

---

## ✅ 每日檢查清單

- [ ] 今天的代碼已提交 Git
- [ ] 所有測試通過
- [ ] 代碼已經過 Code Review
- [ ] 更新了 TODO.md 的進度
- [ ] 文檔已同步更新

---

## 🆘 遇到問題？

### 技術問題
1. 查看 `docs/QUICK_START.md`
2. 查看 `docs/ARCHITECTURE.md`
3. 運行示例程序 `examples/multi_game_example.go`
4. 查看現有測試 `internal/game/domain/*_test.go`

### 設計問題
1. 查看 `docs/ARCHITECTURE_DIAGRAM.md`
2. 查看 `CODE_REVIEW.md` 了解現有問題

### 遷移問題
1. 查看 `docs/MIGRATION_GUIDE.md`
2. 保持新舊代碼並存，逐步切換

---

## 📊 進度追蹤

更新 TODO.md 中的進度：

```markdown
- [x] 任務名稱 (已完成)
- [ ] 任務名稱 (未開始)
- [~] 任務名稱 (進行中)
```

---

**開始日期**: 2026-01-22  
**預計完成**: 2026-02-05 (2 週)  
**當前狀態**: 🟢 框架設計完成，準備實施

---

**祝你開發順利！有任何問題隨時回來查看文檔。** 🚀
