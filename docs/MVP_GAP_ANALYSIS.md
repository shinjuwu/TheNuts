# MVP 差距分析 - TheNuts 德州撲克

**分析日期**: 2026-01-27  
**當前狀態**: 核心基礎設施完成 (~85%)，遊戲邏輯待整合 (~15%)

---

## 📊 整體進度概覽

```
總體完成度: ████████████░░░░ 75%

✅ 已完成 (75%)
🟡 進行中 (15%)  
⭕ 未開始 (10%)
```

---

## ✅ 已完成的功能 (75%)

### 1. 基礎設施層 (100%)
- ✅ PostgreSQL 連接池 (25 max, 5 min connections)
- ✅ Redis 客戶端 (連接池 size=10)
- ✅ 依賴注入系統 (Wire DI)
- ✅ 配置管理 (支援多環境)
- ✅ 日誌系統 (Zap logger)
- ✅ 錯誤處理機制

### 2. 認證系統 (100%)
- ✅ 用戶註冊 (bcrypt 密碼雜湊, cost=12)
- ✅ 用戶登入 (JWT Token, 24小時有效)
- ✅ WebSocket Ticket 機制 (30秒一次性使用)
- ✅ 帳號鎖定機制 (5次失敗 → 30分鐘鎖定)
- ✅ 帳號狀態管理 (active/suspended/banned)
- ✅ IP 追蹤與失敗登入計數

**API 端點**:
- `POST /api/auth/register` - 註冊
- `POST /api/auth/login` - 登入
- `POST /api/auth/ticket` - 獲取 WS Ticket

### 3. 數據持久化層 (100%)
- ✅ Account Repository (帳號管理)
- ✅ Player Repository (玩家資料)
- ✅ Wallet Repository (錢包管理)
- ✅ Transaction Repository (交易記錄)
- ✅ GameSession Repository (遊戲會話)
- ✅ UnitOfWork 模式 (事務安全)

**資料庫表**:
```sql
accounts          -- 用戶帳號
players           -- 玩家資料
wallets           -- 錢包餘額
transactions      -- 交易歷史
game_sessions     -- 遊戲會話
```

### 4. 遊戲服務層 (100%)
- ✅ GameService (遊戲核心服務)
  - ✅ `BuyIn()` - 買入籌碼 (真實扣款)
  - ✅ `CashOut()` - 兌現籌碼 (返回錢包)
  - ✅ `GetPlayerBalance()` - 查詢餘額
  - ✅ `GetActiveSession()` - 獲取活躍會話
  - ✅ `UpdateSessionChips()` - 更新籌碼
  - ✅ `EnsureWalletExists()` - 自動創建錢包

**交易安全**:
- ✅ 資料庫事務保護
- ✅ Idempotency Key (防止重複扣款)
- ✅ 餘額檢查 (扣款前驗證)
- ✅ 自動回滾機制

### 5. WebSocket 基礎設施 (100%)
- ✅ WebSocket Handler (連接管理)
- ✅ Hub 系統 (客戶端註冊/註銷)
- ✅ SessionManager (會話生命週期管理)
  - ✅ 自動清理 (每 1 分鐘檢查)
  - ✅ 會話超時 (30 分鐘無活動)
  - ✅ 自動兌現機制 (超時後自動 cash-out)
- ✅ MessageHandler (8 種訊息類型)
  - ✅ `BUY_IN` - 買入
  - ✅ `CASH_OUT` - 兌現
  - ✅ `JOIN_TABLE` - 加入牌桌
  - ✅ `LEAVE_TABLE` - 離開牌桌
  - ✅ `SIT_DOWN` - 坐下
  - ✅ `STAND_UP` - 站起
  - ✅ `GAME_ACTION` - 遊戲動作
  - ✅ `GET_BALANCE` - 查詢餘額

**WebSocket 配置**:
- Ping 間隔: 54 秒
- Pong 超時: 60 秒
- 訊息緩衝: 256 則/客戶端
- 讀取超時: 60 秒
- 寫入超時: 10 秒

### 6. 領域模型 (90%)
- ✅ Card (撲克牌)
- ✅ Deck (牌組 + 洗牌)
- ✅ Evaluator (牌型評估器)
- ✅ PotManager (底池管理)
- ✅ Player (玩家狀態)
- ✅ Table (牌桌 + Run 循環)
- ✅ Distributor (發牌器)
- ⭕ **遊戲流程整合 (缺)**

---

## 🟡 進行中/需完成 (25%)

### 1. 遊戲邏輯整合 (60% 完成)

#### ✅ 已有的組件
- `internal/game/domain/table.go` - 牌桌邏輯 (Run 循環, 動作處理)
- `internal/game/domain/evaluator.go` - 牌型評估
- `internal/game/poker/poker_engine.go` - 撲克引擎框架
- `internal/game/adapter/ws/message_handler.go` - 訊息路由

#### ⭕ 缺少的部分

**A. 自動遊戲流程 (Critical - MVP 必須)**
```go
// 需要實現: 當有足夠玩家時自動開始遊戲
func (t *Table) AutoStartGame() {
    // 1. 檢查是否有足夠玩家 (至少 2 人)
    // 2. 等待玩家準備 (SitDown 後自動標記為 Ready)
    // 3. 自動開始新手牌 (StartHand)
    // 4. 自動處理超時 (玩家未行動時自動 Fold)
}
```

**位置**: `internal/game/domain/table.go`  
**預估工作量**: 2-3 小時

---

**B. Showdown 邏輯 (Critical - MVP 必須)**
```go
// 需要實現: 攤牌與結算
func (t *Table) Showdown() {
    // 1. 收集所有未棄牌玩家的手牌
    // 2. 使用 Evaluator 評估牌力
    // 3. 確定贏家
    // 4. 分配彩池 (支援邊池 side pots)
    // 5. 更新玩家籌碼
    // 6. 廣播結果
}
```

**位置**: `internal/game/domain/table.go`  
**預估工作量**: 3-4 小時

---

**C. 狀態同步到資料庫 (Important - MVP 必須)**
```go
// 需要實現: 每個關鍵操作後更新資料庫
func (h *MessageHandler) syncChipsToDatabase(playerID uuid.UUID, chips int64) {
    // 在以下時機調用:
    // 1. 每手牌結束後
    // 2. 玩家離桌時
    // 3. 定期同步 (每 5 分鐘)
    
    h.gameService.UpdateSessionChips(ctx, sessionID, chips)
}
```

**位置**: `internal/game/adapter/ws/message_handler.go`  
**預估工作量**: 1-2 小時

---

**D. 遊戲事件廣播 (Important - MVP 建議)**
```go
// 需要實現: 即時廣播遊戲事件
type GameEventBroadcaster struct {
    sessionManager *SessionManager
}

func (b *GameEventBroadcaster) BroadcastGameEvent(tableID string, event GameEvent) {
    // 廣播事件類型:
    // - HAND_START (新手牌開始)
    // - CARDS_DEALT (發牌)
    // - PLAYER_ACTION (玩家行動)
    // - COMMUNITY_CARDS (公共牌)
    // - HAND_END (手牌結束)
    // - WINNER_DECLARED (贏家宣布)
}
```

**位置**: `internal/game/adapter/ws/event_broadcaster.go` (新文件)  
**預估工作量**: 2-3 小時

---

**E. 超時處理機制 (Important - MVP 建議)**
```go
// 需要實現: 玩家行動超時自動處理
type ActionTimer struct {
    timeout time.Duration // 預設 30 秒
}

func (t *ActionTimer) StartTimer(playerID string, callback func()) {
    // 倒數計時
    // 超時後自動執行 callback (通常是 Auto-Fold)
}
```

**位置**: `internal/game/domain/timer.go` (新文件)  
**預估工作量**: 2 小時

---

### 2. 前端客戶端 (0% 完成)

#### ⭕ MVP 最小前端需求

**A. WebSocket 客戶端**
```javascript
// 需要實現基本的 WS 客戶端
class PokerClient {
    connect(ticket)      // 連接 WebSocket
    buyIn(amount)        // 買入
    joinTable(tableId)   // 加入牌桌
    sendAction(action)   // 發送動作 (FOLD/CHECK/CALL/BET/RAISE)
    cashOut()            // 兌現
}
```

**B. 基本 UI 組件**
- 登入/註冊頁面
- 牌桌視圖 (9 個座位)
- 手牌顯示
- 公共牌顯示
- 動作按鈕 (Fold/Check/Call/Bet/Raise)
- 籌碼顯示
- 底池顯示

**技術選擇建議**:
- React + TypeScript (推薦)
- 或 Vue 3 + TypeScript
- WebSocket 客戶端庫

**預估工作量**: 8-12 小時 (基本功能)

---

### 3. 測試與驗證 (20% 完成)

#### ✅ 已有的測試
- `internal/game/domain/*_test.go` - 領域模型單元測試
- `test_auth.ps1` - 認證流程測試腳本
- `test_websocket.ps1` - WebSocket 流程測試腳本

#### ⭕ 需要補充的測試

**A. 集成測試**
```bash
# 完整遊戲流程測試
1. 2 個玩家註冊並登入
2. 兩人都買入 1000 籌碼
3. 加入同一張牌桌
4. 完成一手牌 (Preflop → Flop → Turn → River → Showdown)
5. 驗證籌碼轉移正確
6. 驗證資料庫記錄正確
```

**預估工作量**: 3-4 小時

---

**B. 負載測試**
```bash
# 測試目標:
- 100 個並發 WebSocket 連接
- 10 張牌桌同時運行
- 每秒 50 個動作請求
```

**工具**: k6 或 artillery  
**預估工作量**: 2-3 小時

---

## ⭕ MVP 之後的功能 (Nice to Have)

### 1. 高級功能
- [ ] 手牌歷史記錄 (Hand History)
- [ ] 重播功能 (Hand Replay)
- [ ] 錦標賽模式 (Tournament)
- [ ] Sit & Go 模式
- [ ] 多桌支援 (Multi-table)
- [ ] 觀眾模式 (Spectator)

### 2. 優化與監控
- [ ] Redis 緩存整合
  - 快取玩家資料
  - 快取牌桌狀態
  - Session 存到 Redis
- [ ] 審計日誌系統 (Audit Log)
- [ ] Prometheus 監控指標
- [ ] 告警系統 (Alert)
- [ ] 效能分析 (Profiling)

### 3. 安全性增強
- [ ] Rate Limiting (API 限流)
- [ ] DDOS 防護
- [ ] 防作弊機制
  - 動作時間分析
  - 異常行為偵測
- [ ] 數據加密 (傳輸層)

### 4. 運營功能
- [ ] 管理後台 (Admin Panel)
- [ ] 玩家管理
- [ ] 遊戲配置管理
- [ ] 統計報表
- [ ] 財務報表

---

## 📋 MVP 待辦清單 (優先順序排序)

### 🔴 Critical (必須完成才能運行)

| 任務 | 檔案位置 | 預估時間 | 狀態 |
|------|---------|---------|------|
| 1. Showdown 邏輯 | `domain/table.go` | 3-4h | ⭕ |
| 2. 自動遊戲流程 | `domain/table.go` | 2-3h | ⭕ |
| 3. 狀態同步到資料庫 | `ws/message_handler.go` | 1-2h | ⭕ |
| 4. 基本前端客戶端 | `frontend/` (新建) | 8-12h | ⭕ |

**Critical 總計**: 14-21 小時

---

### 🟡 Important (MVP 建議有)

| 任務 | 檔案位置 | 預估時間 | 狀態 |
|------|---------|---------|------|
| 5. 遊戲事件廣播 | `ws/event_broadcaster.go` | 2-3h | ⭕ |
| 6. 超時處理機制 | `domain/timer.go` | 2h | ⭕ |
| 7. 集成測試 | `tests/integration_test.go` | 3-4h | ⭕ |
| 8. 錯誤處理完善 | 各文件 | 2h | ⭕ |

**Important 總計**: 9-13 小時

---

### 🟢 Nice to Have (MVP 後再做)

| 任務 | 預估時間 |
|------|---------|
| 9. Redis 緩存整合 | 4-6h |
| 10. 審計日誌 | 3-4h |
| 11. 負載測試 | 2-3h |
| 12. 管理後台 | 12-16h |

---

## ⏱️ MVP 時間預估

```
Critical 任務:  14-21 小時
Important 任務:  9-13 小時
-------------------------
MVP 總計:      23-34 小時

預估完成時間: 3-5 個工作天 (每天 8 小時)
```

---

## 🎯 MVP 定義

**MVP 功能範圍**:
1. ✅ 用戶可以註冊、登入
2. ✅ 用戶可以買入籌碼
3. ✅ 用戶可以加入牌桌
4. ⭕ **2+ 玩家可以完成一手牌** (Critical - 待完成)
   - Preflop → Flop → Turn → River → Showdown
   - 正確的牌型評估
   - 正確的籌碼轉移
5. ⭕ **即時顯示遊戲狀態** (Critical - 待完成)
   - 手牌
   - 公共牌
   - 底池
   - 玩家動作
6. ✅ 用戶可以兌現籌碼
7. ⭕ **基本錯誤處理** (Important - 部分完成)
8. ⭕ **基本測試覆蓋** (Important - 部分完成)

---

## 🚀 下一步行動建議

### 立即開始 (今天)
1. **實現 Showdown 邏輯** (3-4h)
   - 使用現有的 `Evaluator`
   - 實現邊池分配
   - 更新玩家籌碼

2. **實現自動遊戲流程** (2-3h)
   - 玩家準備機制
   - 自動開始新手牌
   - 處理玩家不足情況

### 明天
3. **實現狀態同步** (1-2h)
   - 每手牌結束同步
   - 定期同步機制

4. **開始前端開發** (8-12h)
   - 設置項目結構
   - 實現 WebSocket 客戶端
   - 基本 UI 組件

### 第三天
5. **遊戲事件廣播** (2-3h)
6. **集成測試** (3-4h)
7. **Bug 修復與優化** (2-3h)

---

## 📊 風險評估

| 風險 | 機率 | 影響 | 緩解措施 |
|------|------|------|---------|
| Showdown 邏輯複雜 (邊池) | 中 | 高 | 先實現簡單情況，邊池後續迭代 |
| 前端開發延期 | 高 | 中 | 使用測試腳本先驗證後端 |
| 並發問題 | 中 | 高 | 充分的集成測試與負載測試 |
| 籌碼同步錯誤 | 低 | 極高 | 嚴格的事務控制與測試 |

---

## ✅ 結論

**當前狀態**: 基礎設施 100% 完成，遊戲邏輯整合 60% 完成

**距離 MVP**: 
- **Critical 任務**: 4 項，14-21 小時
- **Important 任務**: 4 項，9-13 小時
- **總計**: 23-34 小時 (3-5 個工作天)

**最大挑戰**:
1. Showdown 邏輯正確性 (特別是邊池)
2. 前端開發時間
3. 完整的集成測試

**優勢**:
1. ✅ 堅實的基礎設施
2. ✅ 完整的資料持久化層
3. ✅ 健全的認證與安全機制
4. ✅ 良好的架構設計

**建議**: 專注於 Critical 任務，優先完成核心遊戲循環，前端可以使用簡單的 HTML + JavaScript 快速驗證功能。
