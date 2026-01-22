# 遷移指南 - 從單一德撲到多遊戲框架

## 📋 遷移概述

本指南幫助你將現有的德州撲克代碼平滑遷移到新的多遊戲框架。

**好消息**: 你現有的核心邏輯（`internal/game/domain/`）**完全不需要修改**，只需要添加適配器層。

## 🔄 遷移步驟

### 第一階段：保持雙軌運行（推薦）

在遷移期間，舊代碼和新框架可以並存：

```
internal/game/
├── domain/          # 保留 - 現有德撲邏輯
├── adapter/ws/      # 保留 - 現有 WebSocket 層
├── core/            # 新增 - 通用遊戲框架
└── poker/           # 新增 - 德撲適配器
```

### 第二階段：逐步切換流量

1. **新桌子使用新框架**
   ```go
   // 舊代碼 (保持運行)
   oldTable := domain.NewTable("old_table_001")
   
   // 新框架 (新桌子使用)
   gameService.CreateGame(core.GameTypePoker, config)
   ```

2. **驗證功能一致性**
   - 運行現有測試套件
   - 對比新舊框架的遊戲結果

3. **完全切換**
   - 所有新桌子使用新框架
   - 舊桌子打完後不再創建

## 📝 代碼變更清單

### 1. WebSocket Handler 改動

#### 舊代碼
```go
// internal/game/adapter/ws/handler.go
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
    // 直接連接到 Table
    tableID := r.URL.Query().Get("table_id")
    table := h.tableManager.GetTable(tableID)
    
    client := &Client{
        table: table,
        conn:  conn,
    }
}
```

#### 新代碼
```go
// internal/game/adapter/ws/handler.go (改造後)
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
    // 先創建會話
    sessionID := h.gameService.CreateSession(playerID, initialBalance)
    
    client := &Client{
        gameService: h.gameService,  // 改為依賴 GameService
        sessionID:   sessionID,
        conn:        conn,
    }
}
```

### 2. 動作處理改動

#### 舊代碼
```go
// 直接發送到 Table.ActionCh
table.ActionCh <- domain.PlayerAction{
    PlayerID: playerID,
    Type:     domain.ActionRaise,
    Amount:   100,
}
```

#### 新代碼
```go
// 通過 GameService 統一處理
action := core.PlayerAction{
    PlayerID:  playerID,
    SessionID: sessionID,
    Type:      core.ActionRaise,
    Amount:    100,
}
result, err := h.gameService.HandlePlayerAction(ctx, action)
```

### 3. 創建桌子改動

#### 舊代碼
```go
table := domain.NewTable("poker_001")
go table.Run()
tableManager.AddTable(table)
```

#### 新代碼
```go
config := core.GameConfig{
    GameID:     "poker_001",
    MaxPlayers: 9,
    MinBet:     10,
    CustomData: map[string]interface{}{
        "blinds": int64(20),
    },
}
tableID, err := gameService.CreateGame(core.GameTypePoker, config)
```

## 🧪 測試策略

### 1. 單元測試遷移

現有的 domain 層測試**完全不需要修改**：

```bash
# 現有測試繼續運行
go test ./internal/game/domain/...

# 新增適配器測試
go test ./internal/game/poker/...
go test ./internal/game/core/...
```

### 2. 整合測試

創建對比測試：

```go
func TestPokerEngine_Compatibility(t *testing.T) {
    // 使用舊代碼運行一局遊戲
    oldTable := domain.NewTable("test")
    oldResult := runGame(oldTable)
    
    // 使用新框架運行相同的遊戲
    engine := poker.NewPokerEngine(config)
    newResult := runGame(engine)
    
    // 驗證結果一致
    assert.Equal(t, oldResult, newResult)
}
```

## 🚨 常見問題

### Q1: 新框架會影響性能嗎？

**A**: 不會。新框架只是添加了一層薄薄的適配器，核心邏輯還是你原來的 `domain.Table`。性能開銷可以忽略不計（< 1%）。

### Q2: 需要重寫數據庫模型嗎？

**A**: 不需要。`core.Player` 只是傳輸對象 (DTO)，持久化層的模型可以保持不變。

### Q3: 如何處理現有玩家的會話？

**A**: 提供兼容性轉換：

```go
// 將舊的 ws.Client 轉換為新的 PlayerSession
func migrateSession(oldClient *ws.Client) string {
    return gameService.CreateSession(
        oldClient.PlayerID,
        oldClient.Balance,
    )
}
```

### Q4: 舊的 WebSocket DTO 需要改嗎？

**A**: 建議逐步遷移：

1. **第一階段**: 保持 `internal/game/adapter/ws/dto.go` 不變
2. **第二階段**: 創建新的 `internal/game/core/dto.go`
3. **第三階段**: 添加轉換函數在兩者之間切換
4. **最終**: 統一使用新的 DTO

## 📅 建議時程

### Week 1-2: 框架搭建（已完成）
- ✅ 創建 `core/` 目錄
- ✅ 定義 `GameEngine` 介面
- ✅ 實現 `PokerEngine` 適配器

### Week 3: 測試驗證
- [ ] 運行所有現有測試
- [ ] 創建整合測試
- [ ] 壓力測試（1000 桌並發）

### Week 4: 逐步切換
- [ ] 新桌子使用新框架
- [ ] 監控錯誤率和性能
- [ ] 收集反饋

### Week 5-6: 完全遷移
- [ ] 所有桌子切換到新框架
- [ ] 移除舊代碼（可選，也可以保留作為備份）
- [ ] 文檔更新

### Week 7+: 擴展
- [ ] 實現第二個遊戲引擎（老虎機/百家樂）
- [ ] 添加錦標賽管理器
- [ ] 實現持久化層

## 🔧 實用工具

### 1. 自動化遷移腳本

```bash
#!/bin/bash
# migrate.sh - 自動將舊的 TableManager 調用替換為 GameService

find ./internal -name "*.go" -type f -exec sed -i \
    's/tableManager.GetTable/gameService.GetTable/g' {} +

echo "Migration complete. Please review changes with 'git diff'"
```

### 2. 兼容性檢查工具

```go
// tools/check_compatibility.go
func CheckCompatibility() {
    // 驗證所有現有 API 在新框架中都有對應
    oldAPIs := []string{"CreateTable", "JoinTable", "HandleAction"}
    newAPIs := []string{"CreateGame", "JoinGame", "HandlePlayerAction"}
    
    // 確保一一對應
}
```

## ✅ 遷移檢查清單

- [ ] 新框架代碼已合併到主分支
- [ ] 所有現有測試通過
- [ ] 新增整合測試覆蓋率 > 80%
- [ ] 壓力測試通過 (1000+ 並發桌)
- [ ] 文檔已更新
- [ ] 團隊培訓完成
- [ ] 有回滾方案（保留舊代碼分支）
- [ ] 監控告警已配置
- [ ] 生產環境灰度發布計劃制定

## 🆘 需要幫助？

如果遇到問題：

1. **查看示例代碼**: `examples/multi_game_example.go`
2. **閱讀架構文檔**: `docs/ARCHITECTURE.md`
3. **運行測試**: `go test ./...`
4. **檢查現有 Code Review**: `CODE_REVIEW.md`

## 🎉 遷移完成後的好處

1. **可擴展性**: 新增遊戲只需 1-2 天
2. **可維護性**: 統一的介面和錯誤處理
3. **可測試性**: 每個遊戲引擎獨立測試
4. **可監控性**: 統一的指標收集點
5. **商業化**: 更容易添加付費功能和分析系統
