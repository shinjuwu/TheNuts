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
	authService *AuthService
	logger      *zap.Logger
	ticketTTL   time.Duration // 票券有效期（預設 30 秒）
}

// NewHandler 創建認證 Handler
func NewHandler(jwtService *JWTService, ticketStore TicketStore, authService *AuthService, logger *zap.Logger) *Handler {
	return &Handler{
		jwtService:  jwtService,
		ticketStore: ticketStore,
		authService: authService,
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
	Token       string `json:"token"`
	PlayerID    string `json:"player_id"`
	AccountID   string `json:"account_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse 注册回应
type RegisterResponse struct {
	AccountID string `json:"account_id"`
	PlayerID  string `json:"player_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
}

// ErrorResponse 错误回应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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

// HandleLogin 處理登入請求（生產環境版本）
// 使用真實的用户数据库验证和 bcrypt 密码验证
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// 驗證輸入
	if req.Username == "" || req.Password == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Username and password are required")
		return
	}

	// 获取客户端 IP 地址
	ipAddress := getClientIP(r)

	// 使用 AuthService 进行身份验证
	account, player, err := h.authService.Authenticate(r.Context(), req.Username, req.Password, ipAddress)
	if err != nil {
		// 根据错误类型返回不同的状态码
		switch err {
		case ErrInvalidCredentials:
			h.writeErrorResponse(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
		case ErrAccountLocked:
			h.writeErrorResponse(w, http.StatusForbidden, "account_locked", "Account is temporarily locked due to too many failed login attempts")
		case ErrAccountSuspended:
			h.writeErrorResponse(w, http.StatusForbidden, "account_suspended", "Account has been suspended")
		case ErrAccountBanned:
			h.writeErrorResponse(w, http.StatusForbidden, "account_banned", "Account has been banned")
		default:
			h.logger.Error("authentication failed", zap.Error(err))
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	// 生成 JWT Token（有效期 24 小時）
	token, err := h.jwtService.GenerateToken(player.ID.String(), account.Username, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to generate token")
		return
	}

	// 返回 Token 和用户信息
	resp := LoginResponse{
		Token:       token,
		PlayerID:    player.ID.String(),
		AccountID:   account.ID.String(),
		Username:    account.Username,
		DisplayName: player.DisplayName,
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

// HandleRegister 处理注册请求
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// 验证输入
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Username, email and password are required")
		return
	}

	// 注册用户
	account, player, err := h.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch err {
		case ErrUsernameExists:
			h.writeErrorResponse(w, http.StatusConflict, "username_exists", "Username already exists")
		case ErrEmailExists:
			h.writeErrorResponse(w, http.StatusConflict, "email_exists", "Email already exists")
		default:
			h.logger.Error("registration failed", zap.Error(err))
			h.writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		}
		return
	}

	// 返回注册结果
	resp := RegisterResponse{
		AccountID: account.ID.String(),
		PlayerID:  player.ID.String(),
		Username:  account.Username,
		Message:   "Registration successful. Please login.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// writeErrorResponse 写入错误响应
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// getClientIP 获取客户端 IP 地址
func getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 获取（如果使用了代理）
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// 尝试从 X-Real-IP 获取
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// 直接从 RemoteAddr 获取
	return r.RemoteAddr
}
