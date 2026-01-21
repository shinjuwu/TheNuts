package game

import (
	"sync"

	"github.com/shinjuwu/TheNuts/internal/game/domain"
)

type TableManager struct {
	tables map[string]*domain.Table
	mu     sync.RWMutex
}

func NewTableManager() *TableManager {
	return &TableManager{
		tables: make(map[string]*domain.Table),
	}
}

func (tm *TableManager) GetOrCreateTable(id string) *domain.Table {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t, ok := tm.tables[id]; ok {
		return t
	}

	t := domain.NewTable(id)
	tm.tables[id] = t
	go t.Run()
	return t
}

func (tm *TableManager) GetTable(id string) *domain.Table {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tables[id]
}
