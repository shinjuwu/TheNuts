-- ============================================================================
-- TheNuts 數據庫初始化 Schema
-- Version: 1.0
-- Date: 2026-01-26
-- ============================================================================

-- 啟用 UUID 擴展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- 輔助函數
-- ============================================================================

-- 自動更新 updated_at 欄位
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 錢包餘額檢查
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

-- ============================================================================
-- 表 1: accounts - 帳號認證
-- ============================================================================

CREATE TABLE accounts (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 認證資訊
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    
    -- 狀態
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    email_verified BOOLEAN NOT NULL DEFAULT false,
    
    -- 安全
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT chk_username_length CHECK (char_length(username) >= 3),
    CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'),
    CONSTRAINT chk_status_value CHECK (status IN ('active', 'suspended', 'banned', 'deleted'))
);

-- 索引
CREATE INDEX idx_accounts_email ON accounts(email);
CREATE INDEX idx_accounts_username ON accounts(username);
CREATE INDEX idx_accounts_status ON accounts(status) WHERE status != 'active';
CREATE INDEX idx_accounts_created_at ON accounts(created_at DESC);

-- 觸發器
CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 註釋
COMMENT ON TABLE accounts IS '使用者帳號認證資訊';
COMMENT ON COLUMN accounts.password_hash IS 'bcrypt hash of password';
COMMENT ON COLUMN accounts.failed_login_attempts IS '登入失敗次數，5次後鎖定';
COMMENT ON COLUMN accounts.locked_until IS '帳號鎖定至此時間';

-- ============================================================================
-- 表 2: players - 玩家資料
-- ============================================================================

CREATE TABLE players (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    
    -- 玩家資訊
    display_name VARCHAR(50) NOT NULL,
    avatar_url VARCHAR(500),
    level INT NOT NULL DEFAULT 1,
    experience BIGINT NOT NULL DEFAULT 0,
    
    -- 統計資料
    total_games_played INT NOT NULL DEFAULT 0,
    total_hands_played INT NOT NULL DEFAULT 0,
    total_winnings BIGINT NOT NULL DEFAULT 0,
    
    -- VIP 狀態
    vip_level INT NOT NULL DEFAULT 0,
    vip_expires_at TIMESTAMPTZ,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT chk_display_name_length CHECK (char_length(display_name) >= 2),
    CONSTRAINT chk_level_positive CHECK (level >= 1),
    CONSTRAINT chk_vip_level_range CHECK (vip_level >= 0 AND vip_level <= 10)
);

-- 索引
CREATE INDEX idx_players_account_id ON players(account_id);
CREATE INDEX idx_players_display_name ON players(display_name);
CREATE INDEX idx_players_level ON players(level DESC);
CREATE INDEX idx_players_total_winnings ON players(total_winnings DESC);
CREATE INDEX idx_players_vip_level ON players(vip_level DESC);

-- 觸發器
CREATE TRIGGER update_players_updated_at
    BEFORE UPDATE ON players
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 註釋
COMMENT ON TABLE players IS '玩家遊戲資料';
COMMENT ON COLUMN players.total_winnings IS '歷史總贏得金額（可為負數）';

-- ============================================================================
-- 表 3: wallets - 錢包餘額
-- ============================================================================

CREATE TABLE wallets (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
    
    -- 餘額 (BIGINT 存儲，單位：分/角子)
    balance BIGINT NOT NULL DEFAULT 0,
    locked_balance BIGINT NOT NULL DEFAULT 0,
    
    -- 貨幣類型
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    
    -- 樂觀鎖版本號
    version INT NOT NULL DEFAULT 1,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT chk_locked_balance_non_negative CHECK (locked_balance >= 0)
);

-- 索引
CREATE INDEX idx_wallets_player_id ON wallets(player_id);
CREATE INDEX idx_wallets_balance ON wallets(balance DESC);
CREATE INDEX idx_wallets_currency ON wallets(currency);

-- 觸發器
CREATE TRIGGER update_wallets_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_check_wallet_balance
    BEFORE INSERT OR UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION check_wallet_balance();

-- 註釋
COMMENT ON TABLE wallets IS '玩家錢包餘額';
COMMENT ON COLUMN wallets.balance IS '可用餘額（單位：分）';
COMMENT ON COLUMN wallets.locked_balance IS '鎖定餘額（進行中的遊戲）';
COMMENT ON COLUMN wallets.version IS '樂觀鎖版本號，每次更新+1';

-- ============================================================================
-- 表 4: game_sessions - 遊戲會話
-- ============================================================================

CREATE TABLE game_sessions (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 遊戲資訊
    game_type VARCHAR(50) NOT NULL,
    table_id VARCHAR(100) NOT NULL,
    
    -- 玩家資訊
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    
    -- 會話狀態
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    
    -- 籌碼資訊
    buy_in_amount BIGINT NOT NULL,
    cash_out_amount BIGINT,
    net_profit BIGINT,
    
    -- 統計
    hands_played INT NOT NULL DEFAULT 0,
    
    -- 時間
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT chk_status_value CHECK (status IN ('active', 'completed', 'abandoned')),
    CONSTRAINT chk_buy_in_positive CHECK (buy_in_amount > 0)
);

-- 索引
CREATE INDEX idx_game_sessions_player_id ON game_sessions(player_id);
CREATE INDEX idx_game_sessions_table_id ON game_sessions(table_id);
CREATE INDEX idx_game_sessions_status ON game_sessions(status);
CREATE INDEX idx_game_sessions_started_at ON game_sessions(started_at DESC);
CREATE INDEX idx_game_sessions_game_type ON game_sessions(game_type);
CREATE INDEX idx_game_sessions_player_status ON game_sessions(player_id, status);

-- 觸發器
CREATE TRIGGER update_game_sessions_updated_at
    BEFORE UPDATE ON game_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 註釋
COMMENT ON TABLE game_sessions IS '玩家遊戲會話記錄';
COMMENT ON COLUMN game_sessions.net_profit IS '淨盈虧 = cash_out_amount - buy_in_amount';

-- ============================================================================
-- 表 5: transactions - 交易記錄
-- ============================================================================

CREATE TABLE transactions (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    
    -- 交易資訊
    type VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    
    -- 關聯資訊
    game_session_id UUID REFERENCES game_sessions(id),
    reference_id VARCHAR(100),
    
    -- 冪等性
    idempotency_key VARCHAR(100) UNIQUE,
    
    -- 元數據
    metadata JSONB,
    description TEXT,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES accounts(id),
    
    -- 約束
    CONSTRAINT chk_amount_not_zero CHECK (amount != 0),
    CONSTRAINT chk_type_value CHECK (type IN ('deposit', 'withdraw', 'game_win', 'game_loss', 'buy_in', 'cash_out', 'refund', 'bonus'))
);

-- 索引
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_game_session_id ON transactions(game_session_id) WHERE game_session_id IS NOT NULL;
CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_transactions_reference_id ON transactions(reference_id) WHERE reference_id IS NOT NULL;

-- GIN 索引 for JSONB
CREATE INDEX idx_transactions_metadata ON transactions USING GIN(metadata);

-- 註釋
COMMENT ON TABLE transactions IS '交易記錄（不可變）';
COMMENT ON COLUMN transactions.amount IS '交易金額：正數=入帳，負數=出帳';
COMMENT ON COLUMN transactions.idempotency_key IS '冪等性鍵，防止重複扣款';

-- ============================================================================
-- 表 6: hand_history - 手牌歷史
-- ============================================================================

CREATE TABLE hand_history (
    -- 主鍵
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 關聯
    game_session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    table_id VARCHAR(100) NOT NULL,
    hand_number INT NOT NULL,
    
    -- 盲注資訊
    small_blind BIGINT NOT NULL,
    big_blind BIGINT NOT NULL,
    
    -- 遊戲狀態 (JSONB)
    players JSONB NOT NULL,
    actions JSONB NOT NULL,
    pots JSONB NOT NULL,
    community_cards JSONB,
    
    -- 結果
    winners JSONB NOT NULL,
    
    -- 時間
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INT,
    
    -- 審計
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT uq_hand_number UNIQUE(table_id, hand_number)
);

-- 索引
CREATE INDEX idx_hand_history_game_session_id ON hand_history(game_session_id);
CREATE INDEX idx_hand_history_table_id ON hand_history(table_id);
CREATE INDEX idx_hand_history_started_at ON hand_history(started_at DESC);

-- GIN 索引 for JSONB
CREATE INDEX idx_hand_history_players ON hand_history USING GIN(players);
CREATE INDEX idx_hand_history_winners ON hand_history USING GIN(winners);
CREATE INDEX idx_hand_history_actions ON hand_history USING GIN(actions);

-- 註釋
COMMENT ON TABLE hand_history IS '手牌歷史記錄';
COMMENT ON COLUMN hand_history.players IS 'JSONB array: [{player_id, seat, chips, cards}]';
COMMENT ON COLUMN hand_history.actions IS 'JSONB array: [{player_id, action, amount, timestamp}]';

-- ============================================================================
-- 表 7: audit_logs - 審計日誌
-- ============================================================================

CREATE TABLE audit_logs (
    -- 主鍵 (使用 BIGSERIAL 提升插入性能)
    id BIGSERIAL PRIMARY KEY,
    
    -- 審計資訊
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    
    -- 變更資訊
    changes JSONB,
    
    -- 請求資訊
    ip_address INET,
    user_agent TEXT,
    
    -- 執行者
    actor_id UUID REFERENCES accounts(id),
    actor_type VARCHAR(20) NOT NULL DEFAULT 'user',
    
    -- 時間
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 約束
    CONSTRAINT chk_actor_type CHECK (actor_type IN ('user', 'system', 'admin'))
);

-- 索引
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- GIN 索引 for JSONB
CREATE INDEX idx_audit_logs_changes ON audit_logs USING GIN(changes);

-- 註釋
COMMENT ON TABLE audit_logs IS '審計日誌（所有重要操作）';
COMMENT ON COLUMN audit_logs.changes IS 'JSONB: {before: {...}, after: {...}}';

-- ============================================================================
-- 表 8: sessions - Session 備份
-- ============================================================================

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
CREATE INDEX idx_sessions_cleanup ON sessions(expires_at) WHERE expires_at < NOW();

-- 觸發器
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 註釋
COMMENT ON TABLE sessions IS 'Redis Session 的持久化備份';

-- ============================================================================
-- 初始化數據
-- ============================================================================

-- 創建系統管理員帳號 (密碼: admin123，需在生產環境更換)
INSERT INTO accounts (id, username, email, password_hash, status, email_verified)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'admin@thenuts.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- admin123
    'active',
    true
);

INSERT INTO players (id, account_id, display_name, level)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Admin',
    999
);

INSERT INTO wallets (player_id, balance, currency)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    1000000000, -- 10,000,000.00
    'USD'
);

-- ============================================================================
-- 完成
-- ============================================================================

-- 顯示所有表
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 統計信息
SELECT 
    COUNT(*) FILTER (WHERE table_type = 'BASE TABLE') AS tables,
    COUNT(*) FILTER (WHERE table_type = 'VIEW') AS views
FROM information_schema.tables
WHERE table_schema = 'public';
