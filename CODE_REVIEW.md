# TheNuts 德州撲克遊戲服務器 - 代碼審查報告

**審查日期**: 2026-01-22  
**審查範圍**: 完整代碼庫  
**項目階段**: Alpha/Prototype  

---

## 執行摘要 (Executive Summary)

TheNuts 是一個使用 Go 語言開發的德州撲克遊戲服務器，採用 **Domain-Driven Design (DDD)** 和 **六邊形架構 (Hexagonal Architecture)**。核心撲克邏輯實現優秀，架構清晰，但在生產就緒性方面存在一些關鍵問題需要解決。

### 整體評分

| 類別 | 評分 | 說明 |
|-----|------|------|
| **架構設計** | ⭐⭐⭐⭐⭐ | 優秀的 DDD 和六邊形架構實現 |
| **代碼質量** | ⭐⭐⭐⭐ | 清晰、可維護，有良好的測試覆蓋 |
| **安全性** | ⭐⭐ | 缺乏認證、授權和輸入驗證 |
| **穩定性** | ⭐⭐⭐ | 核心邏輯穩定，但缺少錯誤處理 |
| **生產就緒度** | ⭐⭐ | 缺少持久化、監控、日誌等關鍵功能 |

---

## 🔴 嚴重問題 (Critical Issues)

### 1. 盲注邏輯未實現

**位置**: `internal/game/domain/table.go:69-71`

```go
// 4. 設定盲注 (Blind)
// 這裡簡化處理，假設 DealerPos + 1 是 SB, + 2 是 BB
// TODO: 需處理人數少於 2 的情況
```

**問題描述**:
- 盲注沒有自動扣除
- 玩家不會被強制下盲注
- 遊戲無法按照標準德州撲克規則開始

**影響**: 🔴 **遊戲破壞性 Bug** - 遊戲無法正常進行

**建議修復**:
```go
func (t *Table) postBlinds() {
    smallBlindAmount := int64(10)
    bigBlindAmount := int64(20)
    
    // 計算盲注位置
    sbPos := (t.DealerPos + 1) % 9
    bbPos := (t.DealerPos + 2) % 9
    
    // 小盲注
    if sb := t.Seats[sbPos]; sb != nil && sb.IsActive() {
        amount := min(smallBlindAmount, sb.Chips)
        sb.Chips -= amount
        sb.CurrentBet = amount
        t.Pots.Pots[0].Amount += amount
        t.Pots.Pots[0].Contributors[sb.ID] = true
        
        if sb.Chips == 0 {
            sb.Status = StatusAllIn
        }
    }
    
    // 大盲注
    if bb := t.Seats[bbPos]; bb != nil && bb.IsActive() {
        amount := min(bigBlindAmount, bb.Chips)
        bb.Chips -= amount
        bb.CurrentBet = amount
        t.Pots.Pots[0].Amount += amount
        t.Pots.Pots[0].Contributors[bb.ID] = true
        
        if bb.Chips == 0 {
            bb.Status = StatusAllIn
        }
    }
    
    t.MinBet = bigBlindAmount
}

// 在 StartHand() 中調用
func (t *Table) StartHand() {
    // ... 現有代碼 ...
    
    t.postBlinds() // 添加這一行
    
    // 設定行動權為 BB 後一位 (UTG)
    t.CurrentPos = (t.DealerPos + 2) % 9
    t.moveToNextPlayer()
}
```

---

### 2. WebSocket 無認證機制

**位置**: `internal/game/adapter/ws/handler.go:33-38`

```go
playerID := r.URL.Query().Get("player_id")
if playerID == "" {
    http.Error(w, "player_id is required", http.StatusBadRequest)
    return
}
```

**問題描述**:
- 任何人可以使用任意 `player_id` 連接
- 沒有驗證玩家身份的真實性
- 惡意用戶可以冒充其他玩家

**安全風險**: 🔴 **嚴重安全漏洞**
- 玩家身份偽造
- 操控他人遊戲
- 作弊風險

**建議修復**:

實現 JWT Token 認證：

```go
// 1. 添加認證中間件
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 從 Header 或 Query 獲取 token
    token := r.URL.Query().Get("token")
    if token == "" {
        token = r.Header.Get("Authorization")
    }
    
    if token == "" {
        http.Error(w, "missing authentication token", http.StatusUnauthorized)
        return
    }
    
    // 驗證 JWT token
    claims, err := validateJWT(token)
    if err != nil {
        http.Error(w, "invalid token", http.StatusUnauthorized)
        return
    }
    
    playerID := claims.PlayerID
    
    // ... 繼續 WebSocket 升級 ...
}

// 2. 實現 JWT 驗證
func validateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })
    
    if err != nil || !token.Valid {
        return nil, err
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok {
        return nil, errors.New("invalid claims")
    }
    
    return claims, nil
}
```

---

### 3. CORS 完全開放

**位置**: `internal/game/adapter/ws/handler.go:14-16`

```go
CheckOrigin: func(r *http.Request) bool {
    return true // B2B 開發階段先允許所有來源
}
```

**問題描述**:
- 任何網站都可以連接到 WebSocket
- 容易受到 CSRF (跨站請求偽造) 攻擊
- 惡意網站可以在用戶不知情的情況下連接

**安全風險**: 🔴 **CSRF 攻擊風險**

**建議修復**:
```go
var allowedOrigins = map[string]bool{
    "https://yourdomain.com":     true,
    "https://www.yourdomain.com": true,
    "http://localhost:3000":      true, // 開發環境
}

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        
        // 開發環境可以放寬限制
        if isDevelopment() {
            return true
        }
        
        return allowedOrigins[origin]
    },
}
```

---

### 4. 輸入驗證不足

**位置**: `internal/game/domain/table.go:120-127`

```go
case ActionBet, ActionRaise:
    if act.Amount < t.MinBet {
        return // Invalid bet amount
    }
    diff := act.Amount - player.CurrentBet
    if player.Chips < diff {
        return // Not enough chips
    }
```

**問題描述**:
1. 沒有檢查負數金額
2. 沒有檢查整數溢出
3. 沒有檢查超大金額
4. 加注規則不正確（應該至少是前一個下注的 2 倍）

**安全風險**: 🔴 **可能導致籌碼作弊**

**建議修復**:
```go
case ActionBet, ActionRaise:
    // 1. 基本驗證
    if act.Amount <= 0 {
        h.logger.Warn("invalid bet amount: negative or zero", 
            zap.String("player_id", act.PlayerID), 
            zap.Int64("amount", act.Amount))
        return
    }
    
    // 2. 防止整數溢出
    if act.Amount > math.MaxInt64 / 2 {
        h.logger.Warn("invalid bet amount: too large", 
            zap.String("player_id", act.PlayerID), 
            zap.Int64("amount", act.Amount))
        return
    }
    
    // 3. 檢查是否超過玩家擁有的籌碼
    maxAllowed := player.Chips + player.CurrentBet
    if act.Amount > maxAllowed {
        h.logger.Warn("bet amount exceeds player chips", 
            zap.String("player_id", act.PlayerID), 
            zap.Int64("amount", act.Amount),
            zap.Int64("max_allowed", maxAllowed))
        return
    }
    
    // 4. 加注規則：至少是當前最小下注的 2 倍
    if act.Type == ActionRaise {
        minRaise := t.MinBet * 2
        if act.Amount < minRaise {
            h.logger.Warn("raise amount too small", 
                zap.String("player_id", act.PlayerID), 
                zap.Int64("amount", act.Amount),
                zap.Int64("min_raise", minRaise))
            return
        }
    }
    
    // 5. 下注規則：至少等於大盲注
    if act.Type == ActionBet && act.Amount < t.MinBet {
        return
    }
    
    // ... 繼續處理下注邏輯 ...
```

---

### 5. Table Goroutine 無優雅關閉機制

**位置**: `internal/game/domain/table.go:79-88`

```go
func (t *Table) Run() {
    for {
        select {
        case action := <-t.ActionCh:
            t.handleAction(action)
        case <-t.CloseCh:
            return
        }
    }
}
```

**問題描述**:
- Table goroutine 不接收 `context.Context`
- 在 `main.go:62` 的 `app.Stop(ctx)` 被調用時無法通知 Table
- 可能導致 goroutine 洩漏

**影響**: 🔴 **資源洩漏**

**建議修復**:

```go
// 1. 修改 Table 結構
type Table struct {
    // ... 現有字段 ...
    ctx    context.Context
    cancel context.CancelFunc
}

// 2. 修改 NewTable
func NewTable(id string) *Table {
    ctx, cancel := context.WithCancel(context.Background())
    return &Table{
        ID:       id,
        Pots:     NewPotManager(),
        Deck:     NewDeck(),
        Players:  make(map[string]*Player),
        ActionCh: make(chan PlayerAction, 100),
        State:    StateIdle,
        ctx:      ctx,
        cancel:   cancel,
    }
}

// 3. 修改 Run 方法
func (t *Table) Run() {
    for {
        select {
        case action := <-t.ActionCh:
            t.handleAction(action)
        case <-t.ctx.Done():
            t.logger.Info("table shutting down", zap.String("table_id", t.ID))
            return
        }
    }
}

// 4. 添加 Stop 方法
func (t *Table) Stop() {
    t.cancel()
}

// 5. 在 TableManager 中管理關閉
func (tm *TableManager) StopAll(ctx context.Context) {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    for id, table := range tm.tables {
        table.Stop()
        tm.logger.Info("stopped table", zap.String("table_id", id))
    }
}
```

---

## 🟡 中等問題 (Medium Priority Issues)

### 6. 重複的 Card 實現

**位置**:
- `internal/game/domain/card.go` - 使用位元編碼
- `pkg/poker/card.go` - 使用結構體

**問題描述**:
- 存在兩種不同的 Card 實現
- `pkg/poker/card.go` 沒有被任何地方使用
- 造成代碼冗余和潛在的混淆

**建議**: 刪除 `pkg/poker/card.go`，統一使用 `internal/game/domain/card.go`

---

### 7. Hub Broadcast 可能導致消息丟失

**位置**: `internal/game/adapter/ws/hub.go:46-56`

```go
case message := <-h.broadcast:
    h.mu.RLock()
    for _, client := range h.clients {
        select {
        case client.send <- message:
        default:
            // 如果發送失敗（通道滿），主動斷開或略過
            // 這裡暫時略過
        }
    }
    h.mu.RUnlock()
```

**問題描述**:
- `default` case 只是略過，沒有任何處理
- 沒有 logging
- 沒有斷開連接
- 玩家可能收不到關鍵消息但連接仍然存在

**建議修復**:
```go
case message := <-h.broadcast:
    h.mu.RLock()
    disconnectedClients := make([]string, 0)
    
    for playerID, client := range h.clients {
        select {
        case client.send <- message:
            // 成功發送
        default:
            // 發送失敗，記錄並標記為斷開
            h.logger.Warn("client send buffer full, disconnecting", 
                zap.String("player_id", playerID))
            disconnectedClients = append(disconnectedClients, playerID)
        }
    }
    h.mu.RUnlock()
    
    // 清理斷開的連接
    if len(disconnectedClients) > 0 {
        h.mu.Lock()
        for _, playerID := range disconnectedClients {
            if client, ok := h.clients[playerID]; ok {
                close(client.send)
                delete(h.clients, playerID)
            }
        }
        h.mu.Unlock()
    }
```

---

### 8. 缺少玩家行動超時機制

**位置**: `internal/game/domain/table.go:79-88`

**問題描述**:
- `config.yaml` 定義了 `timeout_seconds: 15` 但沒有實現
- 如果玩家不做動作，遊戲會永遠卡住
- 其他玩家體驗很差

**建議修復**:
```go
func (t *Table) Run() {
    actionTimeout := time.Duration(t.config.TimeoutSeconds) * time.Second
    timer := time.NewTimer(actionTimeout)
    defer timer.Stop()
    
    for {
        select {
        case action := <-t.ActionCh:
            // 重置計時器
            if !timer.Stop() {
                <-timer.C
            }
            timer.Reset(actionTimeout)
            
            t.handleAction(action)
            
        case <-timer.C:
            // 超時處理：自動 Fold 當前玩家
            currentPlayer := t.Seats[t.CurrentPos]
            if currentPlayer != nil && currentPlayer.CanAct() {
                t.logger.Warn("player action timeout, auto-folding", 
                    zap.String("player_id", currentPlayer.ID))
                
                currentPlayer.Status = StatusFolded
                currentPlayer.HasActed = true
                
                if t.isRoundComplete() {
                    t.nextStreet()
                } else {
                    t.moveToNextPlayer()
                }
            }
            
            // 重置計時器
            timer.Reset(actionTimeout)
            
        case <-t.ctx.Done():
            return
        }
    }
}
```

---

### 9. 缺少數據持久化

**問題描述**:
- 所有遊戲狀態都在內存中
- 服務器重啟後所有數據丟失
- 玩家籌碼、遊戲記錄無法保存

**影響**: 無法用於生產環境

**建議**: 實現多層存儲策略

```go
// 1. 熱數據 (Redis) - 當前遊戲狀態
type RedisRepository struct {
    client *redis.Client
}

func (r *RedisRepository) SaveTableState(table *Table) error {
    data, err := json.Marshal(table)
    if err != nil {
        return err
    }
    
    key := fmt.Sprintf("table:%s", table.ID)
    return r.client.Set(context.Background(), key, data, 30*time.Minute).Err()
}

// 2. 冷數據 (PostgreSQL) - 歷史記錄
type HandHistory struct {
    ID            string
    TableID       string
    Players       []PlayerSnapshot
    Actions       []PlayerAction
    CommunityCards []Card
    Pots          []*Pot
    Winners       map[string]int64
    Timestamp     time.Time
}

func (db *PostgresDB) SaveHandHistory(history *HandHistory) error {
    // 存儲到 PostgreSQL
}
```

---

### 10. 底池餘數分配不符合規則

**位置**: `internal/game/domain/distributor.go:56-58`

```go
// TODO: 目前餘數分配是基於 Map 迭代順序 (隨機) 或者 Slice 順序。
// 標準規則應分配給最靠近 Button 的玩家 (Position-based)。
amt++ // 把餘數分給前幾位
```

**問題描述**:
- 餘數隨機分配不符合德州撲克規則
- 應該分配給順時針最靠近 Button 的玩家

**建議修復**:
```go
// 1. 在 Pot 中記錄玩家位置
type Pot struct {
    Amount       int64
    Contributors map[string]bool
    Positions    map[string]int // 添加位置信息
}

// 2. 修改分配邏輯
func Distribute(pots []*Pot, players map[string]*Player, board []Card, buttonPos int) map[string]int64 {
    payouts := make(map[string]int64)
    
    for _, pot := range pots {
        // ... 找出贏家 ...
        
        if len(winners) == 0 {
            continue
        }
        
        // 按照位置排序贏家（順時針從 Button 開始）
        sortedWinners := sortByPosition(winners, pot.Positions, buttonPos)
        
        share := pot.Amount / int64(len(winners))
        remainder := pot.Amount % int64(len(winners))
        
        for i, pid := range sortedWinners {
            amt := share
            if i < int(remainder) {
                amt++ // 餘數分給最靠近 Button 的玩家
            }
            payouts[pid] += amt
        }
    }
    
    return payouts
}

func sortByPosition(winners []string, positions map[string]int, buttonPos int) []string {
    sort.Slice(winners, func(i, j int) bool {
        posI := (positions[winners[i]] - buttonPos + 9) % 9
        posJ := (positions[winners[j]] - buttonPos + 9) % 9
        return posI < posJ
    })
    return winners
}
```

---

## 🟢 輕微問題 (Minor Issues)

### 11. 編譯產物在 Git 倉庫中

**問題**: `game-server.exe` (10.6 MB) 被提交到倉庫

**建議**: 添加到 `.gitignore`:
```
*.exe
*.bin
*.test
game-server
```

---

### 12. Magic Numbers 應提取為配置

**位置**: `internal/game/adapter/ws/client.go:13-17`

```go
const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 512
)
```

**建議**: 移到 `config.yaml`:
```yaml
websocket:
  write_timeout_seconds: 10
  pong_timeout_seconds: 60
  max_message_size: 512
```

---

### 13. 混合語言註釋

**問題**: 部分註釋中文，部分英文

**建議**: 統一為英文（便於國際化）或全部中文

---

### 14. 缺少 Rate Limiting

**問題**: 惡意用戶可以瞬間發送大量請求

**建議**: 實現令牌桶算法:
```go
import "golang.org/x/time/rate"

type Client struct {
    // ... 現有字段 ...
    rateLimiter *rate.Limiter
}

func NewClient(...) *Client {
    return &Client{
        // ... 現有初始化 ...
        rateLimiter: rate.NewLimiter(rate.Limit(10), 20), // 每秒 10 個請求，突發 20
    }
}

func (c *Client) ReadPump() {
    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            break
        }
        
        // Rate limiting
        if !c.rateLimiter.Allow() {
            c.logger.Warn("rate limit exceeded", zap.String("player_id", c.PlayerID))
            continue
        }
        
        // ... 處理消息 ...
    }
}
```

---

### 15. 錯誤處理不一致

**問題**: 
- 有些錯誤只是 `return`
- 有些會 log
- 客戶端收不到錯誤訊息

**建議**: 統一錯誤處理:
```go
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func (t *Table) handleAction(act PlayerAction) {
    // 驗證失敗時發送錯誤給客戶端
    if err := t.validateAction(act); err != nil {
        t.sendError(act.PlayerID, err)
        return
    }
    
    // ... 處理動作 ...
}

func (t *Table) sendError(playerID string, err error) {
    errResp := ErrorResponse{
        Code:    "INVALID_ACTION",
        Message: err.Error(),
    }
    t.hub.SendToPlayer(playerID, errResp)
}
```

---

## ✅ 優點 (Strengths)

### 架構設計
1. **優秀的 DDD 實現**: Domain 層邏輯清晰，完全獨立於基礎設施
2. **六邊形架構**: Ports & Adapters 分離得很好
3. **依賴注入**: 使用 Google Wire 實現編譯期 DI
4. **有限狀態機**: 遊戲狀態管理清晰

### 代碼質量
1. **測試覆蓋率高**: Domain 層有 6 個測試文件
2. **無 Race Condition**: 通過了 `-race` 檢測
3. **Side Pot 邏輯正確**: 複雜的邊池算法實現準確
4. **牌力評估完整**: 實現了完整的 7 張牌評估算法

### 技術選型
1. **高性能**: 使用位元編碼優化性能
2. **併發設計**: 每個 Table 獨立 Goroutine
3. **結構化日誌**: 使用 Zap 實現高性能日誌
4. **WebSocket**: 實時通訊選型正確

---

## 📊 代碼統計

| 指標 | 數值 |
|------|------|
| 總文件數 | 30 個 Go 文件 |
| 代碼行數 | ~2,186 行 |
| 測試文件 | 6 個 |
| 測試覆蓋率 | Domain 層 > 80% (估計) |
| Go 版本 | 1.25.5 |
| 依賴數量 | 4 個主要依賴 |

---

## 📋 修復優先級建議

### 🔴 立即修復 (Critical - 1 週內)

1. ✅ **實現盲注邏輯** (2-3 小時)
2. ✅ **添加輸入驗證** (2-3 小時)
3. ✅ **實現玩家超時機制** (3-4 小時)
4. ✅ **修復 Goroutine 洩漏** (2 小時)

**總工時**: ~12 小時

---

### 🟡 短期修復 (High Priority - 2-4 週內)

1. ✅ **實現認證機制** (1-2 天)
2. ✅ **限制 CORS** (1 小時)
3. ✅ **修復 Hub Broadcast** (2 小時)
4. ✅ **修復餘數分配** (3 小時)
5. ✅ **實現 Rate Limiting** (4 小時)

**總工時**: ~3 天

---

### 🟢 長期優化 (Medium Priority - 1-2 個月)

1. ✅ **數據持久化** (1-2 週)
   - Redis 集成
   - PostgreSQL 集成
   - 手牌歷史記錄

2. ✅ **監控和可觀測性** (1 週)
   - Prometheus metrics
   - 分布式追蹤
   - 告警系統

3. ✅ **統一錯誤處理** (3 天)
4. ✅ **代碼重構** (1 週)
   - 移除重複代碼
   - 統一註釋語言
   - 提取 Magic Numbers

**總工時**: ~5-6 週

---

## 🚀 生產就緒檢查清單

### 功能完整性
- [ ] 盲注邏輯
- [ ] 行動超時
- [ ] 自動下一局
- [ ] 玩家重連
- [ ] 斷線處理
- [ ] 觀察者模式

### 安全性
- [ ] 認證/授權
- [ ] CORS 限制
- [ ] Rate Limiting
- [ ] 輸入驗證
- [ ] SQL 注入防護
- [ ] XSS 防護

### 穩定性
- [ ] 優雅關閉
- [ ] 錯誤恢復
- [ ] 健康檢查
- [ ] 熔斷機制
- [ ] 重試機制

### 可觀測性
- [ ] 結構化日誌
- [ ] Metrics 導出
- [ ] 分布式追蹤
- [ ] 告警規則
- [ ] 性能監控

### 數據管理
- [ ] 數據持久化
- [ ] 備份策略
- [ ] 災難恢復
- [ ] 數據遷移

### 運維
- [ ] CI/CD 管道
- [ ] 自動化測試
- [ ] 容器化 (Docker)
- [ ] 編排 (Kubernetes)
- [ ] 滾動更新

---

## 📚 參考資源

### 德州撲克規則
- [Official Poker Rules](https://www.pokernews.com/poker-rules/)
- [Side Pot Calculation](https://en.wikipedia.org/wiki/Betting_in_poker#Side_pots)

### Go 最佳實踐
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 架構模式
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)

---

## 🎯 總結

TheNuts 項目展現了**優秀的架構設計**和**清晰的代碼結構**，核心撲克邏輯實現正確且有良好的測試覆蓋。然而，在生產就緒性方面還有一些關鍵問題需要解決：

**必須修復的問題**:
1. 盲注邏輯（遊戲無法正常進行）
2. 認證機制（安全風險）
3. 輸入驗證（防止作弊）
4. 超時機制（用戶體驗）

**建議優先級**:
- **第一階段** (1 週): 修復所有 Critical 問題，使遊戲可玩
- **第二階段** (2-4 週): 實現安全性和穩定性改進
- **第三階段** (1-2 月): 添加持久化、監控等生產功能

完成以上修復後，該項目將具備上線的基本條件。

---

**審查人**: Claude (AI Code Reviewer)  
**審查版本**: Latest commit as of 2026-01-22  
**下次審查建議**: 2 週後（完成 Critical 修復後）
