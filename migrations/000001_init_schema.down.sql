-- ============================================================================
-- TheNuts 數據庫降級 Schema
-- Version: 1.0
-- Date: 2026-01-26
-- Description: 回滾初始化 Schema
-- ============================================================================

-- 警告：此操作將刪除所有數據！

-- 刪除表 (按依賴順序反向刪除)
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS hand_history CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS game_sessions CASCADE;
DROP TABLE IF EXISTS wallets CASCADE;
DROP TABLE IF EXISTS players CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;

-- 刪除函數
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
DROP FUNCTION IF EXISTS check_wallet_balance() CASCADE;

-- 刪除擴展 (謹慎，可能影響其他應用)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
-- DROP EXTENSION IF EXISTS "pgcrypto";

-- 顯示結果
DO $$
BEGIN
    RAISE NOTICE 'All tables and functions have been dropped successfully.';
END $$;
