# 認證系統快速開始指南

## 🚀 5 分鐘快速測試

### 1. 啟動伺服器

```bash
# Windows
.\game-server.exe

# Linux/Mac
./game-server
```

伺服器會在 `http://localhost:8080` 啟動。

### 2. 開啟測試客戶端

在瀏覽器開啟：
```
http://localhost:8080/test-client.html
```

### 3. 按照步驟測試

1. **登入**：輸入任意使用者名稱和密碼（開發階段接受任何值）
2. **獲取票券**：點擊「獲取票券」按鈕
3. **建立連線**：點擊「連線」按鈕

你應該會在日誌區看到「WebSocket 連線成功」的訊息。

## 🔐 認證流程簡介

```
登入 → 獲取 JWT Token → 換取 Ticket → 建立 WebSocket 連線
```

- **JWT Token**：有效期 24 小時，用於 HTTP API 認證
- **Ticket**：有效期 30 秒，一次性使用，用於 WebSocket 連線

## 📡 API 端點

### 登入
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"password123"}'
```

回應：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "player_id": "player_alice",
  "username": "alice"
}
```

### 獲取票券
```bash
curl -X POST http://localhost:8080/api/auth/ticket \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{}'
```

回應：
```json
{
  "ticket": "a1b2c3d4e5f6...",
  "expires_in": 30,
  "ws_url": "ws://localhost:8080/ws?ticket=a1b2c3d4e5f6..."
}
```

### 建立 WebSocket 連線
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?ticket=<YOUR_TICKET>');
```

## 💻 程式碼範例

### JavaScript (完整範例)

```javascript
// 1. 登入
const loginResponse = await fetch('http://localhost:8080/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ 
    username: 'alice', 
    password: 'password123' 
  })
});
const { token } = await loginResponse.json();

// 2. 獲取票券
const ticketResponse = await fetch('http://localhost:8080/api/auth/ticket', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({})
});
const { ws_url } = await ticketResponse.json();

// 3. 建立 WebSocket 連線
const ws = new WebSocket(ws_url);
ws.onopen = () => console.log('Connected!');
ws.onmessage = (event) => console.log('Received:', event.data);
```

### Python (完整範例)

```python
import requests
import websocket

# 1. 登入
login_resp = requests.post(
    'http://localhost:8080/api/auth/login',
    json={'username': 'alice', 'password': 'password123'}
)
token = login_resp.json()['token']

# 2. 獲取票券
ticket_resp = requests.post(
    'http://localhost:8080/api/auth/ticket',
    headers={'Authorization': f'Bearer {token}'},
    json={}
)
ws_url = ticket_resp.json()['ws_url']

# 3. 建立 WebSocket 連線
ws = websocket.WebSocket()
ws.connect(ws_url)
print("Connected!")
```

## 🛡️ 安全特性

- ✅ **JWT Token 不會出現在 URL 中**：防止 Token 洩漏
- ✅ **Ticket 短效（30 秒）**：即使洩漏影響也很小
- ✅ **Ticket 一次性使用**：驗證後立即銷毀，防止重放攻擊
- ✅ **密碼學安全的隨機 Ticket**：使用 `crypto/rand` 生成

## ⚙️ 配置

在 `config.yaml` 中修改設定：

```yaml
auth:
  jwt_secret: "your-secret-key-change-in-production"  # ⚠️ 生產環境必須更換
  ticket_ttl_seconds: 30  # 票券有效期（秒）
```

## 🚨 生產環境注意事項

在部署到生產環境前，必須：

1. ✅ **更換 JWT Secret**：使用高熵值的隨機字串（至少 32 字元）
2. ✅ **實作真實的使用者認證**：連接數據庫、密碼雜湊（bcrypt）
3. ✅ **使用 HTTPS/WSS**：加密傳輸
4. ✅ **限制 CORS**：只允許信任的來源
5. ✅ **使用 Redis 儲存 Ticket**：支援分散式部署
6. ✅ **實作速率限制**：防止暴力破解

## 📚 完整文檔

詳細的認證系統文檔請參考：
- [AUTHENTICATION.md](docs/AUTHENTICATION.md) - 完整的認證系統文檔
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - 整體架構設計

## 🐛 常見問題

### 連線失敗：invalid ticket

**原因**：Ticket 可能已過期（30 秒）或已被使用（一次性）

**解決**：重新呼叫 `/api/auth/ticket` 獲取新的 Ticket

### 401 Unauthorized

**原因**：JWT Token 無效或已過期

**解決**：重新登入獲取新的 Token

### Ticket 獲取成功但 WebSocket 連線失敗

**原因**：Ticket 在 30 秒內未使用已過期

**解決**：獲取 Ticket 後立即建立 WebSocket 連線（不要延遲）

## 📞 需要幫助？

查看完整文檔或提交 Issue：
- GitHub: https://github.com/shinjuwu/TheNuts
- Email: support@example.com
