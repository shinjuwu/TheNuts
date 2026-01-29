# MVP to Production — 待處理問題清單

本文檔記錄從 MVP 邁向正式版需要解決的所有已知問題。
按類別分組，每個項目標註嚴重度、所在位置及建議做法。

---

## 1. 併發安全 (Concurrency)

### 1.1 Table.Players 資料競爭

- **嚴重度**：高
- **位置**：
  - `internal/game/domain/table.go:53-61` (`PlayerSitDown`)
  - `internal/game/domain/table.go:64-72` (`PlayerStandUp`)
  - `internal/game/adapter/ws/message_handler.go:294, 336` (handleSitDown/handleStandUp)
  - `internal/game/adapter/ws/message_handler.go:237-240` (handleJoinTable 直接寫 Players map)
- **問題**：`handleSitDown`、`handleStandUp`、`handleJoinTable` 在 WebSocket goroutine 中直接讀寫 `table.Players`，而 `Table.Run()` 在另一個 goroutine 中也讀寫同一份資料。
- **建議**：統一透過 `ActionCh` 序列化所有對 Table 狀態的變更。定義新的 action 類型（如 `ActionSitDown`、`ActionStandUp`、`ActionJoinTable`），讓 `Run()` 的 select loop 統一處理。

### 1.2 OnHandComplete 同步阻塞 Run()

- **嚴重度**：中
- **位置**：`internal/game/domain/table.go:523-525`
- **問題**：`endHand()` 內同步呼叫 `OnHandComplete(t)`，此回調執行 DB I/O（`table_manager.go:47-74`）。如果資料庫慢或超時，會阻塞整張桌子的 action 處理和 ticker。
- **建議**：改為非同步執行（`go t.OnHandComplete(t)` 或丟到專用 channel），但需確保下一手牌不會在同步完成前讀到不一致狀態。可考慮加一個 sync barrier 或在回調完成前不進入下一手。

---

## 2. 安全性 (Security)

### 2.1 WebSocket Origin 檢查未啟用

- **嚴重度**：高
- **位置**：`internal/game/adapter/ws/handler.go:19`
- **問題**：`CheckOrigin` 固定回傳 `true`，允許任何來源的 WebSocket 連線，存在 CSRF/跨站 WebSocket 劫持風險。
- **建議**：實作 Origin 白名單機制：
  ```go
  CheckOrigin: func(r *http.Request) bool {
      origin := r.Header.Get("Origin")
      return isAllowedOrigin(origin)
  }
  ```

### 2.2 會話 ID 生成不安全

- **嚴重度**：中
- **位置**：`internal/game/core/service.go:182`
- **問題**：`generateSessionID()` 用 `time.Now().UnixNano()` 生成，可預測且可碰撞。
- **建議**：改用 `uuid.New()` 或 `crypto/rand` 生成。

---

## 3. 斷線處理 (Disconnection)

### 3.1 玩家斷線未通知牌桌

- **嚴重度**：高
- **位置**：`internal/game/adapter/ws/session.go:262`
- **問題**：玩家斷線後只記了日誌，未通知牌桌。如果輪到該玩家行動，牌局會永遠卡住。
- **建議**：
  1. 設定玩家為 `StatusDisconnected`（需新增狀態）
  2. 啟動重連計時器（例如 60 秒）
  3. 超時未重連 → 自動 Fold 並 StandUp
  4. 如果不在手牌中 → 直接 StandUp

### 3.2 斷線重連序列化未實作

- **嚴重度**：中
- **位置**：`internal/game/poker/poker_engine.go:292`
- **問題**：`Serialize()` 回傳 `nil, nil`，斷線重連時無法恢復遊戲狀態。
- **建議**：實作 Table 狀態的 JSON 序列化，包含公牌、底池、玩家狀態等，供重連客戶端恢復畫面。

---

## 4. 遊戲邏輯 (Game Logic)

### 4.1 底池餘數分配不符標準規則

- **嚴重度**：低
- **位置**：`internal/game/domain/distributor.go:58`
- **問題**：多人平分底池時的餘數（例如 3 人分 100，每人 33 餘 1）目前按 slice 順序分配，標準規則應分配給最靠近 Button 左手邊（即位置最早）的玩家。
- **建議**：`Distribute` 函數需要接收 `DealerPos` 和座位資訊，按座位順序從 Button 左邊開始分配餘數。

### 4.2 handleLeaveTable 未從 domain 層移除玩家

- **嚴重度**：中
- **位置**：`internal/game/adapter/ws/message_handler.go:276-277`
- **問題**：`handleLeaveTable` 只更新了 session 狀態，沒有從 `table.Players` 和 `table.Seats` 移除玩家。離開的玩家仍會被發牌。
- **建議**：實作 `Table.RemovePlayer(playerID)` 方法，處理：
  1. 如果在手牌中 → 先 StandUp（自動 Fold）
  2. 從 `Players` map 和 `Seats` array 移除
  3. 廣播更新

### 4.3 盲注金額硬編碼

- **嚴重度**：低
- **位置**：`internal/game/domain/table.go:84`
- **問題**：`MinBet = 20` 硬編碼，所有桌子都是 10/20 盲注。
- **建議**：在 `Table` struct 增加 `SmallBlind`/`BigBlind` 配置欄位，建桌時由外部傳入。

---

## 5. 可觀測性 (Observability)

### 5.1 Domain 層大量使用 fmt.Printf

- **嚴重度**：中
- **位置**：`internal/game/domain/table.go` (15+ 處), `internal/game/table_manager.go:53,67`
- **問題**：所有遊戲事件（發牌、盲注、Showdown 等）都用 `fmt.Printf` 輸出到 stdout，正式環境：
  - 無法控制日誌級別
  - 無結構化欄位，難以搜尋/分析
  - 高流量下 stdout I/O 可能成為瓶頸
- **建議**：
  - Domain 層：改為透過事件回調（類似 `OnHandComplete`）或注入 logger interface
  - `TableManager`：注入 `*zap.Logger`
  - 正式環境使用 structured logging（`zap.Info("dealing flop", zap.String("table_id", ...), zap.Strings("cards", ...))`）

---

## 6. 測試穩定性 (Test Reliability)

### 6.1 TestAutoGameFlow 依賴 time.Sleep

- **嚴重度**：低
- **位置**：`internal/game/domain/auto_game_test.go:29`
- **問題**：用 `time.Sleep(1500ms)` 等 ticker 觸發，CI 環境或高負載機器上可能 flaky。
- **建議**：改為事件驅動測試。例如在 `Table` 增加 `OnHandStarted` 回調，測試中用 channel 等待開局信號而非 sleep。

---

## 7. 基礎設施 (Infrastructure)

### 7.1 Redis 整合未完成

- **嚴重度**：中
- **位置**：`internal/infra/redis/`（計劃中）
- **問題**：票券（ticket）目前存在記憶體中（推測），多實例部署時票券無法共享。
- **建議**：將票券存儲遷移到 Redis，支持 TTL 自動過期和多實例部署。

### 7.2 缺少限流與 DDoS 防護

- **嚴重度**：中
- **問題**：WebSocket 和 HTTP 端點無 rate limiting。
- **建議**：
  - HTTP API：加入 rate limiter middleware（per-IP / per-user）
  - WebSocket：限制訊息頻率（per-connection）
  - 考慮使用 `golang.org/x/time/rate`

### 7.3 缺少 Prometheus 指標

- **嚴重度**：低
- **問題**：無法監控在線人數、手牌數/秒、延遲分布等關鍵指標。
- **建議**：整合 `prometheus/client_golang`，暴露 `/metrics` 端點，追蹤：
  - 活躍連線數
  - 每秒手牌完成數
  - action 處理延遲
  - DB 操作延遲

---

## 優先順序建議

| 優先級 | 項目 | 原因 |
|--------|------|------|
| **P0** | 1.1 併發安全 | 正式環境高併發下必崩 |
| **P0** | 2.1 Origin 檢查 | 安全漏洞 |
| **P0** | 3.1 斷線通知 | 會導致牌局卡死 |
| **P1** | 4.2 LeaveTable 移除 | 功能缺失 |
| **P1** | 5.1 替換 fmt.Printf | 正式環境不可用 |
| **P1** | 1.2 同步阻塞 | DB 慢時影響遊戲體驗 |
| **P1** | 7.1 Redis 整合 | 多實例部署必需 |
| **P2** | 2.2 會話 ID | 安全性強化 |
| **P2** | 3.2 斷線重連 | 用戶體驗 |
| **P2** | 7.2 限流防護 | 防禦性需求 |
| **P2** | 4.1 餘數分配 | 規則正確性 |
| **P2** | 4.3 盲注配置化 | 產品靈活性 |
| **P3** | 6.1 測試穩定性 | CI 品質 |
| **P3** | 7.3 Prometheus | 運維可觀測性 |
