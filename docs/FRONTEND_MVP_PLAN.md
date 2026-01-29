# Vue 3 + TypeScript 前端 MVP 規劃

## 專案結構

```
TheNuts/
└── web/
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── index.html
    ├── src/
    │   ├── main.ts
    │   ├── App.vue
    │   ├── router/
    │   │   └── index.ts              # vue-router（Login / Lobby / Table）
    │   ├── stores/
    │   │   ├── auth.ts               # Pinia — JWT token、player info
    │   │   ├── game.ts               # Pinia — 牌桌狀態、玩家、公牌、底池
    │   │   └── wallet.ts             # Pinia — 餘額、買入/兌現狀態
    │   ├── composables/
    │   │   └── useWebSocket.ts       # WS 連線管理、自動重連、訊息分發
    │   ├── services/
    │   │   └── api.ts                # HTTP client（login, register, ticket）
    │   ├── types/
    │   │   └── index.ts              # 所有 TS 型別定義（WS 訊息、玩家、牌等）
    │   ├── views/
    │   │   ├── LoginView.vue         # 登入 / 註冊
    │   │   ├── LobbyView.vue         # 大廳（選桌 / 買入）
    │   │   └── TableView.vue         # 牌桌主畫面
    │   └── components/
    │       ├── table/
    │       │   ├── PokerTable.vue     # 牌桌佈局（橢圓 + 9 座位）
    │       │   ├── Seat.vue           # 單一座位（玩家名、籌碼、狀態）
    │       │   ├── CommunityCards.vue # 公牌區
    │       │   ├── PotDisplay.vue     # 底池顯示
    │       │   └── DealerButton.vue   # 莊家按鈕
    │       ├── player/
    │       │   ├── HandCards.vue      # 自己的手牌
    │       │   └── ActionPanel.vue    # 操作按鈕（Fold/Check/Call/Bet/Raise）
    │       ├── card/
    │       │   └── PlayingCard.vue    # 單張撲克牌
    │       └── common/
    │           ├── BuyInModal.vue     # 買入對話框
    │           ├── CashOutModal.vue   # 兌現對話框
    │           └── Toast.vue          # 錯誤 / 通知提示
    └── public/
        └── cards/                     # 撲克牌圖片素材（或用 CSS 繪製）
```

## 技術選型

| 項目 | 選擇 | 理由 |
|------|------|------|
| 框架 | Vue 3 + Composition API | 響應式天然適合 WS 即時更新 |
| 語言 | TypeScript | WS 訊息型別安全 |
| 建構工具 | Vite | 快、Vue 官方推薦 |
| 狀態管理 | Pinia | Vue 3 官方推薦，比 Vuex 簡潔 |
| 路由 | vue-router 4 | 三個頁面：Login / Lobby / Table |
| UI 框架 | 不用（手寫 CSS） | MVP 夠用，避免多一層依賴 |

## 實作步驟與 Checklist

### Phase 1：專案初始化

- [ ] `npm create vite@latest web -- --template vue-ts`
- [ ] 安裝依賴：`pinia`, `vue-router`
- [ ] 設定 `vite.config.ts`（proxy `/api` 和 `/ws` 到 `localhost:8080`）
- [ ] 設定 `tsconfig.json` path alias（`@/` → `src/`）
- [ ] 根目錄 `.gitignore` 加上 `web/node_modules`、`web/dist`

### Phase 2：型別定義 + API 層

- [ ] `types/index.ts` — 定義所有型別：
  - `Card`（`"AS"`, `"KH"` 格式，rank + suit）
  - `Player`（id, seatIdx, chips, currentBet, status, hasActed）
  - `TableState`（state, players, communityCards, dealerPos, currentPos, minBet, potTotal）
  - WS 請求：`WSRequest`（action, table_id, amount, seat_no, game_action）
  - WS 回應：`WSResponse`（type, payload, timestamp, trace_id）
  - 各事件 payload 型別（HandStart, HoleCards, BlindPosted, YourTurn, PlayerAction, CommunityCards, ShowdownResult 等）
- [ ] `services/api.ts` — HTTP 封裝：
  - `login(username, password)` → `POST /api/auth/login`
  - `register(username, email, password)` → `POST /api/auth/register`
  - `getTicket(token)` → `POST /api/auth/ticket`

### Phase 3：WebSocket 管理

- [ ] `composables/useWebSocket.ts`：
  - 取得 ticket → 建立 WS 連線
  - 訊息分發：依 `response.type` 呼叫對應 store action
  - 斷線自動重連（指數退避，最多 30 秒）
  - `send(action, payload)` 封裝發送
  - 連線狀態暴露為 `ref<'connecting' | 'connected' | 'disconnected'>`

### Phase 4：狀態管理（Pinia Stores）

- [ ] `stores/auth.ts`：
  - state：`token`, `playerID`, `username`, `isLoggedIn`
  - actions：`login()`, `register()`, `logout()`
  - token 存 `localStorage`，啟動時自動恢復
- [ ] `stores/game.ts`：
  - state：`tableState`, `myCards`, `myTurn`, `validActions`, `amountToCall`, `timeRemaining`
  - actions：對應每個 WS 事件更新 state
    - `onHandStart()`, `onHoleCards()`, `onBlindsPosted()`
    - `onYourTurn()`, `onPlayerAction()`, `onCommunityCards()`
    - `onShowdownResult()`, `onWinByFold()`, `onHandEnd()`
    - `onTableState()`（全量快照覆蓋）
    - `onActionTimeout()`
- [ ] `stores/wallet.ts`：
  - state：`walletBalance`, `lockedBalance`, `currentChips`
  - actions：`onBalanceInfo()`, `onBuyInSuccess()`, `onCashOutSuccess()`

### Phase 5：頁面與路由

- [ ] `router/index.ts`：
  - `/login` → LoginView（未登入預設頁）
  - `/lobby` → LobbyView（需登入）
  - `/table/:id` → TableView（需登入 + 已買入）
  - Navigation guard：未登入導向 `/login`
- [ ] `views/LoginView.vue`：
  - 登入 / 註冊 tab 切換
  - 表單驗證（username 必填、password 最少 6 字元）
  - 錯誤顯示（帳號已存在、密碼錯誤、帳號鎖定等）
- [ ] `views/LobbyView.vue`：
  - 顯示餘額
  - 選擇牌桌（MVP 先硬編碼一張桌）
  - 買入金額輸入 → 進入牌桌
- [ ] `views/TableView.vue`：
  - 組合所有 table components
  - 進入時 `JOIN_TABLE` + `SIT_DOWN`
  - 離開時 `LEAVE_TABLE`

### Phase 6：牌桌 UI 元件

- [ ] `PokerTable.vue`：
  - 橢圓形牌桌（CSS 繪製）
  - 9 個座位固定位置排列
  - 中央顯示公牌 + 底池
- [ ] `Seat.vue`：
  - 玩家名稱、籌碼數
  - 狀態指示（playing / folded / all-in / sitting out）
  - 當前行動者高亮
  - 空座位顯示「空」
- [ ] `PlayingCard.vue`：
  - 牌面渲染（rank + suit，用 CSS 或 Unicode 符號 ♠♥♦♣）
  - 背面狀態（對手的牌）
- [ ] `CommunityCards.vue`：依 street 顯示 0/3/4/5 張
- [ ] `PotDisplay.vue`：顯示底池總額
- [ ] `DealerButton.vue`：標示在對應座位旁

### Phase 7：操作面板

- [ ] `ActionPanel.vue`：
  - 根據 `YOUR_TURN` 的 `valid_actions` 動態顯示按鈕
  - Fold / Check / Call（顯示金額）/ Bet / Raise
  - Bet/Raise 帶金額滑桿或輸入框（最小值 = minBet，最大值 = 自己籌碼）
  - All-In 按鈕
  - 倒數計時條（`time_remaining`）
  - 非自己回合時 disable
- [ ] `HandCards.vue`：顯示自己的兩張手牌（`HOLE_CARDS` 事件）

### Phase 8：買入 / 兌現 / 通知

- [ ] `BuyInModal.vue`：金額輸入 + 確認
- [ ] `CashOutModal.vue`：顯示 profit/loss + 確認
- [ ] `Toast.vue`：錯誤訊息 + 遊戲事件通知（XX 玩家 fold、超時等）

### Phase 9：收尾

- [ ] Go 靜態服務指向 `web/dist`（修改 `main.go` 的 `http.FileServer`）
- [ ] 測試完整流程：註冊 → 登入 → 買入 → 入座 → 玩一手牌 → 兌現
- [ ] 處理邊界情況：斷線重連、餘額不足、座位已滿

## 後端對接重點

### HTTP API

| 端點 | 方法 | 用途 |
|------|------|------|
| `/api/auth/register` | POST | `{ username, email, password }` → 註冊 |
| `/api/auth/login` | POST | `{ username, password }` → 返回 JWT |
| `/api/auth/ticket` | POST | Header: `Bearer <JWT>` → 返回一次性 ticket |
| `/ws?ticket=xxx` | WS | WebSocket 連線 |

### WS 客戶端 → 伺服器（8 種 action）

`BUY_IN`, `CASH_OUT`, `JOIN_TABLE`, `LEAVE_TABLE`, `SIT_DOWN`, `STAND_UP`, `GAME_ACTION`, `GET_BALANCE`

### WS 伺服器 → 客戶端（事件）

`HAND_START`, `HOLE_CARDS`, `BLINDS_POSTED`, `YOUR_TURN`, `PLAYER_ACTION`, `COMMUNITY_CARDS`, `SHOWDOWN_RESULT`, `WIN_BY_FOLD`, `HAND_END`, `ACTION_TIMEOUT`, `TABLE_STATE`

### 牌面格式

2 字元字串：`rank` + `suit`
- Rank: `A, 2, 3, 4, 5, 6, 7, 8, 9, T, J, Q, K`
- Suit: `H`(hearts), `D`(diamonds), `C`(clubs), `S`(spades)
- 例：`"AS"` = 黑桃 A，`"TH"` = 紅心 10

## 驗證方式

```bash
# 1. 建構前端
cd web && npm run build

# 2. 啟動後端（會同時 serve 前端靜態檔）
cd .. && go run cmd/game-server/main.go

# 3. 開瀏覽器 http://localhost:8080
# 4. 註冊 → 登入 → 進入大廳 → 買入 → 入座 → 等第二位玩家加入 → 自動開局
```
