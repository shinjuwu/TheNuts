package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/shinjuwu/TheNuts/internal/game"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // B2B 開發階段先允許所有來源
	},
}

type Handler struct {
	Hub          *Hub
	TableManager *game.TableManager
	Logger       *zap.Logger
}

func NewHandler(hub *Hub, tableMgr *game.TableManager, logger *zap.Logger) *Handler {
	return &Handler{
		Hub:          hub,
		TableManager: tableMgr,
		Logger:       logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		http.Error(w, "player_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(h.Hub, h.TableManager, conn, playerID, h.Logger)
	h.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
