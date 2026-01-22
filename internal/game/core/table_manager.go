package core

import (
	"context"
	"fmt"
	"sync"
)

// TableManager 管理所有遊戲桌 (通用於所有遊戲類型)
type TableManager struct {
	mu     sync.RWMutex
	tables map[string]GameEngine // key: tableID

	// 遊戲工廠: 根據 GameType 創建對應的遊戲引擎
	factories map[GameType]GameEngineFactory
}

// GameEngineFactory 遊戲引擎工廠介面
type GameEngineFactory interface {
	Create(config GameConfig) (GameEngine, error)
}

func NewTableManager() *TableManager {
	return &TableManager{
		tables:    make(map[string]GameEngine),
		factories: make(map[GameType]GameEngineFactory),
	}
}

// RegisterGameType 註冊遊戲類型 (啟動時調用)
// 例如: tm.RegisterGameType(GameTypePoker, &PokerEngineFactory{})
func (tm *TableManager) RegisterGameType(gameType GameType, factory GameEngineFactory) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.factories[gameType] = factory
}

// CreateTable 創建新桌子
func (tm *TableManager) CreateTable(gameType GameType, config GameConfig) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	factory, ok := tm.factories[gameType]
	if !ok {
		return "", fmt.Errorf("unsupported game type: %s", gameType)
	}

	engine, err := factory.Create(config)
	if err != nil {
		return "", fmt.Errorf("failed to create game engine: %w", err)
	}

	tableID := config.GameID
	tm.tables[tableID] = engine

	// 啟動遊戲循環
	go engine.Start(context.Background())

	return tableID, nil
}

// GetTable 獲取桌子
func (tm *TableManager) GetTable(tableID string) (GameEngine, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	engine, ok := tm.tables[tableID]
	if !ok {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}
	return engine, nil
}

// CloseTable 關閉桌子
func (tm *TableManager) CloseTable(tableID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	engine, ok := tm.tables[tableID]
	if !ok {
		return fmt.Errorf("table not found: %s", tableID)
	}

	if err := engine.Stop(); err != nil {
		return fmt.Errorf("failed to stop engine: %w", err)
	}

	delete(tm.tables, tableID)
	return nil
}

// ListTables 列出所有桌子
func (tm *TableManager) ListTables(gameType *GameType) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tableIDs []string
	for id, engine := range tm.tables {
		if gameType == nil || engine.GetType() == *gameType {
			tableIDs = append(tableIDs, id)
		}
	}
	return tableIDs
}
