# TheNuts 認證系統文檔

## 概述

TheNuts 使用**票券機制 (Ticket-Based Authentication)** 來保護 WebSocket 連線。這是業界推薦的最佳實踐，結合了 JWT 認證和一次性票券的優勢。

## 為什麼需要票券機制？

### 問題：WebSocket 無法使用標準 HTTP Headers

WebSocket 在瀏覽器中建立連線時，無法自訂 HTTP Headers（例如 `Authorization: Bearer <token>`）。這導致以下問題：

1. **不能直接在 URL 中放 JWT Token**：URL 會被記錄在伺服器日誌、瀏覽器歷史中，存在洩漏風險
2. **Cookie 方式有 CSRF 風險**：需要額外的 CSRF 保護機制
3. **Query String 中的 Token 可能被重放攻擊**：如果 Token 長效，一旦洩漏就會造成安全問題

### 解決方案：票券機制

票券機制的核心思想是：**不要將長效憑證暴露在 URL 中，而是先換取一個短效、一次性的票券**。

## 認證流程

```
┌─────────┐                    ┌─────────┐                    ┌─────────┐
│ 客戶端  │                    │ HTTP API│                    │WebSocket│
└────┬────┘                    └────┬────┘                    └────┬────┘
     │                              │                              │
     │ 1. POST /api/auth/login      │                              │
     │    {username, password}      │                              │
     ├─────────────────────────────>│                              │
     │                              │                              │
     │ 2. JWT Token (24小時有效)   │                              │
     │<─────────────────────────────┤                              │
     │                              │                              │
     │ 3. POST /api/auth/ticket     │                              │
     │    Authorization: Bearer JWT │                              │
     ├─────────────────────────────>│                              │
     │                              │                              │
     │ 4. Ticket (30秒有效，一次性) │                              │
     │<─────────────────────────────┤                              │
     │                              │                              │
     │ 5. WS /ws?ticket=xxx         │                              │
     ├──────────────────────────────────────────────────────────>│
     │                              │                              │
     │                              │      驗證並銷毀 Ticket        │
     │                              │                              │
     │ 6. WebSocket 連線建立        │                              │
     │<──────────────────────────────────────────────────────────┤
     │                              │                              │
```

### 步驟說明

#### 步驟 1: 登入獲取 JWT Token

**請求：**
```bash
POST /api/auth/login
Content-Type: application/json

{
  "username": "alice",
  "password": "password123"
}
```

**回應：**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "player_id": "player_alice",
  "username": "alice"
}
```

- JWT Token 有效期為 **24 小時**
- Token 應該安全地儲存在記憶體中（不要放在 localStorage）
- Token 用於後續的 HTTP API 請求

#### 步驟 2: 換取 WebSocket 票券

**請求：**
```bash
POST /api/auth/ticket
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{}
```

**回應：**
```json
{
  "ticket": "a1b2c3d4e5f6...",
  "expires_in": 30,
  "ws_url": "ws://localhost:8080/ws?ticket=a1b2c3d4e5f6..."
}
```

- Ticket 有效期為 **30 秒**（可配置）
- Ticket 是**一次性**的，驗證後立即銷毀
- 客戶端應該在獲取 Ticket 後立即建立 WebSocket 連線

#### 步驟 3: 建立 WebSocket 連線

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?ticket=a1b2c3d4e5f6...');

ws.onopen = function() {
  console.log('連線成功');
};
```

- 伺服器驗證 Ticket 並取得關聯的 Player ID
- 驗證成功後，Ticket 立即被銷毀（防止重放攻擊）
- 如果 Ticket 過期或無效，連線會被拒絕

## 安全特性

### 1. 防止 Token 洩漏
- JWT Token 永不出現在 URL 中
- Ticket 即使洩漏也只有 30 秒有效期

### 2. 防止重放攻擊
- Ticket 驗證成功後立即銷毀
- 即使攻擊者攔截到 Ticket，也無法重複使用

### 3. 時間限制
- JWT Token: 24 小時（可配置）
- Ticket: 30 秒（可配置）

### 4. 職責分離
- **JWT**: 用於 HTTP API 認證（長效）
- **Ticket**: 用於 WebSocket 連線（短效、一次性）

## 配置

在 `config.yaml` 中配置認證參數：

```yaml
auth:
  jwt_secret: "your-secret-key-change-in-production"  # ⚠️ 生產環境必須更換
  ticket_ttl_seconds: 30  # 票券有效期（秒）
```

⚠️ **重要安全提示**：
- `jwt_secret` 必須使用高熵值的隨機字串（至少 32 字元）
- 不要將 Secret 提交到版本控制系統
- 生產環境應該從環境變數讀取 Secret

## API 參考

### POST /api/auth/login

登入並獲取 JWT Token。

**請求體：**
```json
{
  "username": "string",  // 必填
  "password": "string"   // 必填
}
```

**回應 200 OK：**
```json
{
  "token": "string",     // JWT Token
  "player_id": "string", // 玩家 ID
  "username": "string"   // 使用者名稱
}
```

**錯誤回應：**
- `400 Bad Request`: 缺少必填欄位或格式錯誤
- `401 Unauthorized`: 帳號密碼錯誤（開發階段任何密碼都接受）
- `500 Internal Server Error`: 伺服器錯誤

---

### POST /api/auth/ticket

獲取 WebSocket 連線票券（需要 JWT 認證）。

**請求 Headers：**
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**請求體：**
```json
{
  "table_id": "string"  // 可選：指定要連接的桌子 ID
}
```

**回應 200 OK：**
```json
{
  "ticket": "string",    // 一次性票券
  "expires_in": 30,      // 有效期（秒）
  "ws_url": "string"     // 完整的 WebSocket URL
}
```

**錯誤回應：**
- `401 Unauthorized`: 缺少 Token 或 Token 無效
- `500 Internal Server Error`: 伺服器錯誤

---

### GET /ws?ticket=<TICKET>

建立 WebSocket 連線（需要有效票券）。

**Query Parameters：**
- `ticket`: 從 `/api/auth/ticket` 獲取的票券

**錯誤回應：**
- `400 Bad Request`: 缺少 ticket 參數
- `401 Unauthorized`: Ticket 無效或已過期

## 客戶端範例

### JavaScript (瀏覽器)

```javascript
class TheNutsClient {
  constructor(apiUrl) {
    this.apiUrl = apiUrl;
    this.jwtToken = null;
    this.ws = null;
  }

  // 登入
  async login(username, password) {
    const response = await fetch(`${this.apiUrl}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    
    if (!response.ok) {
      throw new Error('Login failed');
    }
    
    const data = await response.json();
    this.jwtToken = data.token;
    return data;
  }

  // 獲取票券
  async getTicket() {
    const response = await fetch(`${this.apiUrl}/api/auth/ticket`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.jwtToken}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({})
    });
    
    if (!response.ok) {
      throw new Error('Failed to get ticket');
    }
    
    return await response.json();
  }

  // 連接 WebSocket
  async connect() {
    const ticketData = await this.getTicket();
    
    this.ws = new WebSocket(ticketData.ws_url);
    
    this.ws.onopen = () => {
      console.log('Connected');
    };
    
    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log('Received:', data);
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
    
    this.ws.onclose = () => {
      console.log('Disconnected');
    };
  }

  // 發送訊息
  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }
}

// 使用範例
const client = new TheNutsClient('http://localhost:8080');

await client.login('alice', 'password123');
await client.connect();
client.send({ action: 'join_table', table_id: 'table_001' });
```

### Python

```python
import requests
import websocket
import json

class TheNutsClient:
    def __init__(self, api_url):
        self.api_url = api_url
        self.jwt_token = None
        self.ws = None
    
    def login(self, username, password):
        """登入並獲取 JWT Token"""
        response = requests.post(
            f'{self.api_url}/api/auth/login',
            json={'username': username, 'password': password}
        )
        response.raise_for_status()
        data = response.json()
        self.jwt_token = data['token']
        return data
    
    def get_ticket(self):
        """獲取 WebSocket 票券"""
        response = requests.post(
            f'{self.api_url}/api/auth/ticket',
            headers={'Authorization': f'Bearer {self.jwt_token}'},
            json={}
        )
        response.raise_for_status()
        return response.json()
    
    def connect(self):
        """建立 WebSocket 連線"""
        ticket_data = self.get_ticket()
        
        self.ws = websocket.WebSocketApp(
            ticket_data['ws_url'],
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        
        # 在新執行緒中執行
        import threading
        wst = threading.Thread(target=self.ws.run_forever)
        wst.daemon = True
        wst.start()
    
    def send(self, data):
        """發送訊息"""
        if self.ws:
            self.ws.send(json.dumps(data))
    
    def on_open(self, ws):
        print("Connected")
    
    def on_message(self, ws, message):
        data = json.loads(message)
        print(f"Received: {data}")
    
    def on_error(self, ws, error):
        print(f"Error: {error}")
    
    def on_close(self, ws, close_status_code, close_msg):
        print("Disconnected")

# 使用範例
client = TheNutsClient('http://localhost:8080')
client.login('alice', 'password123')
client.connect()
client.send({'action': 'join_table', 'table_id': 'table_001'})
```

## 測試

### 使用測試客戶端

專案提供了一個互動式的測試客戶端 `test-client.html`：

1. 啟動伺服器：
   ```bash
   ./game-server
   ```

2. 在瀏覽器開啟：
   ```
   http://localhost:8080/test-client.html
   ```

3. 按照介面上的步驟測試：
   - 步驟 1: 登入
   - 步驟 2: 獲取票券
   - 步驟 3: 建立 WebSocket 連線

### 使用 curl 測試

```bash
# 1. 登入
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}' \
  | jq -r '.token')

echo "JWT Token: $TOKEN"

# 2. 獲取票券
TICKET=$(curl -X POST http://localhost:8080/api/auth/ticket \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' \
  | jq -r '.ticket')

echo "Ticket: $TICKET"

# 3. 使用 websocat 建立 WebSocket 連線
websocat "ws://localhost:8080/ws?ticket=$TICKET"
```

## 生產環境部署

### 1. 使用環境變數儲存 Secret

```bash
export JWT_SECRET="your-256-bit-secret-key-here"
```

修改配置載入邏輯：
```go
func LoadConfig(path string) (*Config, error) {
    // ... 載入 YAML 配置 ...
    
    // 從環境變數覆蓋敏感配置
    if secret := os.Getenv("JWT_SECRET"); secret != "" {
        cfg.Auth.JWTSecret = secret
    }
    
    return &cfg, nil
}
```

### 2. 使用 Redis 儲存 Ticket

目前使用記憶體儲存（`MemoryTicketStore`），生產環境建議使用 Redis：

```go
// internal/auth/redis_ticket_store.go
type RedisTicketStore struct {
    client *redis.Client
}

func NewRedisTicketStore(addr string) (*RedisTicketStore, error) {
    client := redis.NewClient(&redis.Options{
        Addr: addr,
    })
    
    return &RedisTicketStore{client: client}, nil
}

func (s *RedisTicketStore) Generate(ctx context.Context, playerID string, ttl time.Duration) (string, error) {
    ticket, err := generateRandomTicket(32)
    if err != nil {
        return "", err
    }
    
    // 儲存到 Redis with TTL
    err = s.client.Set(ctx, "ticket:"+ticket, playerID, ttl).Err()
    return ticket, err
}

func (s *RedisTicketStore) Validate(ctx context.Context, ticket string) (string, error) {
    // 使用 GETDEL 原子地獲取並刪除
    playerID, err := s.client.GetDel(ctx, "ticket:"+ticket).Result()
    if err == redis.Nil {
        return "", fmt.Errorf("invalid ticket")
    }
    return playerID, err
}
```

### 3. HTTPS/WSS

生產環境必須使用 HTTPS 和 WSS（WebSocket Secure）：

```go
srv := &http.Server{
    Addr:    ":443",
    Handler: mux,
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
}

srv.ListenAndServeTLS("cert.pem", "key.pem")
```

### 4. CORS 設定

限制允許的來源：

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        allowedOrigins := []string{
            "https://yourdomain.com",
            "https://app.yourdomain.com",
        }
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                return true
            }
        }
        return false
    },
}
```

## 常見問題

### Q: 為什麼不直接使用 JWT 作為 WebSocket Query Parameter？

A: JWT Token 通常有較長的有效期（如 24 小時），如果直接放在 URL 中：
- URL 會被記錄在伺服器日誌、反向代理日誌中
- 可能被記錄在瀏覽器歷史中
- 如果洩漏，攻擊者可以在 Token 過期前一直使用

Ticket 機制解決了這個問題：
- Ticket 只有 30 秒有效期
- Ticket 是一次性的，用完即銷毀
- 即使洩漏，影響範圍很小

### Q: 如果 Ticket 在 30 秒內沒有使用會怎樣？

A: Ticket 會自動過期並被清理。客戶端需要重新呼叫 `/api/auth/ticket` 獲取新的 Ticket。

### Q: WebSocket 斷線後需要重新登入嗎？

A: 不需要。只要 JWT Token 還沒過期，可以直接獲取新的 Ticket 並重新連線：

```javascript
// 斷線重連
ws.onclose = async () => {
  console.log('Disconnected, reconnecting...');
  const ticketData = await getTicket(); // 使用現有的 JWT Token
  connect(ticketData.ws_url);
};
```

### Q: 如何處理 JWT Token 過期？

A: JWT Token 過期後，需要重新登入：

```javascript
async function getTicket() {
  try {
    const response = await fetch('/api/auth/ticket', {
      headers: { 'Authorization': `Bearer ${jwtToken}` }
    });
    
    if (response.status === 401) {
      // Token 過期，重新登入
      await login(username, password);
      return getTicket(); // 重試
    }
    
    return await response.json();
  } catch (error) {
    console.error('Failed to get ticket:', error);
  }
}
```

### Q: 開發階段的登入邏輯是如何運作的？

A: 目前為簡化開發，`/api/auth/login` 接受任何非空的使用者名稱和密碼。**生產環境必須實作真實的身份驗證**：

```go
// 生產環境應該這樣實作：
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
    // 1. 查詢數據庫取得使用者
    user, err := h.userRepo.FindByUsername(req.Username)
    if err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    
    // 2. 驗證密碼（使用 bcrypt）
    if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(req.Password)); err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    
    // 3. 檢查帳號是否被封禁
    if user.IsBanned {
        http.Error(w, "account banned", http.StatusForbidden)
        return
    }
    
    // 4. 生成 Token
    token, _ := h.jwtService.GenerateToken(user.ID, user.Username, 24*time.Hour)
    // ...
}
```

## 下一步

1. **實作真實的使用者認證**：連接數據庫、密碼雜湊、帳號管理
2. **使用 Redis 儲存 Ticket**：支援分散式部署
3. **實作 Token 刷新機制**：避免使用者頻繁登入
4. **新增速率限制**：防止暴力破解和 DDoS 攻擊
5. **實作 2FA（雙因素認證）**：提升帳號安全性

## 參考資料

- [RFC 6749 - OAuth 2.0](https://tools.ietf.org/html/rfc6749)
- [RFC 7519 - JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)
- [WebSocket 安全最佳實踐](https://owasp.org/www-community/vulnerabilities/WebSocket_security)
- [OWASP 認證備忘錄](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
