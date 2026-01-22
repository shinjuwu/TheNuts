package ws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 生產環境應該檢查 Origin Header
		// 建議使用白名單機制，例如：
		// origin := r.Header.Get("Origin")
		// return origin == "https://yourdomain.com"
		return true
	},
}

type Handler struct {
	Hub          *Hub
	TableManager *game.TableManager
	TicketStore  auth.TicketStore
	Logger       *zap.Logger
}

func NewHandler(hub *Hub, tableMgr *game.TableManager, ticketStore auth.TicketStore, logger *zap.Logger) *Handler {
	return &Handler{
		Hub:          hub,
		TableManager: tableMgr,
		TicketStore:  ticketStore,
		Logger:       logger,
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	// 創建客戶端並註冊
	client := NewClient(h.Hub, h.TableManager, conn, playerID, h.Logger)
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
