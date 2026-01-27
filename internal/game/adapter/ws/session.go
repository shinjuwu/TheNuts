package ws

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"go.uber.org/zap"
)

// PlayerSession 代表一个玩家的完整会话状态
type PlayerSession struct {
	// 基本信息
	PlayerID    uuid.UUID
	Username    string
	Client      *Client
	GameService *service.GameService
	Logger      *zap.Logger

	// 游戏状态
	CurrentTableID string
	GameSessionID  uuid.UUID
	Chips          int64
	SeatNo         int
	IsSeated       bool

	// 连接状态
	ConnectedAt    time.Time
	LastActivityAt time.Time
	IsConnected    bool

	// 同步锁
	mu sync.RWMutex
}

// NewPlayerSession 创建新的玩家会话
func NewPlayerSession(
	playerID uuid.UUID,
	username string,
	client *Client,
	gameService *service.GameService,
	logger *zap.Logger,
) *PlayerSession {
	return &PlayerSession{
		PlayerID:       playerID,
		Username:       username,
		Client:         client,
		GameService:    gameService,
		Logger:         logger,
		ConnectedAt:    time.Now(),
		LastActivityAt: time.Now(),
		IsConnected:    true,
	}
}

// UpdateActivity 更新最后活动时间
func (s *PlayerSession) UpdateActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityAt = time.Now()
}

// SetTable 设置玩家当前所在桌子
func (s *PlayerSession) SetTable(tableID string, seatNo int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentTableID = tableID
	s.SeatNo = seatNo
	s.IsSeated = true
}

// LeaveTable 玩家离开桌子
func (s *PlayerSession) LeaveTable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentTableID = ""
	s.SeatNo = -1
	s.IsSeated = false
}

// UpdateChips 更新玩家筹码
func (s *PlayerSession) UpdateChips(chips int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Chips = chips
}

// GetChips 获取当前筹码（线程安全）
func (s *PlayerSession) GetChips() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Chips
}

// GetTableID 获取当前桌子ID（线程安全）
func (s *PlayerSession) GetTableID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentTableID
}

// SetGameSession 设置游戏会话ID
func (s *PlayerSession) SetGameSession(sessionID uuid.UUID, chips int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GameSessionID = sessionID
	s.Chips = chips
}

// Disconnect 断开连接
func (s *PlayerSession) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsConnected = false
}

// SendMessage 发送消息给玩家
func (s *PlayerSession) SendMessage(msg interface{}) {
	if s.Client != nil {
		select {
		case s.Client.send <- msg:
		default:
			s.Logger.Warn("failed to send message: channel full",
				zap.String("player_id", s.PlayerID.String()),
			)
		}
	}
}

// GetSnapshot 获取会话快照（用于日志和调试）
type SessionSnapshot struct {
	PlayerID       string    `json:"player_id"`
	Username       string    `json:"username"`
	CurrentTableID string    `json:"current_table_id,omitempty"`
	GameSessionID  string    `json:"game_session_id,omitempty"`
	Chips          int64     `json:"chips"`
	SeatNo         int       `json:"seat_no"`
	IsSeated       bool      `json:"is_seated"`
	IsConnected    bool      `json:"is_connected"`
	ConnectedAt    time.Time `json:"connected_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

// GetSnapshot 获取会话快照
func (s *PlayerSession) GetSnapshot() SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionSnapshot{
		PlayerID:       s.PlayerID.String(),
		Username:       s.Username,
		CurrentTableID: s.CurrentTableID,
		GameSessionID:  s.GameSessionID.String(),
		Chips:          s.Chips,
		SeatNo:         s.SeatNo,
		IsSeated:       s.IsSeated,
		IsConnected:    s.IsConnected,
		ConnectedAt:    s.ConnectedAt,
		LastActivityAt: s.LastActivityAt,
	}
}

// SessionManager 管理所有玩家会话
type SessionManager struct {
	sessions    map[uuid.UUID]*PlayerSession // playerID -> session
	gameService *service.GameService
	logger      *zap.Logger
	mu          sync.RWMutex

	// 清理配置
	cleanupInterval time.Duration
	sessionTimeout  time.Duration
	stopCh          chan struct{}
}

// NewSessionManager 创建会话管理器
func NewSessionManager(
	gameService *service.GameService,
	logger *zap.Logger,
) *SessionManager {
	sm := &SessionManager{
		sessions:        make(map[uuid.UUID]*PlayerSession),
		gameService:     gameService,
		logger:          logger,
		cleanupInterval: 1 * time.Minute,
		sessionTimeout:  30 * time.Minute,
		stopCh:          make(chan struct{}),
	}

	// 启动清理协程
	go sm.cleanupLoop()

	return sm
}

// AddSession 添加新会话
func (sm *SessionManager) AddSession(session *PlayerSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 如果已有会话，先清理旧会话
	if oldSession, exists := sm.sessions[session.PlayerID]; exists {
		sm.logger.Warn("replacing existing session",
			zap.String("player_id", session.PlayerID.String()),
			zap.Bool("old_connected", oldSession.IsConnected),
		)
		oldSession.Disconnect()
	}

	sm.sessions[session.PlayerID] = session

	sm.logger.Info("session added",
		zap.String("player_id", session.PlayerID.String()),
		zap.String("username", session.Username),
	)
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(playerID uuid.UUID) (*PlayerSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[playerID]
	return session, exists
}

// RemoveSession 移除会话
func (sm *SessionManager) RemoveSession(playerID uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[playerID]; exists {
		session.Disconnect()
		delete(sm.sessions, playerID)

		sm.logger.Info("session removed",
			zap.String("player_id", playerID.String()),
			zap.String("username", session.Username),
		)
	}
}

// HandleDisconnect 处理玩家断开连接
func (sm *SessionManager) HandleDisconnect(playerID uuid.UUID) error {
	session, exists := sm.GetSession(playerID)
	if !exists {
		return nil
	}

	session.Disconnect()

	// 如果玩家在游戏中，进行必要的清理
	if session.IsSeated && session.CurrentTableID != "" {
		sm.logger.Info("player disconnected while in game",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", session.CurrentTableID),
		)

		// TODO: 通知桌子玩家断线
		// 可以设置玩家为 "disconnected" 状态，给予一定时间重连
		// 如果超时未重连，则自动 fold/sit out
	}

	return nil
}

// BroadcastToTable 向桌子内所有玩家广播消息
func (sm *SessionManager) BroadcastToTable(tableID string, message interface{}) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		if session.GetTableID() == tableID && session.IsConnected {
			session.SendMessage(message)
			count++
		}
	}

	sm.logger.Debug("broadcast to table",
		zap.String("table_id", tableID),
		zap.Int("recipient_count", count),
	)
}

// SendToPlayer 向特定玩家发送消息
func (sm *SessionManager) SendToPlayer(playerID uuid.UUID, message interface{}) bool {
	session, exists := sm.GetSession(playerID)
	if !exists || !session.IsConnected {
		return false
	}

	session.SendMessage(message)
	return true
}

// GetActiveSessions 获取所有活跃会话
func (sm *SessionManager) GetActiveSessions() []*PlayerSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*PlayerSession, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		if session.IsConnected {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetSessionCount 获取会话数量
func (sm *SessionManager) GetSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GetActiveSessionCount 获取活跃会话数量
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		if session.IsConnected {
			count++
		}
	}
	return count
}

// cleanupLoop 定期清理超时的会话
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredSessions()
		case <-sm.stopCh:
			return
		}
	}
}

// cleanupExpiredSessions 清理过期会话
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for playerID, session := range sm.sessions {
		// 如果断线且超时，清理会话
		if !session.IsConnected && now.Sub(session.LastActivityAt) > sm.sessionTimeout {
			sm.logger.Info("cleaning up expired session",
				zap.String("player_id", playerID.String()),
				zap.Duration("inactive_duration", now.Sub(session.LastActivityAt)),
			)

			// 如果有活跃的游戏会话，尝试兑现
			if session.GameSessionID != uuid.Nil {
				go sm.handleAbandonedSession(session)
			}

			delete(sm.sessions, playerID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		sm.logger.Info("cleaned up expired sessions",
			zap.Int("count", expiredCount),
			zap.Int("remaining", len(sm.sessions)),
		)
	}
}

// handleAbandonedSession 处理被遗弃的会话
func (sm *SessionManager) handleAbandonedSession(session *PlayerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取当前游戏会话
	gameSession, err := sm.gameService.GetActiveSession(ctx, session.PlayerID)
	if err != nil {
		sm.logger.Error("failed to get game session for cleanup",
			zap.String("player_id", session.PlayerID.String()),
			zap.Error(err),
		)
		return
	}

	// 自动兑现
	_, err = sm.gameService.CashOut(ctx, service.CashOutRequest{
		PlayerID:  session.PlayerID,
		SessionID: gameSession.ID,
		Chips:     session.GetChips(),
	})

	if err != nil {
		sm.logger.Error("failed to auto cash-out abandoned session",
			zap.String("player_id", session.PlayerID.String()),
			zap.String("session_id", gameSession.ID.String()),
			zap.Int64("chips", session.GetChips()),
			zap.Error(err),
		)
	} else {
		sm.logger.Info("auto cash-out completed for abandoned session",
			zap.String("player_id", session.PlayerID.String()),
			zap.String("session_id", gameSession.ID.String()),
			zap.Int64("chips", session.GetChips()),
		)
	}
}

// Stop 停止会话管理器
func (sm *SessionManager) Stop() {
	close(sm.stopCh)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.logger.Info("stopping session manager",
		zap.Int("active_sessions", len(sm.sessions)),
	)

	// 断开所有连接
	for _, session := range sm.sessions {
		session.Disconnect()
	}

	sm.sessions = make(map[uuid.UUID]*PlayerSession)
}

// GetTablePlayers 获取桌子上的所有玩家
func (sm *SessionManager) GetTablePlayers(tableID string) []*PlayerSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	players := make([]*PlayerSession, 0)
	for _, session := range sm.sessions {
		if session.GetTableID() == tableID {
			players = append(players, session)
		}
	}

	return players
}

// UpdatePlayerChips 更新玩家筹码并同步到数据库
func (sm *SessionManager) UpdatePlayerChips(playerID uuid.UUID, chips int64) error {
	session, exists := sm.GetSession(playerID)
	if !exists {
		return errors.New("session not found")
	}

	session.UpdateChips(chips)

	// 异步更新数据库
	if session.GameSessionID != uuid.Nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := sm.gameService.UpdateSessionChips(ctx, session.GameSessionID, chips)
			if err != nil {
				sm.logger.Error("failed to update session chips in database",
					zap.String("player_id", playerID.String()),
					zap.String("session_id", session.GameSessionID.String()),
					zap.Int64("chips", chips),
					zap.Error(err),
				)
			}
		}()
	}

	return nil
}
