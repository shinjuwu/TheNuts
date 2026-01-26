# 下一步行動計劃 (Next Steps)

## 🎯 目標
在不破壞現有代碼的情況下，完成多遊戲框架的整合。

---

## ✅ 已完成的任務 (2026-01-26)

### 任務 1: ~~修復盲注邏輯~~ ✅ 完成
**完成時間**: 2026-01-26  
**實際耗時**: ~4 小時  
**測試結果**: 7/7 通過 ✅

#### 實作內容
- ✅ `postBlinds()` 函數 (table.go:78-195)
- ✅ 標準多人桌盲注 (3-9人)
- ✅ Heads-Up 特殊規則 (Button = SB)
- ✅ 籌碼不足自動 All-in
- ✅ 單人桌保護
- ✅ 完整測試 (blinds_test.go)

**代碼位置**: `internal/game/domain/table.go:78-195`

---

### 任務 2: ~~WebSocket 認證系統~~ ✅ 完成
**完成時間**: 2026-01-26  
**實際耗時**: ~8 小時 (遠超原計劃)  
**實作方式**: 票券機制 (業界最佳實踐)

#### 實作內容 (超出原計劃)
- ✅ JWT Token 生成與驗證 (HMAC-SHA256)
- ✅ 票券機制 (30秒一次性票券)
- ✅ MemoryTicketStore 實作
- ✅ 自動清理過期票券機制
- ✅ JWT 中介層 (middleware.go)
- ✅ 認證 Handler (login, ticket 端點)
- ✅ 完整認證流程 (登入 → JWT → 票券 → WebSocket)
- ✅ 測試客戶端 (test-client.html)
- ✅ 完整文檔 (1000+ 行)

**代碼位置**: `internal/auth/`  
**文檔**: `AUTHENTICATION_IMPLEMENTATION_SUMMARY.md`, `docs/AUTHENTICATION.md`

---

### 任務 3: ~~邊池邏輯驗證~~ ✅ 完成
**完成時間**: 2026-01-26  
**測試結果**: 3/3 通過 ✅  
**演算法**: Slicing Algorithm (業界標準)

#### 驗證內容
- ✅ Main Pot 計算正確
- ✅ Side Pot 自動切分
- ✅ 多輪下注合併
- ✅ Contributors 追蹤
- ✅ Distributor 正確過濾 Folded 玩家
- ✅ 複雜場景測試 (P1:100, P2:200, P3:500)

**代碼位置**: `internal/game/domain/pot.go`  
**測試檔案**: `internal/game/domain/pot_test.go`

---

## 📅 當前任務 (Week 2)

### ⚡ 立即可開始的任務

#### 任務 4: 持久化層設計 🟡 P1
**目標**: 設計數據庫 Schema，為商業化做準備

**步驟**:
1. 設計 PostgreSQL Schema
   - Players 表 (玩家基本資料)
   - Accounts 表 (帳號登入資訊)
   - Wallets 表 (玩家餘額)
   - Games 表 (遊戲記錄)
   - HandHistory 表 (手牌歷史)
   - Transactions 表 (資金流水)
   
2. 創建資料庫遷移腳本
   ```bash
   # 使用 golang-migrate
   migrate create -ext sql -dir migrations -seq init_schema
   ```

3. 實作 Repository 介面
   ```go
   // internal/infra/repository/player_repo.go
   type PlayerRepository interface {
       GetByID(ctx context.Context, id string) (*Player, error)
       Create(ctx context.Context, player *Player) error
       Update(ctx context.Context, player *Player) error
   }
   ```

**預估時間**: 12-16 小時  
**優先級**: 🟡 P1

---

#### 任務 5: Wallet Service 實作 🟡 P1
**目標**: 統一的餘額管理系統

**設計考量**:
- 交易原子性 (Database Transaction)
- 防止重複扣款 (Idempotency Key)
- 餘額檢查與鎖定
- 交易記錄審計

**實作步驟**:
```go
// internal/game/wallet/service.go
package wallet

type Service struct {
    repo TransactionRepository
}

func (s *Service) GetBalance(ctx context.Context, playerID string) (int64, error) {
    // 查詢當前餘額
}

func (s *Service) Deduct(ctx context.Context, playerID string, amount int64, reason string) error {
    // 1. 開始 Transaction
    // 2. 鎖定餘額 (SELECT FOR UPDATE)
    // 3. 檢查餘額充足
    // 4. 扣款
    // 5. 記錄交易
    // 6. Commit
}

func (s *Service) Credit(ctx context.Context, playerID string, amount int64, reason string) error {
    // 加款邏輯
}
```

**預估時間**: 8-10 小時  
**優先級**: 🟡 P1

---

#### 任務 6: WebSocket Handler 改造 🟡 P1
**目標**: 使用 GameService 替代直接操作 Table

**改動點**:
1. 將依賴從 `TableManager` 改為 `GameService`
2. 使用 `PlayerSession` 管理玩家狀態
3. 動作處理改用 `GameService.HandlePlayerAction()`

**參考文檔**: `docs/MIGRATION_GUIDE.md` 第 2.1 節

**預估時間**: 6-8 小時  
**優先級**: 🟡 P1

---

## 🎯 里程碑檢查 (更新)

### 第一個里程碑 (2 週後)

#### 目標狀態
- [x] 多遊戲框架代碼已合併 ✅
- [x] 盲注邏輯修復 ✅ (2026-01-26)
- [x] 認證系統實現 ✅ (2026-01-26)
- [x] 邊池邏輯驗證 ✅ (2026-01-26)
- [ ] 持久化層設計 (進行中)
- [ ] Wallet Service 實現 (待開始)
- [ ] WebSocket 層使用 GameService (待開始)
- [ ] 整合測試通過 (待補充)

#### 驗收標準
1. ✅ 能夠創建德撲桌並正常遊戲
2. ✅ 盲注正確扣除
3. ✅ 前端連接需要認證 (票券機制)
4. ✅ 所有現有測試通過 (23/23)
5. ⏳ 新增整合測試覆蓋率 > 60%
6. ⏳ 持久化層就緒
7. ⏳ Wallet Service 可用

---

## 📊 進度追蹤 (更新)

### 總體進度
- **總任務數**: 42
- **已完成**: 10 (24%) ⬆️
- **進行中**: 3 (7%)
- **未開始**: 29 (69%)

### 測試統計
```
Domain 層測試:  23/23 通過 (100%) ✅
認證系統:       功能完整 ✅
完整遊戲流程:   測試通過 ✅
```

### P0 任務完成度
```
盲注邏輯:  ✅ 完成 (2026-01-26)
認證機制:  ✅ 完成 (2026-01-26)
邊池驗證:  ✅ 完成 (2026-01-26)
```

### 最近成就 (2026-01-26)
- ✅ 完成所有 P0 緊急任務
- ✅ 實作超出計劃的票券認證系統
- ✅ 23個 Domain 測試全部通過
- ✅ 生產就緒的盲注和邊池邏輯

---

## 🚀 下一步建議

### 本週 (Week 2)
1. **持久化層設計** - Schema 設計 + 遷移腳本
2. **Wallet Service** - 核心餘額管理
3. **整合測試** - 補充端到端測試

### 下週 (Week 3)
4. **WebSocket Handler 改造** - 使用 GameService
5. **審計日誌** - 記錄所有關鍵操作
6. **斷線重連** - Session 持久化

### 長期 (Week 4-6)
7. **錦標賽管理器** - 盲注結構、併桌邏輯
8. **第二個遊戲引擎** - 驗證框架通用性
9. **監控和上線** - Prometheus + Grafana

---

## ✅ 每日檢查清單

- [x] 盲注邏輯已完成 (2026-01-26)
- [x] 認證系統已完成 (2026-01-26)
- [x] 邊池邏輯已驗證 (2026-01-26)
- [x] 所有測試通過
- [x] 文檔已同步更新
- [ ] 準備下一階段任務

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

**開始日期**: 2026-01-22  
**P0 完成**: 2026-01-26 ✅  
**預計整合完成**: 2026-02-05 (2 週)  
**當前狀態**: 🟢 核心功能完成，準備持久化層開發

---

**最後更新**: 2026-01-26  
**祝你開發順利！** 🚀
