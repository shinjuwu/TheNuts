-- 创建测试用户的脚本
-- 密码都是: password123
-- bcrypt hash 使用 cost=12 生成

-- 清理旧数据（可选，谨慎使用）
-- DELETE FROM transactions;
-- DELETE FROM wallets;
-- DELETE FROM players;
-- DELETE FROM accounts WHERE username LIKE 'test%';

-- 测试用户 1: testuser1
INSERT INTO accounts (id, username, email, password_hash, status, email_verified, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001'::uuid,
    'testuser1',
    'testuser1@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5jtHq7PmVxQyG', -- password123
    'active',
    true,
    NOW(),
    NOW()
) ON CONFLICT (username) DO NOTHING;

INSERT INTO players (id, account_id, display_name, level, vip_level, created_at, updated_at)
VALUES (
    'p0000000-0000-0000-0000-000000000001'::uuid,
    'a0000000-0000-0000-0000-000000000001'::uuid,
    'Test User 1',
    5,
    1,
    NOW(),
    NOW()
) ON CONFLICT (account_id) DO NOTHING;

INSERT INTO wallets (id, player_id, balance, locked_balance, currency, version, created_at, updated_at)
VALUES (
    'w0000000-0000-0000-0000-000000000001'::uuid,
    'p0000000-0000-0000-0000-000000000001'::uuid,
    100000, -- $1000.00
    0,
    'USD',
    1,
    NOW(),
    NOW()
) ON CONFLICT (player_id) DO NOTHING;

-- 测试用户 2: testuser2
INSERT INTO accounts (id, username, email, password_hash, status, email_verified, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000002'::uuid,
    'testuser2',
    'testuser2@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5jtHq7PmVxQyG', -- password123
    'active',
    true,
    NOW(),
    NOW()
) ON CONFLICT (username) DO NOTHING;

INSERT INTO players (id, account_id, display_name, level, vip_level, created_at, updated_at)
VALUES (
    'p0000000-0000-0000-0000-000000000002'::uuid,
    'a0000000-0000-0000-0000-000000000002'::uuid,
    'Test User 2',
    3,
    0,
    NOW(),
    NOW()
) ON CONFLICT (account_id) DO NOTHING;

INSERT INTO wallets (id, player_id, balance, locked_balance, currency, version, created_at, updated_at)
VALUES (
    'w0000000-0000-0000-0000-000000000002'::uuid,
    'p0000000-0000-0000-0000-000000000002'::uuid,
    50000, -- $500.00
    0,
    'USD',
    1,
    NOW(),
    NOW()
) ON CONFLICT (player_id) DO NOTHING;

-- 测试用户 3: testuser3
INSERT INTO accounts (id, username, email, password_hash, status, email_verified, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000003'::uuid,
    'testuser3',
    'testuser3@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5jtHq7PmVxQyG', -- password123
    'active',
    true,
    NOW(),
    NOW()
) ON CONFLICT (username) DO NOTHING;

INSERT INTO players (id, account_id, display_name, level, vip_level, created_at, updated_at)
VALUES (
    'p0000000-0000-0000-0000-000000000003'::uuid,
    'a0000000-0000-0000-0000-000000000003'::uuid,
    'Test User 3',
    1,
    0,
    NOW(),
    NOW()
) ON CONFLICT (account_id) DO NOTHING;

INSERT INTO wallets (id, player_id, balance, locked_balance, currency, version, created_at, updated_at)
VALUES (
    'w0000000-0000-0000-0000-000000000003'::uuid,
    'p0000000-0000-0000-0000-000000000003'::uuid,
    25000, -- $250.00
    0,
    'USD',
    1,
    NOW(),
    NOW()
) ON CONFLICT (player_id) DO NOTHING;

-- 验证插入
SELECT 
    a.username,
    a.email,
    a.status,
    p.display_name,
    w.balance / 100.0 as balance_usd
FROM accounts a
JOIN players p ON a.id = p.account_id
JOIN wallets w ON p.id = w.player_id
WHERE a.username LIKE 'test%'
ORDER BY a.username;
