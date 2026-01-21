package ws

import (
	"sync"

	"go.uber.org/zap"
)

type Hub struct {
	clients    map[string]*Client // key: PlayerID
	register   chan *Client
	unregister chan *Client
	broadcast  chan interface{}
	mu         sync.RWMutex
	logger     *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan interface{}),
		logger:     logger,
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
				h.logger.Info("player disconnected", zap.String("player_id", client.PlayerID))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 如果發送失敗（通道滿），主動斷開或略過
					// 這裡暫時略過
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
		client.send <- message
	}
}
