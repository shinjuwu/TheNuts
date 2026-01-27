# 认证系统测试指南

## 概述

本文档说明如何测试新实现的认证系统，包括用户注册、登录、JWT Token 生成和 WebSocket 票券获取。

## 系统要求

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL (通过 Docker)
- Redis (通过 Docker)

## 快速开始

### 1. 启动数据库服务

```bash
docker-compose up -d
```

等待数据库就绪（约 10-15 秒）。

### 2. 创建测试用户（可选）

如果想使用预先创建的测试用户：

```bash
docker exec -i thenuts-postgres psql -U thenuts -d thenuts < scripts/create_test_user.sql
```

这将创建以下测试用户：
- **Username**: `testuser1`
- **Password**: `password123`
- **Initial Balance**: $1000.00

### 3. 编译并运行服务器

```bash
# 编译
go build -o game-server.exe ./cmd/game-server

# 运行
./game-server.exe
```

服务器将在 `http://localhost:8080` 启动。

### 4. 运行自动化测试

使用 PowerShell 运行测试脚本：

```powershell
./test_auth.ps1
```

## API 端点

### 1. 注册新用户

**POST** `/api/auth/register`

**请求体**:
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "securepassword"
}
```

**成功响应** (201 Created):
```json
{
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "player_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "username": "newuser",
  "message": "Registration successful. Please login."
}
```

**错误响应** (409 Conflict):
```json
{
  "error": "username_exists",
  "message": "Username already exists"
}
```

### 2. 用户登录

**POST** `/api/auth/login`

**请求体**:
```json
{
  "username": "testuser1",
  "password": "password123"
}
```

**成功响应** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "player_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "testuser1",
  "display_name": "Test User One"
}
```

**错误响应** (401 Unauthorized):
```json
{
  "error": "invalid_credentials",
  "message": "Invalid username or password"
}
```

**错误响应** (403 Forbidden - 账号锁定):
```json
{
  "error": "account_locked",
  "message": "Account is temporarily locked due to too many failed login attempts"
}
```

### 3. 获取 WebSocket 票券

**POST** `/api/auth/ticket`

**请求头**:
```
Authorization: Bearer <jwt_token>
```

**请求体** (可选):
```json
{
  "table_id": "table_001"
}
```

**成功响应** (200 OK):
```json
{
  "ticket": "a7f3d9c8e4b2f1a6...",
  "expires_in": 30,
  "ws_url": "ws://localhost:8080/ws?ticket=a7f3d9c8e4b2f1a6..."
}
```

## 手动测试示例

### 使用 curl

#### 1. 注册新用户

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "alicepassword"
  }'
```

#### 2. 登录

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "alicepassword"
  }'
```

保存返回的 `token`。

#### 3. 获取票券

```bash
curl -X POST http://localhost:8080/api/auth/ticket \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token_here>" \
  -d '{}'
```

### 使用 PowerShell

#### 1. 注册新用户

```powershell
$body = @{
    username = "bob"
    email = "bob@example.com"
    password = "bobpassword"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/auth/register" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body
```

#### 2. 登录

```powershell
$body = @{
    username = "bob"
    password = "bobpassword"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/auth/login" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

$token = $response.token
```

#### 3. 获取票券

```powershell
$headers = @{
    "Authorization" = "Bearer $token"
}

Invoke-RestMethod -Uri "http://localhost:8080/api/auth/ticket" `
    -Method Post `
    -ContentType "application/json" `
    -Headers $headers `
    -Body "{}"
```

## 安全特性

### 1. 密码哈希 (bcrypt)

- 使用 bcrypt 算法，cost=12
- 密码永不明文存储
- 每个密码都有独特的 salt

### 2. 账号锁定机制

- 5 次连续失败登录后自动锁定
- 锁定时长：30 分钟
- 成功登录后自动重置失败计数

### 3. JWT Token

- 使用 HMAC-SHA256 签名
- 有效期：24 小时
- 包含 player_id 和 username

### 4. WebSocket 票券

- 一次性使用
- 短效期限：30 秒
- 使用 `crypto/rand` 生成
- 验证后立即销毁

### 5. 审计日志

- 记录所有登录尝试（包括失败）
- 记录 IP 地址
- 记录最后登录时间

## 测试场景

### 场景 1: 新用户注册并登录

1. 注册新用户
2. 使用新用户登录
3. 获取 WebSocket 票券
4. 验证所有步骤成功

### 场景 2: 重复注册保护

1. 注册用户 A
2. 尝试再次注册相同用户名 → 应失败
3. 尝试注册相同邮箱 → 应失败

### 场景 3: 密码验证

1. 使用正确密码登录 → 成功
2. 使用错误密码登录 → 失败
3. 验证错误消息不泄露用户是否存在

### 场景 4: 账号锁定

1. 连续 5 次使用错误密码登录
2. 账号应被锁定
3. 使用正确密码登录也应失败
4. 等待 30 分钟后应自动解锁

### 场景 5: JWT Token 验证

1. 登录获取 Token
2. 使用 Token 获取票券 → 成功
3. 使用无效 Token → 应失败
4. 使用过期 Token → 应失败

## 数据库验证

### 查看注册的用户

```sql
SELECT 
    a.username,
    a.email,
    a.status,
    a.email_verified,
    p.display_name,
    w.balance / 100.0 as balance_usd
FROM accounts a
JOIN players p ON a.id = p.account_id
LEFT JOIN wallets w ON p.id = w.player_id
ORDER BY a.created_at DESC
LIMIT 10;
```

### 查看登录失败记录

```sql
SELECT 
    username,
    failed_login_attempts,
    locked_until,
    last_login_at,
    last_login_ip
FROM accounts
WHERE failed_login_attempts > 0
ORDER BY failed_login_attempts DESC;
```

### 查看最近的交易

```sql
SELECT 
    t.id,
    t.type,
    t.amount / 100.0 as amount_usd,
    t.description,
    t.created_at,
    p.display_name
FROM transactions t
JOIN wallets w ON t.wallet_id = w.id
JOIN players p ON w.player_id = p.id
ORDER BY t.created_at DESC
LIMIT 20;
```

## 故障排查

### 问题: "connection refused"

**解决方案**: 确保数据库正在运行

```bash
docker-compose ps
docker-compose up -d
```

### 问题: "table does not exist"

**解决方案**: 运行数据库迁移

```bash
docker exec -i thenuts-postgres psql -U thenuts -d thenuts < migrations/000001_init_schema.up.sql
docker exec -i thenuts-postgres psql -U thenuts -d thenuts < migrations/000002_add_idempotency_constraint.up.sql
```

### 问题: "invalid_credentials" 即使密码正确

**解决方案**: 检查账号状态和锁定情况

```sql
SELECT username, status, locked_until, failed_login_attempts 
FROM accounts 
WHERE username = 'your_username';
```

如果被锁定，可以手动解锁：

```sql
UPDATE accounts 
SET locked_until = NULL, failed_login_attempts = 0 
WHERE username = 'your_username';
```

### 问题: bcrypt hash 不匹配

**解决方案**: 使用正确的 bcrypt hash 生成工具

在 Go 中生成 hash:

```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "your_password"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
    fmt.Println(string(hash))
}
```

## 性能考虑

- bcrypt 验证较慢（故意设计），约 100-200ms
- 建议在生产环境使用 Redis 缓存会话
- JWT Token 验证很快（< 1ms）
- 票券验证也很快（内存操作）

## 下一步

完成认证系统测试后，可以继续：

1. **WebSocket Handler 改造** - 整合真实扣款
2. **Game Service 层** - 实现游戏业务逻辑
3. **速率限制** - 防止暴力破解
4. **Redis TicketStore** - 分布式票券存储

## 参考资料

- [bcrypt 文档](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- [JWT 标准](https://jwt.io/)
- [OWASP 认证备忘单](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
