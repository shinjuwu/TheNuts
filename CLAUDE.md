# CLAUDE.md

此文件為 Claude Code (claude.ai/code) 在此代碼庫中工作時提供指引。

## 專案概述

**TheNuts** 是一個使用 Go 語言構建的可擴展多遊戲博弈框架，目前實現了德州撲克，計劃支援更多遊戲（老虎機、百家樂等）。專案採用整潔架構（Clean Architecture），將領域邏輯、應用服務、基礎設施和適配器分層設計。

**語言：** Go 1.25.5
**主要資料庫：** PostgreSQL 15
**快取/會話存儲：** Redis 7
**即時通訊：** WebSocket (gorilla/websocket)

## 常用命令

### 開發

```bash
# 構建伺服器
go build -o game-server.exe cmd/game-server/main.go

# 運行伺服器
./game-server.exe

# 或直接運行
go run cmd/game-server/main.go
```

### 測試

```bash
# 運行所有測試
go test ./...

# 運行特定套件的測試
go test ./internal/game/domain/...
go test ./internal/auth/...

# 運行測試並顯示詳細輸出
go test -v ./internal/game/domain/...

# 運行特定測試
go test -v -run TestFullGameFlow ./internal/game/domain/...
```

### 資料庫

```bash
# 通過 Docker 啟動 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 停止服務
docker-compose down

# 查看日誌
docker-compose logs -f postgres
docker-compose logs -f redis

# 訪問 PostgreSQL
psql -h localhost -p 5432 -U thenuts -d thenuts

# 訪問 Redis
redis-cli -p 6382
```

### 資料庫遷移

```bash
# 執行遷移（手動 - 使用 psql）
psql -h localhost -p 5432 -U thenuts -d thenuts -f migrations/000001_init_schema.up.sql
psql -h localhost -p 5432 -U thenuts -d thenuts -f migrations/000002_add_idempotency_constraint.up.sql

# 回滾遷移
psql -h localhost -p 5432 -U thenuts -d thenuts -f migrations/000002_add_idempotency_constraint.down.sql
psql -h localhost -p 5432 -U thenuts -d thenuts -f migrations/000001_init_schema.down.sql
```

### 測試腳本

```bash
# 測試認證流程（PowerShell）
.\test_auth.ps1

# 測試 WebSocket 流程（PowerShell）
.\test_websocket.ps1

# 創建測試用戶
psql -h localhost -p 5432 -U thenuts -d thenuts -f scripts/seed_test_users.sql

# 驗證環境設置
.\scripts\verify-environment.bat   # Windows
./scripts/verify-environment.sh    # Unix
```

### 依賴注入（Wire）

```bash
# 重新生成 Wire DI 代碼（修改 pkg/di/wire.go 後）
cd pkg/di
wire
```

## 架構設計

### 分層結構

```
┌─────────────────────────────────────────────────┐
│  cmd/game-server/                               │  入口點
│  - main.go (HTTP/WS 伺服器設置)                 │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  internal/game/adapter/ws/                      │  適配器層
│  - WebSocket 處理器與訊息路由                    │  (協議適配器)
│  - SessionManager (生命週期、超時)               │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  internal/game/service/                         │  應用層
│  - GameService (買入、兌現、會話)                │  (業務編排)
│  - 資料庫事務管理                                │
└─────────────────┬───────────────────────────────┘
                  │
        ┌─────────┼─────────┐
        │         │         │
┌───────▼──┐ ┌────▼────┐ ┌─▼────────┐
│  撲克    │ │  老虎機 │ │  百家樂  │              遊戲引擎
│  引擎    │ │ (待辦)  │ │ (待辦)   │              (領域層)
└──────────┘ └─────────┘ └──────────┘
        │
┌───────▼────────────────────────────────────────┐
│  internal/game/domain/                         │  領域層
│  - Table, Player, Card, Deck                   │  (核心遊戲邏輯)
│  - Evaluator (牌力評估)                         │
│  - Distributor (發牌)                           │
│  - PotManager (下注/底池管理)                    │
└─────────────────┬──────────────────────────────┘
                  │
┌─────────────────▼──────────────────────────────┐
│  internal/infra/                               │  基礎設施層
│  - persistence/ (PostgreSQL repositories)      │
│  - redis/ (Redis 客戶端 - 計劃中)               │
└────────────────────────────────────────────────┘
```

### 核心架構模式

1. **整潔架構（Clean Architecture）**：依賴指向內部（infra → domain，絕不 domain → infra）
2. **工廠模式（Factory Pattern）**：`GameEngineFactory` 用於可插拔的遊戲實現
3. **倉庫模式（Repository Pattern）**：資料訪問抽象（`AccountRepository`、`WalletRepository` 等）
4. **工作單元（Unit of Work）**：通過 `UnitOfWork` 介面管理事務
5. **依賴注入（Dependency Injection）**：使用 Google Wire 進行編譯時 DI（見 `pkg/di/wire.go`）

### 認證流程

TheNuts 使用**票券機制（ticket-based authentication）**來保護 WebSocket 連線：

1. **登入**：`POST /api/auth/login` → 返回 JWT token（24小時有效）
2. **獲取票券**：`POST /api/auth/ticket`（攜帶 JWT）→ 返回一次性票券（30秒有效）
3. **連接 WebSocket**：`GET /ws?ticket=xxx` → 驗證票券並銷毀

這樣可以防止 JWT token 出現在 URL/日誌中，同時保持安全性。詳見 `docs/AUTHENTICATION.md`。

### WebSocket 訊息類型

伺服器處理 8 種訊息類型（見 `internal/game/adapter/ws/message_handler.go`）：

- `BUY_IN` - 從錢包買入籌碼
- `CASH_OUT` - 將籌碼兌現到錢包
- `JOIN_TABLE` - 加入遊戲桌
- `LEAVE_TABLE` - 離開遊戲桌
- `SIT_DOWN` - 在桌子上坐下
- `STAND_UP` - 從座位上站起
- `GAME_ACTION` - 撲克動作（FOLD/CHECK/CALL/BET/RAISE/ALL_IN）
- `GET_BALANCE` - 查詢當前餘額

### 資料庫 Schema

主要資料表（見 `migrations/000001_init_schema.up.sql`）：

- `accounts` - 用戶認證（username, email, password_hash）
- `players` - 玩家資料（關聯到 accounts）
- `wallets` - 玩家餘額（帶餘額約束檢查）
- `transactions` - 財務交易歷史（通過 `idempotency_key` 實現冪等性）
- `game_sessions` - 活躍遊戲會話（追蹤進行中的籌碼）

### 關鍵競態條件修復

專案之前在 `ProvideWSHandler` 中存在競態條件導致堆疊溢位，現已修復（見 `docs/CRITICAL_FIX_RACE_CONDITION_2026-01-26.md`）。關鍵要點：

- **修復前**：`ProvideWSHandler` → `ProvideSessionManager` → `ProvideMessageHandler` → `ProvideWSHandler`（無限循環）
- **修復後**：分離職責 - `SessionManager` 管理生命週期，`MessageHandler` 路由訊息，無循環依賴

## 當前 MVP 狀態

**進度：約 75% 完成**

### 已完成
- 認證系統（JWT + 票券機制）
- WebSocket 基礎設施（連線、會話管理）
- 資料庫持久化層（repositories、事務）
- 遊戲服務層（買入、兌現、餘額管理）
- 領域模型（Card、Deck、Evaluator、PotManager、Player、Table）

### 關鍵差距（MVP 阻塞項）
1. **Showdown 邏輯**：需要實現贏家判定和籌碼分配（特別是邊池）
2. **自動遊戲流程**：當有 2+ 玩家準備好時，牌桌不會自動開始手牌
3. **資料庫同步**：每手牌結束後遊戲狀態（籌碼）未同步到資料庫
4. **前端**：尚未存在 Web 客戶端（僅有測試腳本）

詳見 `docs/MVP_GAP_ANALYSIS.md` 進行詳細差距分析和時間預估。

## 開發規範

### 代碼組織

- **cmd/**：應用程式入口點
- **internal/**：私有應用程式代碼
  - `auth/`：認證服務
  - `game/`：遊戲邏輯（領域、引擎、適配器）
  - `infra/`：基礎設施（資料庫、Redis）
- **pkg/**：公共函式庫（例如 `di/` 用於 Wire）
- **migrations/**：SQL 遷移文件（順序編號）
- **scripts/**：工具腳本（資料庫填充、環境檢查）
- **docs/**：架構和實現文檔

### 錯誤處理

- 將錯誤向上返回；在適配器層處理
- 使用結構化日誌（Zap）並附帶上下文：`logger.Error("msg", zap.Error(err), zap.String("player_id", id))`
- 資料庫錯誤應通過 `UnitOfWork` 觸發事務回滾

### 事務安全

在進行財務操作（買入、兌現）時：

1. 始終使用 `UnitOfWork.Begin()` 開始事務
2. 在扣款**之前**驗證餘額
3. 在 transactions 表中使用 `idempotency_key` 防止重複操作
4. 僅在所有操作成功後調用 `UnitOfWork.Commit()`
5. Defer `UnitOfWork.Rollback()` 作為安全網

範例：
```go
uow := h.unitOfWork.Begin(ctx)
defer uow.Rollback()

// 驗證餘額
wallet, err := uow.Wallets().Get(ctx, playerID)
if wallet.Balance < amount {
    return errors.New("insufficient balance")
}

// 扣款並記錄
err = uow.Wallets().UpdateBalance(ctx, playerID, -amount)
err = uow.Transactions().Create(ctx, tx)
err = uow.GameSessions().Create(ctx, session)

return uow.Commit()
```

### 測試策略

- **單元測試**：隔離測試領域邏輯（`internal/game/domain/*_test.go`）
- **集成測試**：使用模擬依賴測試完整遊戲流程
- **手動測試**：使用 `test_auth.ps1` 和 `test_websocket.ps1` 進行端到端測試
- **負載測試**：尚未實現（計劃在 MVP 後）

### 配置

配置在 `config.yaml` 中，通過 `internal/infra/config/config.go` 載入。關鍵設置：

- `server.port`：HTTP 伺服器端口（預設 8080）
- `auth.jwt_secret`：JWT 簽名密鑰（**生產環境必須更改**）
- `auth.ticket_ttl_seconds`：票券有效期（預設 30秒）
- `database.postgres.*`：PostgreSQL 連線設置
- `database.redis.*`：Redis 連線設置（注意：端口 6382 避免衝突）

### WebSocket 連線生命週期

1. 客戶端通過 HTTP 認證並獲取 JWT
2. 客戶端通過 `/api/auth/ticket` 請求票券
3. 客戶端連接到 `/ws?ticket=xxx`
4. 伺服器驗證票券，創建 `PlayerSession`
5. `SessionManager` 在 `Hub` 中註冊客戶端
6. 客戶端通過 `MessageHandler` 發送/接收訊息
7. 斷線時，`SessionManager.Cleanup()` 在 30 分鐘超時後自動兌現

### 遊戲引擎介面

所有遊戲引擎必須實現 `GameEngine` 介面（見 `internal/game/core/game_engine.go`）：

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

這允許可插拔的遊戲實現（目前僅實現了撲克）。

## 常見陷阱

1. **別忘記重新生成 Wire 代碼** - 修改 `pkg/di/wire.go` 中的 DI providers 後
2. **不要直接在 URL 中放置 JWT tokens** - 始終使用票券機制
3. **不要繞過 UnitOfWork** - 進行資料庫更改時，事務對一致性至關重要
4. **不要在遊戲邏輯中使用 `math/rand`** - 洗牌時使用 `crypto/rand`（見 `Deck.Shuffle()`）
5. **SessionManager 超時為 30 分鐘** - 玩家如果不活躍會被自動兌現

## 下一步（MVP 後）

完成 MVP 阻塞項後：
1. Redis 整合用於票券存儲和會話快取
2. 手牌歷史記錄和回放功能
3. 錦標賽模式（MTT - 多桌錦標賽）
4. 遊戲監控的管理後台
5. 限流和 DDoS 防護
6. Prometheus 指標和告警

## 關鍵文檔

- `docs/ARCHITECTURE.md` - 詳細架構設計（多遊戲框架願景）
- `docs/AUTHENTICATION.md` - 完整認證流程文檔
- `docs/MVP_GAP_ANALYSIS.md` - 詳細差距分析及時間預估
- `docs/QUICK_START.md` - 創建遊戲桌和處理動作的教程
- `docs/CRITICAL_FIX_RACE_CONDITION_2026-01-26.md` - 關鍵 bug 修復文檔
