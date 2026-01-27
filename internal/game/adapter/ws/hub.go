package ws

import (
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Hub struct {
	clients        map[string]*Client
	sessionManager *SessionManager
	register       chan *Client
	unregister     chan *Client
	broadcast      chan interface{}
	mu             sync.RWMutex
	logger         *zap.Logger
}

func NewHub(sessionManager *SessionManager, logger *zap.Logger) *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		sessionManager: sessionManager,
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan interface{}),
		logger:         logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.PlayerID] = client
			h.mu.Unlock()
			h.logger.Info("player connected", zap.String("player_id", client.PlayerID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.PlayerID]; ok {
				delete(h.clients, client.PlayerID)
				close(client.send)
				
				playerID, err := uuid.Parse(client.PlayerID)
				if err == nil {
					go h.sessionManager.HandleDisconnect(playerID)
				}
				
				h.logger.Info("player disconnected", zap.String("player_id", client.PlayerID))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(message interface{}) {
	h.broadcast <- message
}

func (h *Hub) SendToPlayer(playerID string, message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if client, ok := h.clients[playerID]; ok {
		select {
		case client.send <- message:
		default:
			h.logger.Warn("failed to send to player", zap.String("player_id", playerID))
		}
	}
}

func (h *Hub) BroadcastToTable(tableID string, message interface{}) {
	if h.sessionManager != nil {
		h.sessionManager.BroadcastToTable(tableID, message)
	}
}

func (h *Hub) GetConnectedPlayerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
