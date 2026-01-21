package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shinjuwu/TheNuts/internal/game"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	Hub          *Hub
	TableManager *game.TableManager
	Conn         *websocket.Conn
	PlayerID     string
	send         chan interface{}
	logger       *zap.Logger
}

func NewClient(hub *Hub, tableMgr *game.TableManager, conn *websocket.Conn, playerID string, logger *zap.Logger) *Client {
	return &Client{
		Hub:          hub,
		TableManager: tableMgr,
		Conn:         conn,
		PlayerID:     playerID,
		send:         make(chan interface{}, 256),
		logger:       logger,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("error reading message", zap.Error(err))
			}
			break
		}

		var req Request
		if err := json.Unmarshal(message, &req); err != nil {
			c.logger.Warn("invalid message format", zap.Error(err))
			continue
		}

		// 路由到對應的桌子
		if req.TableID != "" {
			table := c.TableManager.GetTable(req.TableID)
			if table != nil {
				table.ActionCh <- req
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
