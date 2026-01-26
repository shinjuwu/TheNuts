# 持久化層實作檢查清單

## 📋 總覽

這個檢查清單將引導你完成持久化層的完整實作，預計需要 **12-16 小時**。

---

## 階段 1: 環境準備 (1-2 小時)

### 1.1 數據庫環境 ⏳

- [ ] 安裝 Docker Desktop
- [ ] 創建 `docker-compose.yml`
- [ ] 啟動 PostgreSQL 容器
  ```bash
  docker-compose up -d postgres
  ```
- [ ] 啟動 Redis 容器
  ```bash
  docker-compose up -d redis
  ```
- [ ] 驗證 PostgreSQL 連接
  ```bash
  psql -h localhost -U thenuts -d thenuts -c "SELECT version();"
  ```
- [ ] 驗證 Redis 連接
  ```bash
  redis-cli ping
  ```
- [ ] (可選) 啟動 pgAdmin
  ```bash
  docker-compose up -d pgadmin
  ```

**驗收標準**:
- ✅ PostgreSQL 狀態為 `healthy`
- ✅ Redis 狀態為 `healthy`
- ✅ 可以通過 psql 連接數據庫

---

### 1.2 遷移工具 ⏳

- [ ] 安裝 golang-migrate
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```
- [ ] 驗證安裝
  ```bash
  migrate -version
  ```
- [ ] 執行初始遷移
  ```bash
  migrate -path migrations -database "$DATABASE_URL" up
  ```
- [ ] 確認版本
  ```bash
  migrate -path migrations -database "$DATABASE_URL" version
  ```

**驗收標準**:
- ✅ migrate 命令可用
- ✅ 遷移執行成功
- ✅ 8 張表已創建

---

### 1.3 Go 依賴 ⏳

- [ ] 更新 `go.mod`
  ```bash
  go get github.com/jackc/pgx/v5@latest
  go get github.com/jackc/pgx/v5/pgxpool@latest
  go get github.com/redis/go-redis/v9@latest
  go get github.com/golang-migrate/migrate/v4@latest
  go get golang.org/x/crypto/bcrypt@latest
  ```
- [ ] 整理依賴
  ```bash
  go mod tidy
  ```
- [ ] 驗證編譯
  ```bash
  go build ./...
  ```

**驗收標準**:
- ✅ go.mod 包含所有依賴
- ✅ 無編譯錯誤

---

## 階段 2: 基礎設施層 (2-3 小時)

### 2.1 配置管理 ⏳

- [ ] 更新 `internal/infra/config/config.go`
  ```go
  type DatabaseConfig struct {
      Postgres PostgresConfig
      Redis    RedisConfig
  }
  
  type PostgresConfig struct {
      Host            string
      Port            int
      User            string
      Password        string
      Database        string
      MaxConns        int32
      MinConns        int32
      MaxConnLifetime time.Duration
  }
  
  type RedisConfig struct {
      Host     string
      Port     int
      Password string
      DB       int
      PoolSize int
  }
  ```
- [ ] 更新 `config.yaml`
- [ ] 添加環境變數支援
- [ ] 編寫配置載入測試

**驗收標準**:
- ✅ 配置可以從 YAML 讀取
- ✅ 可以通過環境變數覆蓋
- ✅ 配置驗證邏輯正確

**文件**: `internal/infra/config/config.go`

---

### 2.2 數據庫連接池 ⏳

- [ ] 創建 `internal/infra/database/postgres.go`
  ```go
  func NewPostgresPool(cfg PostgresConfig) (*pgxpool.Pool, error)
  ```
- [ ] 實作連接池配置
- [ ] 添加健康檢查
  ```go
  func (db *DB) HealthCheck(ctx context.Context) error
  ```
- [ ] 添加優雅關閉
  ```go
  func (db *DB) Close()
  ```
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 連接池可以建立
- ✅ HealthCheck 通過
- ✅ 測試覆蓋率 > 80%

**文件**: `internal/infra/database/postgres.go`

---

### 2.3 Redis 客戶端 ⏳

- [ ] 創建 `internal/infra/database/redis.go`
  ```go
  func NewRedisClient(cfg RedisConfig) (*redis.Client, error)
  ```
- [ ] 實作連接配置
- [ ] 添加健康檢查
- [ ] 編寫單元測試

**驗收標準**:
- ✅ Redis 客戶端可用
- ✅ Ping 命令成功
- ✅ 測試通過

**文件**: `internal/infra/database/redis.go`

---

## 階段 3: Repository 實作 (6-8 小時)

### 3.1 Repository 介面定義 ⏳

- [ ] 創建 `internal/infra/repository/interfaces.go`
- [ ] 定義 `AccountRepository` 介面
- [ ] 定義 `PlayerRepository` 介面
- [ ] 定義 `WalletRepository` 介面 ⭐ 重點
- [ ] 定義 `TransactionRepository` 介面
- [ ] 定義 `GameSessionRepository` 介面
- [ ] 定義 `HandHistoryRepository` 介面
- [ ] 定義 `UnitOfWork` 介面

**驗收標準**:
- ✅ 所有介面方法簽名正確
- ✅ 使用 context.Context
- ✅ 使用 uuid.UUID

**文件**: `internal/infra/repository/interfaces.go`

---

### 3.2 領域模型 ⏳

- [ ] 創建 `internal/infra/repository/models.go`
- [ ] 定義 `Account` struct
- [ ] 定義 `Player` struct
- [ ] 定義 `Wallet` struct
- [ ] 定義 `Transaction` struct
- [ ] 定義 `GameSession` struct
- [ ] 定義 `HandHistory` struct

**驗收標準**:
- ✅ 結構與數據庫 Schema 一致
- ✅ 使用正確的 Go 類型
- ✅ 添加 JSON tags (如需要)

**文件**: `internal/infra/repository/models.go`

---

### 3.3 Account Repository ⏳

- [ ] 創建 `internal/infra/repository/postgres/account_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByID()`
- [ ] 實作 `GetByUsername()`
- [ ] 實作 `GetByEmail()`
- [ ] 實作 `Update()`
- [ ] 實作 `UpdateLastLogin()`
- [ ] 實作 `IncrementFailedAttempts()`
- [ ] 實作 `LockAccount()`
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 所有方法正確實作
- ✅ 使用參數化查詢 (防 SQL 注入)
- ✅ 測試覆蓋率 > 80%

**文件**: `internal/infra/repository/postgres/account_repo.go`

---

### 3.4 Player Repository ⏳

- [ ] 創建 `internal/infra/repository/postgres/player_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByID()`
- [ ] 實作 `GetByAccountID()`
- [ ] 實作 `Update()`
- [ ] 實作 `UpdateStats()`
- [ ] 實作 `GetTopPlayersByWinnings()`
- [ ] 編寫單元測試

**驗收標準**:
- ✅ CRUD 操作正確
- ✅ 統計更新邏輯正確
- ✅ 測試通過

**文件**: `internal/infra/repository/postgres/player_repo.go`

---

### 3.5 Wallet Repository ⭐ 最重要

- [ ] 創建 `internal/infra/repository/postgres/wallet_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByPlayerID()`
- [ ] 實作 `GetWithLock()` - 使用 `SELECT FOR UPDATE`
- [ ] 實作 `Credit()` - 入帳
  - [ ] 檢查冪等性鍵
  - [ ] 鎖定錢包
  - [ ] 更新餘額
  - [ ] 創建交易記錄
  - [ ] 更新版本號
- [ ] 實作 `Debit()` - 出帳
  - [ ] 檢查冪等性鍵
  - [ ] 鎖定錢包
  - [ ] 檢查餘額充足
  - [ ] 更新餘額
  - [ ] 創建交易記錄
  - [ ] 更新版本號
- [ ] 實作 `LockBalance()` - 鎖定餘額
- [ ] 實作 `UnlockBalance()` - 解鎖餘額
- [ ] 編寫完整的單元測試
  - [ ] 測試併發場景
  - [ ] 測試餘額不足
  - [ ] 測試冪等性
  - [ ] 測試樂觀鎖

**驗收標準**:
- ✅ 交易原子性保證
- ✅ 防止重複扣款
- ✅ 併發安全
- ✅ 餘額始終非負
- ✅ 測試覆蓋率 > 90%

**文件**: `internal/infra/repository/postgres/wallet_repo.go`

---

### 3.6 Transaction Repository ⏳

- [ ] 創建 `internal/infra/repository/postgres/transaction_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByID()`
- [ ] 實作 `GetByWalletID()`
- [ ] 實作 `GetByIdempotencyKey()`
- [ ] 實作分頁查詢
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 交易記錄不可變
- ✅ 冪等性鍵唯一性
- ✅ 測試通過

**文件**: `internal/infra/repository/postgres/transaction_repo.go`

---

### 3.7 GameSession Repository ⏳

- [ ] 創建 `internal/infra/repository/postgres/game_session_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByID()`
- [ ] 實作 `GetActiveByPlayerID()`
- [ ] 實作 `Update()`
- [ ] 實作 `End()`
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 會話狀態正確更新
- ✅ 淨盈虧計算正確
- ✅ 測試通過

**文件**: `internal/infra/repository/postgres/game_session_repo.go`

---

### 3.8 HandHistory Repository ⏳

- [ ] 創建 `internal/infra/repository/postgres/hand_history_repo.go`
- [ ] 實作 `Create()`
- [ ] 實作 `GetByID()`
- [ ] 實作 `GetByGameSessionID()`
- [ ] 實作 JSONB 查詢
- [ ] 編寫單元測試

**驗收標準**:
- ✅ JSONB 數據正確序列化
- ✅ 查詢性能良好
- ✅ 測試通過

**文件**: `internal/infra/repository/postgres/hand_history_repo.go`

---

### 3.9 UnitOfWork (事務管理) ⏳

- [ ] 創建 `internal/infra/repository/postgres/unit_of_work.go`
- [ ] 實作 `Begin()`
- [ ] 實作 `Commit()`
- [ ] 實作 `Rollback()`
- [ ] 實作事務傳播
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 事務正確開始/提交/回滾
- ✅ 嵌套事務處理正確
- ✅ 測試通過

**文件**: `internal/infra/repository/postgres/unit_of_work.go`

---

## 階段 4: Redis 整合 (2-3 小時)

### 4.1 Ticket Store (Redis) ⏳

- [ ] 創建 `internal/infra/repository/redis/ticket_store.go`
- [ ] 實作 `Generate()`
- [ ] 實作 `Validate()`
- [ ] 實作自動過期 (使用 Redis TTL)
- [ ] 替換 `MemoryTicketStore`
- [ ] 編寫單元測試

**驗收標準**:
- ✅ 票券存儲在 Redis
- ✅ 自動過期機制正常
- ✅ 驗證後刪除
- ✅ 測試通過

**文件**: `internal/infra/repository/redis/ticket_store.go`

---

### 4.2 Session Store (Redis) ⏳

- [ ] 創建 `internal/infra/repository/redis/session_store.go`
- [ ] 實作 `Create()`
- [ ] 實作 `Get()`
- [ ] 實作 `Update()`
- [ ] 實作 `Delete()`
- [ ] 編寫單元測試

**驗收標準**:
- ✅ Session 緩存正常
- ✅ 過期時間正確
- ✅ 測試通過

**文件**: `internal/infra/repository/redis/session_store.go`

---

## 階段 5: 整合與測試 (2-3 小時)

### 5.1 依賴注入整合 ⏳

- [ ] 更新 `pkg/di/provider.go`
- [ ] 添加 `ProvidePostgresPool()`
- [ ] 添加 `ProvideRedisClient()`
- [ ] 添加所有 Repository Providers
- [ ] 更新 `wire.go`
- [ ] 生成 DI 代碼
  ```bash
  wire gen ./pkg/di
  ```

**驗收標準**:
- ✅ DI 圖正確生成
- ✅ 無循環依賴
- ✅ 編譯通過

**文件**: `pkg/di/provider.go`, `pkg/di/wire.go`

---

### 5.2 整合測試 ⏳

- [ ] 創建 `internal/infra/repository/integration_test.go`
- [ ] 測試完整的買入/遊戲/兌現流程
- [ ] 測試併發場景
- [ ] 測試錯誤處理
- [ ] 測試事務回滾

**驗收標準**:
- ✅ 端到端流程測試通過
- ✅ 併發測試無死鎖
- ✅ 數據一致性保證

**文件**: `internal/infra/repository/integration_test.go`

---

### 5.3 性能測試 ⏳

- [ ] 編寫基準測試
  ```bash
  go test -bench=. -benchmem ./internal/infra/repository/...
  ```
- [ ] 測試連接池性能
- [ ] 測試查詢性能
- [ ] 優化慢查詢

**驗收標準**:
- ✅ 基準測試完成
- ✅ 性能指標記錄
- ✅ 無明顯瓶頸

---

## 階段 6: 文檔與部署 (1-2 小時)

### 6.1 API 文檔 ⏳

- [ ] 為所有 Repository 添加 Godoc 註釋
- [ ] 生成 API 文檔
  ```bash
  godoc -http=:6060
  ```
- [ ] 添加使用範例

**驗收標準**:
- ✅ 所有公開方法有文檔
- ✅ 使用範例完整
- ✅ Godoc 可訪問

---

### 6.2 部署準備 ⏳

- [ ] 準備生產環境配置
- [ ] 創建備份腳本
- [ ] 創建監控腳本
- [ ] 準備遷移計劃

**驗收標準**:
- ✅ 生產配置就緒
- ✅ 備份恢復流程測試
- ✅ 監控指標定義

---

## 🎯 最終驗收

### 功能檢查 ✅

- [ ] 帳號可以註冊/登入
- [ ] 玩家資料可以創建/查詢
- [ ] 錢包可以入帳/出帳
- [ ] 交易記錄完整
- [ ] 遊戲會話可以創建/結束
- [ ] 手牌歷史可以保存
- [ ] Ticket 緩存在 Redis
- [ ] Session 管理正常

### 性能檢查 ✅

- [ ] 連接池配置合理
- [ ] 查詢性能良好 (< 100ms)
- [ ] 併發測試通過 (1000 req/s)
- [ ] 無死鎖問題

### 安全檢查 ✅

- [ ] 無 SQL 注入風險
- [ ] 密碼使用 bcrypt
- [ ] 防止重複扣款
- [ ] 事務原子性保證

### 測試覆蓋 ✅

- [ ] 單元測試覆蓋率 > 80%
- [ ] 整合測試通過
- [ ] 性能測試完成
- [ ] 所有測試通過
  ```bash
  go test ./internal/infra/... -v -cover
  ```

---

## 📊 預估時間表

| 階段 | 任務 | 預估時間 | 狀態 |
|------|------|----------|------|
| 1 | 環境準備 | 1-2 小時 | ⏳ |
| 2 | 基礎設施層 | 2-3 小時 | ⏳ |
| 3 | Repository 實作 | 6-8 小時 | ⏳ |
| 4 | Redis 整合 | 2-3 小時 | ⏳ |
| 5 | 整合與測試 | 2-3 小時 | ⏳ |
| 6 | 文檔與部署 | 1-2 小時 | ⏳ |
| **總計** | - | **14-21 小時** | ⏳ |

---

## 🚀 開始實作

準備好了嗎？從 **階段 1: 環境準備** 開始吧！

參考文檔:
- 設計文檔: `docs/PERSISTENCE_LAYER_DESIGN.md`
- 快速開始: `docs/PERSISTENCE_QUICKSTART.md`

---

**更新日期**: 2026-01-26  
**狀態**: 準備開始實作
