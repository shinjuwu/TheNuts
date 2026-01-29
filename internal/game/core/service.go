package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GameService 遊戲服務 - 統一管理所有遊戲類型
type GameService struct {
	tableManager *TableManager

	// 玩家會話管理 (跨遊戲共用)
	sessions map[string]*PlayerSession // key: sessionID
	mu       sync.RWMutex
}

// PlayerSession 玩家會話 (跨遊戲通用)
type PlayerSession struct {
	SessionID     string
	PlayerID      string
	CurrentGameID string // 當前所在遊戲桌ID
	GameType      GameType
	Balance       int64

	// WebSocket 連接或其他通訊通道
	SendCh chan []byte
}

func NewGameService() *GameService {
	return &GameService{
		tableManager: NewTableManager(),
		sessions:     make(map[string]*PlayerSession),
	}
}

// RegisterGameEngine 註冊遊戲引擎 (啟動時調用)
func (s *GameService) RegisterGameEngine(gameType GameType, factory GameEngineFactory) {
	s.tableManager.RegisterGameType(gameType, factory)
}

// CreateGame 創建遊戲桌
func (s *GameService) CreateGame(gameType GameType, config GameConfig) (string, error) {
	return s.tableManager.CreateTable(gameType, config)
}

// JoinGame 玩家加入遊戲
func (s *GameService) JoinGame(sessionID, gameID string, buyIn int64) error {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// 檢查餘額
	if session.Balance < buyIn {
		s.mu.Unlock()
		return fmt.Errorf("insufficient balance")
	}

	// 扣除買入金額
	session.Balance -= buyIn
	session.CurrentGameID = gameID
	s.mu.Unlock()

	// 獲取遊戲引擎
	engine, err := s.tableManager.GetTable(gameID)
	if err != nil {
		// 回滾餘額
		s.mu.Lock()
		session.Balance += buyIn
		session.CurrentGameID = ""
		s.mu.Unlock()
		return err
	}

	// 創建通用玩家對象
	player := &Player{
		ID:        session.PlayerID,
		Balance:   buyIn,
		SessionID: sessionID,
	}

	// 加入遊戲
	if err := engine.AddPlayer(player); err != nil {
		// 回滾餘額
		s.mu.Lock()
		session.Balance += buyIn
		session.CurrentGameID = ""
		s.mu.Unlock()
		return err
	}

	return nil
}

// GetTable 獲取遊戲桌引擎
func (s *GameService) GetTable(gameID string) (GameEngine, error) {
	return s.tableManager.GetTable(gameID)
}

// HandlePlayerAction 處理玩家動作
func (s *GameService) HandlePlayerAction(ctx context.Context, action PlayerAction) (*ActionResult, error) {
	s.mu.RLock()
	session, ok := s.sessions[action.SessionID]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session not found")
	}

	gameID := session.CurrentGameID
	s.mu.RUnlock()

	if gameID == "" {
		return nil, fmt.Errorf("player not in any game")
	}

	engine, err := s.tableManager.GetTable(gameID)
	if err != nil {
		return nil, err
	}

	return engine.HandleAction(ctx, action)
}

// CreateSession 創建玩家會話
func (s *GameService) CreateSession(playerID string, initialBalance int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := generateSessionID()
	s.sessions[sessionID] = &PlayerSession{
		SessionID: sessionID,
		PlayerID:  playerID,
		Balance:   initialBalance,
		SendCh:    make(chan []byte, 256),
	}

	return sessionID
}

// GetSession 獲取會話
func (s *GameService) GetSession(sessionID string) (*PlayerSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// CloseSession 關閉會話
func (s *GameService) CloseSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// 如果玩家在遊戲中，先離開遊戲
	if session.CurrentGameID != "" {
		engine, err := s.tableManager.GetTable(session.CurrentGameID)
		if err == nil {
			engine.RemovePlayer(session.PlayerID)
		}
	}

	close(session.SendCh)
	delete(s.sessions, sessionID)

	return nil
}

var (
	sessionCounter int64
	sessionMu      sync.Mutex
)

// 輔助函數: 生成會話ID
func generateSessionID() string {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	sessionCounter++
	return fmt.Sprintf("session_%d_%d", time.Now().Unix(), sessionCounter)
}
