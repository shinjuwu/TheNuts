package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Handler 認證相關的 HTTP Handler
type Handler struct {
	jwtService  *JWTService
	ticketStore TicketStore
	logger      *zap.Logger
	ticketTTL   time.Duration // 票券有效期（預設 30 秒）
}

// NewHandler 創建認證 Handler
func NewHandler(jwtService *JWTService, ticketStore TicketStore, logger *zap.Logger) *Handler {
	return &Handler{
		jwtService:  jwtService,
		ticketStore: ticketStore,
		logger:      logger,
		ticketTTL:   30 * time.Second, // 預設 30 秒
	}
}

// SetTicketTTL 設定票券有效期
func (h *Handler) SetTicketTTL(ttl time.Duration) {
	h.ticketTTL = ttl
}

// LoginRequest 登入請求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登入回應
type LoginResponse struct {
	Token    string `json:"token"`
	PlayerID string `json:"player_id"`
	Username string `json:"username"`
}

// TicketRequest 票券請求（需要帶上 JWT Token）
type TicketRequest struct {
	// 可選：指定要連接的桌子 ID
	TableID string `json:"table_id,omitempty"`
}

// TicketResponse 票券回應
type TicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"` // 秒數
	WSUrl     string `json:"ws_url"`
}

// HandleLogin 處理登入請求（開發階段簡化版）
// 生產環境應該驗證密碼、查詢數據庫等
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 驗證輸入
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	// ⚠️ 開發階段：簡化的身份驗證
	// 生產環境應該：
	// 1. 查詢數據庫驗證使用者
	// 2. 使用 bcrypt 驗證密碼雜湊
	// 3. 實作帳號鎖定、速率限制等安全措施

	// 為了演示，接受任何非空的使用者名稱/密碼
	playerID := "player_" + req.Username // 簡化的 ID 生成

	// 生成 JWT Token（有效期 24 小時）
	token, err := h.jwtService.GenerateToken(playerID, req.Username, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("user logged in",
		zap.String("username", req.Username),
		zap.String("player_id", playerID),
	)

	// 返回 Token
	resp := LoginResponse{
		Token:    token,
		PlayerID: playerID,
		Username: req.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetTicket 處理獲取 WebSocket 票券的請求
// 需要先通過 JWT 中介層驗證
func (h *Handler) HandleGetTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 從 context 取得已驗證的玩家 ID
	playerID, ok := GetPlayerIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized: player_id not found in context", http.StatusUnauthorized)
		return
	}

	// 解析請求（可選參數）
	var req TicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 允許空 body
		req = TicketRequest{}
	}

	// 生成一次性票券
	ticket, err := h.ticketStore.Generate(context.Background(), playerID, h.ticketTTL)
	if err != nil {
		h.logger.Error("failed to generate ticket",
			zap.String("player_id", playerID),
			zap.Error(err),
		)
		http.Error(w, "failed to generate ticket", http.StatusInternalServerError)
		return
	}

	h.logger.Info("ticket generated",
		zap.String("player_id", playerID),
		zap.String("ticket", ticket[:8]+"..."), // 只記錄前 8 字元
		zap.Duration("ttl", h.ticketTTL),
	)

	// 構建 WebSocket URL
	scheme := "ws"
	if r.TLS != nil {
		scheme = "wss"
	}
	wsURL := scheme + "://" + r.Host + "/ws?ticket=" + ticket

	// 返回票券
	resp := TicketResponse{
		Ticket:    ticket,
		ExpiresIn: int(h.ticketTTL.Seconds()),
		WSUrl:     wsURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
