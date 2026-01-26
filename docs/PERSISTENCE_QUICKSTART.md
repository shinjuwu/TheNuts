# 持久化層快速開始指南

## 🚀 5 分鐘快速上手

### 步驟 1: 啟動數據庫 (Docker)

```bash
# 創建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: thenuts-postgres
    environment:
      POSTGRES_USER: thenuts
      POSTGRES_PASSWORD: devpassword
      POSTGRES_DB: thenuts
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U thenuts"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: thenuts-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  # 可選: pgAdmin (數據庫管理界面)
  pgadmin:
    image: dpage/pgadmin4:latest
    container_name: thenuts-pgadmin
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@thenuts.com
      PGADMIN_DEFAULT_PASSWORD: admin
    ports:
      - "5050:80"
    depends_on:
      - postgres

volumes:
  postgres_data:
  redis_data:
EOF

# 啟動
docker-compose up -d

# 檢查狀態
docker-compose ps
```

**預期輸出**:
```
NAME                 IMAGE                      STATUS
thenuts-postgres     postgres:15-alpine         Up (healthy)
thenuts-redis        redis:7-alpine             Up (healthy)
thenuts-pgadmin      dpage/pgadmin4:latest      Up
```

---

### 步驟 2: 安裝遷移工具

```bash
# 方法 1: 使用 Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 方法 2: 使用 Docker (推薦)
alias migrate='docker run --rm -v $(pwd)/migrations:/migrations --network host migrate/migrate'

# 驗證安裝
migrate -version
```

---

### 步驟 3: 執行遷移

```bash
# 設置數據庫連接字串
export DATABASE_URL="postgres://thenuts:devpassword@localhost:5432/thenuts?sslmode=disable"

# 執行遷移 (升級到最新版本)
migrate -path migrations -database "$DATABASE_URL" up

# 查看當前版本
migrate -path migrations -database "$DATABASE_URL" version
```

**預期輸出**:
```
1/u init_schema (123.456ms)
```

---

### 步驟 4: 驗證 Schema

```bash
# 連接數據庫
psql -h localhost -U thenuts -d thenuts

# 列出所有表
\dt

# 查看 accounts 表結構
\d accounts

# 查看所有索引
\di

# 退出
\q
```

**預期輸出**:
```
 Schema |     Name      | Type  | Owner
--------+---------------+-------+--------
 public | accounts      | table | thenuts
 public | audit_logs    | table | thenuts
 public | game_sessions | table | thenuts
 public | hand_history  | table | thenuts
 public | players       | table | thenuts
 public | sessions      | table | thenuts
 public | transactions  | table | thenuts
 public | wallets       | table | thenuts
(8 rows)
```

---

### 步驟 5: 更新配置文件

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

---

### 步驟 6: 更新 go.mod

```bash
# 添加依賴
go get github.com/jackc/pgx/v5@latest
go get github.com/jackc/pgx/v5/pgxpool@latest
go get github.com/redis/go-redis/v9@latest
go get github.com/golang-migrate/migrate/v4@latest

# 整理依賴
go mod tidy
```

---

## 📝 常用命令

### 遷移管理

```bash
# 升級一個版本
migrate -path migrations -database "$DATABASE_URL" up 1

# 降級一個版本
migrate -path migrations -database "$DATABASE_URL" down 1

# 強制設置版本 (謹慎使用)
migrate -path migrations -database "$DATABASE_URL" force 1

# 刪除所有數據 (降級到初始狀態)
migrate -path migrations -database "$DATABASE_URL" drop
```

### 數據庫管理

```bash
# 備份數據庫
docker exec -t thenuts-postgres pg_dump -U thenuts thenuts > backup.sql

# 恢復數據庫
docker exec -i thenuts-postgres psql -U thenuts thenuts < backup.sql

# 清空所有表 (保留結構)
psql -h localhost -U thenuts -d thenuts -c "TRUNCATE accounts CASCADE;"

# 重啟容器
docker-compose restart postgres

# 查看日誌
docker-compose logs -f postgres
```

---

## 🧪 測試數據

### 創建測試帳號

```sql
-- 連接數據庫
psql -h localhost -U thenuts -d thenuts

-- 創建測試玩家 (密碼: password123)
INSERT INTO accounts (username, email, password_hash, email_verified)
VALUES ('alice', 'alice@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', true);

INSERT INTO players (account_id, display_name)
SELECT id, 'Alice' FROM accounts WHERE username = 'alice';

INSERT INTO wallets (player_id, balance)
SELECT id, 100000 FROM players WHERE display_name = 'Alice'; -- 1,000.00

-- 查詢測試
SELECT 
    a.username,
    p.display_name,
    w.balance / 100.0 AS balance_usd
FROM accounts a
JOIN players p ON p.account_id = a.id
JOIN wallets w ON w.player_id = p.id;
```

---

## 🔧 故障排除

### 問題 1: 遷移失敗

```bash
# 查看錯誤詳情
migrate -path migrations -database "$DATABASE_URL" version

# 如果顯示 "dirty" 狀態，強制重置
migrate -path migrations -database "$DATABASE_URL" force 0
migrate -path migrations -database "$DATABASE_URL" up
```

### 問題 2: 連接被拒絕

```bash
# 檢查 PostgreSQL 是否運行
docker ps | grep postgres

# 檢查端口是否開放
nc -zv localhost 5432

# 查看容器日誌
docker logs thenuts-postgres
```

### 問題 3: 密碼錯誤

```bash
# 重置 PostgreSQL 密碼
docker-compose down
docker volume rm thenuts_postgres_data
docker-compose up -d
```

---

## 📊 使用 pgAdmin

1. 打開瀏覽器: `http://localhost:5050`
2. 登入: 
   - Email: `admin@thenuts.com`
   - Password: `admin`
3. 添加服務器:
   - Host: `postgres` (Docker 網絡內) 或 `host.docker.internal` (Mac/Windows)
   - Port: `5432`
   - Database: `thenuts`
   - Username: `thenuts`
   - Password: `devpassword`

---

## 🚀 下一步

1. ✅ 數據庫已啟動並遷移完成
2. ⏳ 實作 Repository 層
3. ⏳ 編寫單元測試
4. ⏳ 整合到現有系統

參考完整文檔: `docs/PERSISTENCE_LAYER_DESIGN.md`

---

**更新日期**: 2026-01-26  
**預計完成時間**: 5-10 分鐘
