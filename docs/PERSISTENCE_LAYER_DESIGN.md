# 持久化層設計文檔

## 📋 目錄
- [1. 概述](#1-概述)
- [2. 技術選型](#2-技術選型)
- [3. 數據庫 Schema 設計](#3-數據庫-schema-設計)
- [4. Repository 模式](#4-repository-模式)
- [5. 遷移策略](#5-遷移策略)
- [6. 實作步驟](#6-實作步驟)

---

## 1. 概述

### 1.1 設計目標

- ✅ **可擴展性** - 支援水平擴展
- ✅ **數據一致性** - ACID 保證
- ✅ **高可用性** - 主從複製 + 讀寫分離
- ✅ **審計追蹤** - 所有關鍵操作可追溯
- ✅ **性能優化** - 適當的索引和分區

### 1.2 核心需求

| 需求 | 說明 | 優先級 |
|------|------|--------|
| 玩家資料持久化 | 基本資料、認證資訊 | 🔴 P0 |
| 錢包系統 | 餘額、交易記錄 | 🔴 P0 |
| 遊戲記錄 | 手牌歷史、結果 | 🟡 P1 |
| 審計日誌 | 所有資金變動 | 🟡 P1 |
| Session 管理 | 斷線重連 | 🟢 P2 |

---

## 2. 技術選型

### 2.1 主數據庫：PostgreSQL 15+

**選擇理由**:
- ✅ ACID 完整支援
- ✅ JSONB 支援（靈活存儲複雜數據）
- ✅ 強大的索引能力（B-tree, GIN, GIST）
- ✅ 成熟的 HA 方案（Patroni, Pgpool-II）
- ✅ 優秀的社區支援

### 2.2 緩存層：Redis 7+

**用途**:
- Session 緩存（斷線重連）
- 票券緩存（替代 MemoryTicketStore）
- 排行榜（Sorted Set）
- 分布式鎖（RedLock）

### 2.3 數據庫驅動

```go
// 使用成熟的 Go 驅動
require (
    github.com/jackc/pgx/v5 v5.5.0           // PostgreSQL 驅動
    github.com/jackc/pgx/v5/pgxpool v5.5.0   // 連接池
    github.com/redis/go-redis/v9 v9.3.0      // Redis 客戶端
    github.com/golang-migrate/migrate/v4     // 數據庫遷移
)
```

### 2.4 ORM vs Raw SQL

**決策**: 混合使用

- **簡單 CRUD** → `pgx` (輕量級，性能好)
- **複雜查詢** → Raw SQL (完全控制)
- **不使用** → GORM (過於重量級)

---

## 3. 數據庫 Schema 設計

### 3.1 核心表結構

#### 📊 ER 圖概覽

```
┌─────────────┐       ┌──────────────┐       ┌─────────────┐
│   accounts  │──────>│   players    │──────>│   wallets   │
│  (認證資訊)  │       │  (玩家資料)   │       │  (餘額)      │
└─────────────┘       └──────────────┘       └─────────────┘
                             │                       │
                             │                       │
                             ▼                       ▼
                      ┌──────────────┐       ┌─────────────┐
                      │  game_sessions│       │transactions │
                      │  (遊戲會話)    │       │ (交易記錄)   │
                      └──────────────┘       └─────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │  hand_history│
                      │  (手牌歷史)    │
                      └──────────────┘
```

---

### 3.2 詳細 Schema 設計

#### 表 1: `accounts` - 帳號認證

```sql
CREATE TABLE accounts (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 認證資訊
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- bcrypt hash
    
    -- 狀態
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, suspended, banned
    email_verified BOOLEAN NOT NULL DEFAULT false,
    
    -- 安全
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 索引
    CONSTRAINT chk_username_length CHECK (char_length(username) >= 3),
    CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$')
);

-- 索引
CREATE INDEX idx_accounts_email ON accounts(email);
CREATE INDEX idx_accounts_username ON accounts(username);
CREATE INDEX idx_accounts_status ON accounts(status) WHERE status != 'active';

-- 更新時間觸發器
CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

#### 表 2: `players` - 玩家資料

```sql
CREATE TABLE players (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    
    -- 玩家資訊
    display_name VARCHAR(50) NOT NULL,
    avatar_url VARCHAR(500),
    level INT NOT NULL DEFAULT 1,
    experience BIGINT NOT NULL DEFAULT 0,
    
    -- 統計資料 (快取，可從 game_sessions 聚合)
    total_games_played INT NOT NULL DEFAULT 0,
    total_hands_played INT NOT NULL DEFAULT 0,
    total_winnings BIGINT NOT NULL DEFAULT 0,
    
    -- VIP 狀態
    vip_level INT NOT NULL DEFAULT 0,
    vip_expires_at TIMESTAMPTZ,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_display_name_length CHECK (char_length(display_name) >= 2)
);

-- 索引
CREATE INDEX idx_players_account_id ON players(account_id);
CREATE INDEX idx_players_display_name ON players(display_name);
CREATE INDEX idx_players_level ON players(level DESC);

-- 更新時間觸發器
CREATE TRIGGER update_players_updated_at
    BEFORE UPDATE ON players
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

#### 表 3: `wallets` - 錢包餘額

```sql
CREATE TABLE wallets (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
    
    -- 餘額 (使用 BIGINT 存儲，單位：分/角子)
    balance BIGINT NOT NULL DEFAULT 0,
    locked_balance BIGINT NOT NULL DEFAULT 0, -- 鎖定中的金額 (進行中遊戲)
    
    -- 貨幣類型 (多幣種支援)
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    
    -- 版本號 (樂觀鎖，防止併發問題)
    version INT NOT NULL DEFAULT 1,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT chk_locked_balance_non_negative CHECK (locked_balance >= 0),
    CONSTRAINT chk_total_balance CHECK (balance + locked_balance >= 0)
);

-- 索引
CREATE INDEX idx_wallets_player_id ON wallets(player_id);
CREATE INDEX idx_wallets_balance ON wallets(balance DESC);

-- 更新時間觸發器
CREATE TRIGGER update_wallets_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

#### 表 4: `transactions` - 交易記錄

```sql
CREATE TABLE transactions (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    
    -- 交易資訊
    type VARCHAR(50) NOT NULL, -- deposit, withdraw, game_win, game_loss, buy_in, cash_out
    amount BIGINT NOT NULL, -- 正數=入帳，負數=出帳
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    
    -- 關聯資訊
    game_session_id UUID REFERENCES game_sessions(id),
    reference_id VARCHAR(100), -- 外部系統的參考 ID (如支付網關的訂單號)
    
    -- 冪等性 (防止重複扣款)
    idempotency_key VARCHAR(100) UNIQUE,
    
    -- 元數據
    metadata JSONB, -- 額外資訊 (如: 手牌 ID, 獎勵類型等)
    description TEXT,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES accounts(id),
    
    CONSTRAINT chk_amount_not_zero CHECK (amount != 0)
);

-- 索引
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_game_session_id ON transactions(game_session_id) WHERE game_session_id IS NOT NULL;
CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- GIN 索引 for JSONB
CREATE INDEX idx_transactions_metadata ON transactions USING GIN(metadata);
```

---

#### 表 5: `game_sessions` - 遊戲會話

```sql
CREATE TABLE game_sessions (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 遊戲資訊
    game_type VARCHAR(50) NOT NULL, -- poker, baccarat, slot, etc.
    table_id VARCHAR(100) NOT NULL,
    
    -- 玩家資訊
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    
    -- 會話狀態
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, completed, abandoned
    
    -- 籌碼資訊
    buy_in_amount BIGINT NOT NULL,
    cash_out_amount BIGINT,
    net_profit BIGINT, -- cash_out_amount - buy_in_amount
    
    -- 統計
    hands_played INT NOT NULL DEFAULT 0,
    
    -- 時間
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT GENERATED ALWAYS AS (EXTRACT(EPOCH FROM (ended_at - started_at))::INT) STORED,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_game_sessions_player_id ON game_sessions(player_id);
CREATE INDEX idx_game_sessions_table_id ON game_sessions(table_id);
CREATE INDEX idx_game_sessions_status ON game_sessions(status);
CREATE INDEX idx_game_sessions_started_at ON game_sessions(started_at DESC);
CREATE INDEX idx_game_sessions_game_type ON game_sessions(game_type);

-- 複合索引 (常見查詢)
CREATE INDEX idx_game_sessions_player_status ON game_sessions(player_id, status);

-- 更新時間觸發器
CREATE TRIGGER update_game_sessions_updated_at
    BEFORE UPDATE ON game_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

#### 表 6: `hand_history` - 手牌歷史

```sql
CREATE TABLE hand_history (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 關聯
    game_session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    table_id VARCHAR(100) NOT NULL,
    hand_number INT NOT NULL, -- 該桌的第幾手
    
    -- 盲注資訊
    small_blind BIGINT NOT NULL,
    big_blind BIGINT NOT NULL,
    
    -- 遊戲狀態 (使用 JSONB 存儲完整狀態)
    players JSONB NOT NULL, -- 所有玩家的資訊 [{player_id, seat, chips, cards}]
    actions JSONB NOT NULL, -- 所有動作序列 [{player_id, action, amount, timestamp}]
    pots JSONB NOT NULL,    -- 底池資訊 [{amount, contributors, winners}]
    community_cards JSONB,  -- 公共牌 (德撲專用)
    
    -- 結果
    winners JSONB NOT NULL, -- [{player_id, amount}]
    
    -- 時間
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_hand_number UNIQUE(table_id, hand_number)
);

-- 索引
CREATE INDEX idx_hand_history_game_session_id ON hand_history(game_session_id);
CREATE INDEX idx_hand_history_table_id ON hand_history(table_id);
CREATE INDEX idx_hand_history_started_at ON hand_history(started_at DESC);

-- GIN 索引 for JSONB (查詢玩家參與的手牌)
CREATE INDEX idx_hand_history_players ON hand_history USING GIN(players);
CREATE INDEX idx_hand_history_winners ON hand_history USING GIN(winners);

-- 分區策略 (按月分區，歷史數據會很大)
-- 未來可以使用 pg_partman 自動管理
```

---

#### 表 7: `audit_logs` - 審計日誌

```sql
CREATE TABLE audit_logs (
    -- 主鍵
    id BIGSERIAL PRIMARY KEY, -- 使用 BIGSERIAL 提升插入性能
    
    -- 審計資訊
    entity_type VARCHAR(50) NOT NULL, -- account, wallet, transaction, game_session
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- create, update, delete, login, logout
    
    -- 變更資訊
    changes JSONB, -- 變更前後的差異 {before: {...}, after: {...}}
    
    -- 請求資訊
    ip_address INET,
    user_agent TEXT,
    
    -- 執行者
    actor_id UUID REFERENCES accounts(id),
    actor_type VARCHAR(20) NOT NULL DEFAULT 'user', -- user, system, admin
    
    -- 時間 (只需要創建時間)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- GIN 索引 for JSONB
CREATE INDEX idx_audit_logs_changes ON audit_logs USING GIN(changes);

-- 分區策略 (按月分區，審計日誌會快速增長)
-- ALTER TABLE audit_logs PARTITION BY RANGE (created_at);
```

---

### 3.3 輔助表

#### 表 8: `sessions` - Redis Session 備份

```sql
CREATE TABLE sessions (
    -- 主鍵
    id VARCHAR(100) PRIMARY KEY,
    
    -- Session 資訊
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    data JSONB NOT NULL,
    
    -- 過期時間
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_sessions_player_id ON sessions(player_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- 自動清理過期 Session
CREATE INDEX idx_sessions_cleanup ON sessions(expires_at) WHERE expires_at < NOW();
```

---

### 3.4 函數和觸發器

#### 更新時間戳函數

```sql
-- 自動更新 updated_at 欄位
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

#### 錢包餘額檢查函數

```sql
-- 確保餘額不為負
CREATE OR REPLACE FUNCTION check_wallet_balance()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.balance < 0 THEN
        RAISE EXCEPTION 'Wallet balance cannot be negative: %', NEW.balance;
    END IF;
    
    IF NEW.locked_balance < 0 THEN
        RAISE EXCEPTION 'Locked balance cannot be negative: %', NEW.locked_balance;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_wallet_balance
    BEFORE INSERT OR UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION check_wallet_balance();
```

---

## 4. Repository 模式

### 4.1 Repository 介面設計

```go
// internal/infra/repository/interfaces.go
package repository

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// AccountRepository - 帳號管理
type AccountRepository interface {
    // 創建
    Create(ctx context.Context, account *Account) error
    
    // 查詢
    GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
    GetByUsername(ctx context.Context, username string) (*Account, error)
    GetByEmail(ctx context.Context, email string) (*Account, error)
    
    // 更新
    Update(ctx context.Context, account *Account) error
    UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error
    IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error
    ResetFailedAttempts(ctx context.Context, id uuid.UUID) error
    LockAccount(ctx context.Context, id uuid.UUID, until time.Time) error
    
    // 刪除 (軟刪除)
    SoftDelete(ctx context.Context, id uuid.UUID) error
}

// PlayerRepository - 玩家資料
type PlayerRepository interface {
    Create(ctx context.Context, player *Player) error
    GetByID(ctx context.Context, id uuid.UUID) (*Player, error)
    GetByAccountID(ctx context.Context, accountID uuid.UUID) (*Player, error)
    Update(ctx context.Context, player *Player) error
    UpdateStats(ctx context.Context, id uuid.UUID, stats *PlayerStats) error
    
    // 排行榜
    GetTopPlayersByWinnings(ctx context.Context, limit int) ([]*Player, error)
    GetTopPlayersByLevel(ctx context.Context, limit int) ([]*Player, error)
}

// WalletRepository - 錢包管理
type WalletRepository interface {
    Create(ctx context.Context, wallet *Wallet) error
    GetByID(ctx context.Context, id uuid.UUID) (*Wallet, error)
    GetByPlayerID(ctx context.Context, playerID uuid.UUID) (*Wallet, error)
    
    // 餘額操作 (需要事務支援)
    Credit(ctx context.Context, tx Transaction, walletID uuid.UUID, amount int64, txType string, metadata map[string]interface{}) error
    Debit(ctx context.Context, tx Transaction, walletID uuid.UUID, amount int64, txType string, metadata map[string]interface{}) error
    LockBalance(ctx context.Context, tx Transaction, walletID uuid.UUID, amount int64) error
    UnlockBalance(ctx context.Context, tx Transaction, walletID uuid.UUID, amount int64) error
    
    // 查詢 (使用 FOR UPDATE 鎖定)
    GetWithLock(ctx context.Context, tx Transaction, playerID uuid.UUID) (*Wallet, error)
}

// TransactionRepository - 交易記錄
type TransactionRepository interface {
    Create(ctx context.Context, tx *Transaction) error
    GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
    GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*Transaction, error)
    GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*Transaction, error)
    GetByIdempotencyKey(ctx context.Context, key string) (*Transaction, error)
    
    // 統計
    GetTotalByType(ctx context.Context, walletID uuid.UUID, txType string) (int64, error)
}

// GameSessionRepository - 遊戲會話
type GameSessionRepository interface {
    Create(ctx context.Context, session *GameSession) error
    GetByID(ctx context.Context, id uuid.UUID) (*GameSession, error)
    GetActiveByPlayerID(ctx context.Context, playerID uuid.UUID) (*GameSession, error)
    Update(ctx context.Context, session *GameSession) error
    End(ctx context.Context, id uuid.UUID, cashOutAmount int64) error
    
    // 查詢
    GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*GameSession, error)
    GetByTableID(ctx context.Context, tableID string) ([]*GameSession, error)
}

// HandHistoryRepository - 手牌歷史
type HandHistoryRepository interface {
    Create(ctx context.Context, hand *HandHistory) error
    GetByID(ctx context.Context, id uuid.UUID) (*HandHistory, error)
    GetByGameSessionID(ctx context.Context, sessionID uuid.UUID) ([]*HandHistory, error)
    GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*HandHistory, error)
    GetByTableID(ctx context.Context, tableID string, limit, offset int) ([]*HandHistory, error)
}

// UnitOfWork - 工作單元模式 (事務管理)
type UnitOfWork interface {
    Begin(ctx context.Context) (Transaction, error)
    Commit(ctx context.Context, tx Transaction) error
    Rollback(ctx context.Context, tx Transaction) error
}

// Transaction - 事務介面
type Transaction interface {
    Commit() error
    Rollback() error
}
```

---

### 4.2 領域模型

```go
// internal/infra/repository/models.go
package repository

import (
    "time"
    "github.com/google/uuid"
)

// Account - 帳號
type Account struct {
    ID                   uuid.UUID
    Username             string
    Email                string
    PasswordHash         string
    Status               string
    EmailVerified        bool
    FailedLoginAttempts  int
    LockedUntil          *time.Time
    LastLoginAt          *time.Time
    LastLoginIP          string
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

// Player - 玩家
type Player struct {
    ID               uuid.UUID
    AccountID        uuid.UUID
    DisplayName      string
    AvatarURL        string
    Level            int
    Experience       int64
    TotalGamesPlayed int
    TotalHandsPlayed int
    TotalWinnings    int64
    VIPLevel         int
    VIPExpiresAt     *time.Time
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// Wallet - 錢包
type Wallet struct {
    ID            uuid.UUID
    PlayerID      uuid.UUID
    Balance       int64
    LockedBalance int64
    Currency      string
    Version       int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// Transaction - 交易
type Transaction struct {
    ID              uuid.UUID
    WalletID        uuid.UUID
    Type            string
    Amount          int64
    BalanceBefore   int64
    BalanceAfter    int64
    GameSessionID   *uuid.UUID
    ReferenceID     string
    IdempotencyKey  string
    Metadata        map[string]interface{}
    Description     string
    CreatedAt       time.Time
    CreatedBy       *uuid.UUID
}

// GameSession - 遊戲會話
type GameSession struct {
    ID             uuid.UUID
    GameType       string
    TableID        string
    PlayerID       uuid.UUID
    Status         string
    BuyInAmount    int64
    CashOutAmount  *int64
    NetProfit      *int64
    HandsPlayed    int
    StartedAt      time.Time
    EndedAt        *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// HandHistory - 手牌歷史
type HandHistory struct {
    ID             uuid.UUID
    GameSessionID  uuid.UUID
    TableID        string
    HandNumber     int
    SmallBlind     int64
    BigBlind       int64
    Players        map[string]interface{} // JSONB
    Actions        []interface{}          // JSONB
    Pots           []interface{}          // JSONB
    CommunityCards []interface{}          // JSONB
    Winners        []interface{}          // JSONB
    StartedAt      time.Time
    EndedAt        *time.Time
    DurationSecs   int
    CreatedAt      time.Time
}
```

---

## 5. 遷移策略

### 5.1 使用 golang-migrate

#### 安裝
```bash
# 安裝 CLI 工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 或使用 Docker
docker pull migrate/migrate
```

#### 創建遷移文件

```bash
# 創建 migrations 目錄
mkdir -p migrations

# 創建初始化遷移
migrate create -ext sql -dir migrations -seq init_schema

# 這會創建兩個文件:
# migrations/000001_init_schema.up.sql   (升級)
# migrations/000001_init_schema.down.sql (降級)
```

#### 執行遷移

```bash
# 升級到最新版本
migrate -path migrations -database "postgres://user:pass@localhost:5432/thenuts?sslmode=disable" up

# 降級一個版本
migrate -path migrations -database "postgres://user:pass@localhost:5432/thenuts?sslmode=disable" down 1

# 查看當前版本
migrate -path migrations -database "postgres://user:pass@localhost:5432/thenuts?sslmode=disable" version
```

### 5.2 遷移文件組織

```
migrations/
├── 000001_init_schema.up.sql         # 初始 Schema
├── 000001_init_schema.down.sql       
├── 000002_add_indexes.up.sql         # 索引優化
├── 000002_add_indexes.down.sql
├── 000003_add_audit_logs.up.sql      # 審計日誌
├── 000003_add_audit_logs.down.sql
└── ...
```

---

## 6. 實作步驟

### Step 1: 環境準備 (1-2 小時)

```bash
# 1. 安裝 PostgreSQL (使用 Docker)
docker run --name thenuts-postgres \
  -e POSTGRES_USER=thenuts \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=thenuts \
  -p 5432:5432 \
  -d postgres:15-alpine

# 2. 安裝 Redis
docker run --name thenuts-redis \
  -p 6379:6379 \
  -d redis:7-alpine

# 3. 驗證連接
psql -h localhost -U thenuts -d thenuts
redis-cli ping
```

### Step 2: 更新 go.mod (10 分鐘)

```bash
# 添加依賴
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/redis/go-redis/v9
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

### Step 3: 創建遷移文件 (2-3 小時)

詳見下一個文件：`migrations/000001_init_schema.up.sql`

### Step 4: 實作 Repository (8-12 小時)

```
internal/infra/repository/
├── interfaces.go           # Repository 介面
├── models.go              # 領域模型
├── postgres/
│   ├── account_repo.go    # Account Repository 實作
│   ├── player_repo.go     # Player Repository 實作
│   ├── wallet_repo.go     # Wallet Repository 實作 ⭐ 重點
│   ├── transaction_repo.go
│   ├── game_session_repo.go
│   └── hand_history_repo.go
└── redis/
    ├── session_store.go   # Session 緩存
    └── ticket_store.go    # Ticket 緩存 (替代 Memory 版)
```

### Step 5: 連接池配置 (1-2 小時)

```go
// internal/infra/database/postgres.go
package database

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
    Host         string
    Port         int
    User         string
    Password     string
    Database     string
    MaxConns     int32
    MinConns     int32
    MaxConnLife  time.Duration
}

func NewPostgresPool(cfg PostgresConfig) (*pgxpool.Pool, error) {
    dsn := fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=disable",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
    )
    
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    
    config.MaxConns = cfg.MaxConns
    config.MinConns = cfg.MinConns
    config.MaxConnLifetime = cfg.MaxConnLife
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return nil, err
    }
    
    // 測試連接
    if err := pool.Ping(context.Background()); err != nil {
        return nil, err
    }
    
    return pool, nil
}
```

### Step 6: 更新配置文件 (10 分鐘)

```yaml
# config.yaml
database:
  postgres:
    host: localhost
    port: 5432
    user: thenuts
    password: devpassword
    database: thenuts
    max_conns: 25
    min_conns: 5
    max_conn_lifetime: 5m
    
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0
    pool_size: 10
```

### Step 7: 整合到 DI (2-3 小時)

```go
// pkg/di/provider.go

// ProvidePostgresPool 提供 PostgreSQL 連接池
func ProvidePostgresPool(cfg *config.Config) (*pgxpool.Pool, error) {
    return database.NewPostgresPool(database.PostgresConfig{
        Host:        cfg.Database.Postgres.Host,
        Port:        cfg.Database.Postgres.Port,
        User:        cfg.Database.Postgres.User,
        Password:    cfg.Database.Postgres.Password,
        Database:    cfg.Database.Postgres.Database,
        MaxConns:    cfg.Database.Postgres.MaxConns,
        MinConns:    cfg.Database.Postgres.MinConns,
        MaxConnLife: cfg.Database.Postgres.MaxConnLifetime,
    })
}

// ProvideAccountRepository 提供 Account Repository
func ProvideAccountRepository(pool *pgxpool.Pool) repository.AccountRepository {
    return postgres.NewAccountRepository(pool)
}

// ... 其他 Repositories
```

### Step 8: 編寫測試 (4-6 小時)

```go
// internal/infra/repository/postgres/wallet_repo_test.go
package postgres_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestWalletRepository_CreditDebit(t *testing.T) {
    // 設置測試數據庫
    pool := setupTestDB(t)
    defer pool.Close()
    
    repo := postgres.NewWalletRepository(pool)
    
    // 測試入帳
    // ...
}
```

---

## 7. 性能優化建議

### 7.1 索引策略

- ✅ 主鍵自動索引 (UUID/BIGSERIAL)
- ✅ 外鍵索引
- ✅ 常用查詢字段 (email, username, status)
- ✅ 複合索引 (player_id + status)
- ✅ 部分索引 (WHERE status != 'active')
- ✅ GIN 索引 (JSONB 字段)

### 7.2 連接池配置

```
生產環境建議:
- MaxConns: CPU 核心數 × 2 + 磁碟數
- MinConns: MaxConns / 4
- MaxConnLifetime: 5-10 分鐘
```

### 7.3 查詢優化

- 使用 `EXPLAIN ANALYZE` 分析慢查詢
- 避免 `SELECT *`，只查詢需要的欄位
- 使用 Prepared Statements
- 批量操作使用 `COPY` 或 `INSERT ... VALUES (...), (...)`

---

## 8. 安全考量

### 8.1 SQL 注入防護

✅ **使用參數化查詢**
```go
// ✅ 安全
row := pool.QueryRow(ctx, "SELECT * FROM accounts WHERE username = $1", username)

// ❌ 危險
query := fmt.Sprintf("SELECT * FROM accounts WHERE username = '%s'", username)
```

### 8.2 密碼安全

```go
import "golang.org/x/crypto/bcrypt"

// 生成密碼 hash
hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// 驗證密碼
err := bcrypt.CompareHashAndPassword(hash, []byte(password))
```

### 8.3 防止重複扣款

使用 `idempotency_key` 確保操作冪等性：

```go
func (r *WalletRepository) Debit(ctx context.Context, tx Transaction, walletID uuid.UUID, amount int64, idempotencyKey string) error {
    // 1. 檢查是否已經處理過
    existing, _ := r.transactionRepo.GetByIdempotencyKey(ctx, idempotencyKey)
    if existing != nil {
        return ErrDuplicateTransaction
    }
    
    // 2. 執行扣款 (使用 SELECT FOR UPDATE 鎖定)
    // ...
}
```

---

## 9. 監控與維護

### 9.1 健康檢查

```go
func (db *DB) HealthCheck(ctx context.Context) error {
    return db.pool.Ping(ctx)
}
```

### 9.2 連接池監控

```go
stats := pool.Stat()
log.Info("Pool stats",
    "total_conns", stats.TotalConns(),
    "idle_conns", stats.IdleConns(),
    "acquired_conns", stats.AcquiredConns(),
)
```

### 9.3 慢查詢日誌

PostgreSQL 配置:
```sql
ALTER DATABASE thenuts SET log_min_duration_statement = 1000; -- 1秒
```

---

## 10. 下一步

1. ✅ 閱讀完整設計文檔
2. ⏳ 創建遷移文件 (`migrations/000001_init_schema.up.sql`)
3. ⏳ 實作 Wallet Repository (最重要)
4. ⏳ 編寫單元測試
5. ⏳ 整合到現有系統

---

**文檔版本**: v1.0  
**最後更新**: 2026-01-26  
**預估完成時間**: 12-16 小時
