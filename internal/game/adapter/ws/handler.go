package ws

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"go.uber.org/zap"
)

type Handler struct {
	Hub            *Hub
	TableManager   *game.TableManager
	SessionManager *SessionManager
	GameService    *service.GameService
	TicketStore    auth.TicketStore
	Logger         *zap.Logger
	upgrader       websocket.Upgrader
}

func NewHandler(
	hub *Hub,
	tableMgr *game.TableManager,
	sessionMgr *SessionManager,
	gameService *service.GameService,
	ticketStore auth.TicketStore,
	logger *zap.Logger,
) *Handler {
	h := &Handler{
		Hub:            hub,
		TableManager:   tableMgr,
		SessionManager: sessionMgr,
		GameService:    gameService,
		TicketStore:    ticketStore,
		Logger:         logger,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 預設允許所有（透過 SetAllowedOrigins 覆蓋）
		},
	}
	return h
}

// SetAllowedOrigins 設定 WebSocket Origin 白名單
// 空清單表示允許所有來源（開發模式）
func (h *Handler) SetAllowedOrigins(origins []string) {
	if len(origins) == 0 {
		h.Logger.Warn("no allowed origins configured, accepting all origins (development mode)")
		return
	}

	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}

	h.Logger.Info("WebSocket origin whitelist configured",
		zap.Strings("allowed_origins", origins),
	)

	h.upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// 無 Origin header（非瀏覽器客戶端）→ 允許
			return true
		}
		if allowed[origin] {
			return true
		}
		h.Logger.Warn("WebSocket connection rejected: origin not allowed",
			zap.String("origin", origin),
			zap.String("remote_addr", r.RemoteAddr),
		)
		return false
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 從 URL Query 取得票券
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "ticket is required", http.StatusBadRequest)
		h.Logger.Warn("websocket connection rejected: missing ticket")
		return
	}

	// 驗證票券並取得玩家 ID
	playerID, err := h.TicketStore.Validate(context.Background(), ticket)
	if err != nil {
		http.Error(w, "invalid ticket: "+err.Error(), http.StatusUnauthorized)
		h.Logger.Warn("websocket connection rejected: invalid ticket",
			zap.String("ticket_prefix", ticket[:min(8, len(ticket))]),
			zap.Error(err),
		)
		return
	}

	h.Logger.Info("ticket validated successfully",
		zap.String("player_id", playerID),
		zap.String("ticket_prefix", ticket[:8]),
	)

	// 升級到 WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	// 解析 player UUID
	playerUUID, err := uuid.Parse(playerID)
	if err != nil {
		h.Logger.Error("invalid player ID format", zap.String("player_id", playerID), zap.Error(err))
		conn.Close()
		return
	}

	// 創建客戶端
	client := NewClient(h.Hub, h.TableManager, conn, playerID, h.Logger)

	// 創建 PlayerSession
	session := NewPlayerSession(playerUUID, playerID, client, h.GameService, h.Logger)
	h.SessionManager.AddSession(session)

	// 註冊客戶端到 Hub
	h.Hub.register <- client

	h.Logger.Info("websocket client connected",
		zap.String("player_id", playerID),
		zap.String("remote_addr", r.RemoteAddr),
	)

	go client.WritePump()
	go client.ReadPump()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
