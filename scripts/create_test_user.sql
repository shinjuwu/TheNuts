-- 创建单个测试用户
-- Username: testuser1
-- Password: password123
-- bcrypt hash (cost=12): $2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5jtHq7PmVxQyG

BEGIN;

-- 插入账号
INSERT INTO accounts (id, username, email, password_hash, status, email_verified, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'testuser1',
    'testuser1@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5jtHq7PmVxQyG',
    'active',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (username) DO UPDATE 
SET password_hash = EXCLUDED.password_hash
RETURNING id;

-- 获取账号 ID
DO $$
DECLARE
    v_account_id uuid;
    v_player_id uuid;
BEGIN
    SELECT id INTO v_account_id FROM accounts WHERE username = 'testuser1';
    
    -- 插入或更新玩家
    INSERT INTO players (id, account_id, display_name, level, vip_level, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        v_account_id,
        'Test User One',
        1,
        0,
        NOW(),
        NOW()
    )
    ON CONFLICT (account_id) DO UPDATE
    SET display_name = EXCLUDED.display_name,
        updated_at = NOW()
    RETURNING id INTO v_player_id;
    
    -- 插入或更新钱包
    INSERT INTO wallets (id, player_id, balance, locked_balance, currency, version, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        v_player_id,
        100000, -- $1000.00
        0,
        'USD',
        1,
        NOW(),
        NOW()
    )
    ON CONFLICT (player_id) DO UPDATE
    SET balance = EXCLUDED.balance,
        updated_at = NOW();
    
    RAISE NOTICE 'Test user created: testuser1';
    RAISE NOTICE 'Account ID: %', v_account_id;
    RAISE NOTICE 'Player ID: %', v_player_id;
END $$;

COMMIT;

-- 验证
SELECT 
    a.id as account_id,
    a.username,
    a.email,
    a.status,
    p.id as player_id,
    p.display_name,
    w.balance / 100.0 as balance_usd
FROM accounts a
JOIN players p ON a.id = p.account_id
JOIN wallets w ON p.id = w.player_id
WHERE a.username = 'testuser1';
