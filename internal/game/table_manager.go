package game

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
	"github.com/shinjuwu/TheNuts/internal/game/service"
)

type TableManager struct {
	tables      map[string]*domain.Table
	mu          sync.RWMutex
	gameService *service.GameService
}

func NewTableManager(gs *service.GameService) *TableManager {
	return &TableManager{
		tables:      make(map[string]*domain.Table),
		gameService: gs,
	}
}

func (tm *TableManager) GetOrCreateTable(id string) *domain.Table {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t, ok := tm.tables[id]; ok {
		return t
	}

	t := domain.NewTable(id)
	t.OnHandComplete = tm.onHandComplete
	tm.tables[id] = t
	go t.Run()
	return t
}

func (tm *TableManager) GetTable(id string) *domain.Table {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tables[id]
}

func (tm *TableManager) onHandComplete(t *domain.Table) {
	ctx := context.Background()
	// 遍歷所有玩家同步籌碼
	for playerIDStr, player := range t.Players {
		playerID, err := uuid.Parse(playerIDStr)
		if err != nil {
			fmt.Printf("Failed to parse player ID %s: %v\n", playerIDStr, err)
			continue
		}

		// 獲取活躍會話
		session, err := tm.gameService.GetActiveSession(ctx, playerID)
		if err != nil {
			// 玩家可能已經登出或沒有會話，這是預期內的（例如掉線）
			// 但如果有 session 卻找不到，或者是其他錯誤，值得記錄
			continue
		}

		// 更新籌碼
		if err := tm.gameService.UpdateSessionChips(ctx, session.ID, player.Chips); err != nil {
			fmt.Printf("Failed to update chips for player %s: %v\n", playerIDStr, err)
		} else {
			// fmt.Printf("Synced chips for player %s: %d\n", playerIDStr, player.Chips)
		}
	}

	// 如果是最後一手牌或桌子需要關閉，這裡也可以處理
}
