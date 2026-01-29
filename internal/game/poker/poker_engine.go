package poker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shinjuwu/TheNuts/internal/game/core"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
)

// PokerEngine 德州撲克引擎 (實現 core.GameEngine 介面)
type PokerEngine struct {
	mu     sync.RWMutex
	config core.GameConfig
	table  *domain.Table // 復用現有的 Table 邏輯
	ctx    context.Context
	cancel context.CancelFunc

	// 事件廣播通道
	eventCh chan core.GameEvent
}

// PokerEngineFactory 德州撲克引擎工廠
type PokerEngineFactory struct{}

func (f *PokerEngineFactory) Create(config core.GameConfig) (core.GameEngine, error) {
	engine := &PokerEngine{
		config:  config,
		table:   domain.NewTable(config.GameID),
		eventCh: make(chan core.GameEvent, 100),
	}
	engine.table.OnHandComplete = engine.onHandComplete
	return engine, nil
}

// GetType 實現 GameEngine 介面
func (e *PokerEngine) GetType() core.GameType {
	return core.GameTypePoker
}

// Initialize 實現 GameEngine 介面
func (e *PokerEngine) Initialize(config core.GameConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config = config
	e.table = domain.NewTable(config.GameID)
	// Hook up the OnHandComplete callback
	e.table.OnHandComplete = e.onHandComplete

	// 從 CustomData 提取德撲專屬配置
	if blinds, ok := config.CustomData["blinds"].(int64); ok {
		e.table.MinBet = blinds
	}

	return nil
}

// GetEventChannel 實現 GameEngine 介面
func (e *PokerEngine) GetEventChannel() <-chan core.GameEvent {
	return e.eventCh
}

// onHandComplete 手牌結束回調
func (e *PokerEngine) onHandComplete(table *domain.Table) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 收集玩家籌碼
	playerChips := make(map[string]int64)
	for id, player := range table.Players {
		playerChips[id] = player.Chips
	}

	e.BroadcastEvent(core.GameEvent{
		EventType: core.EventHandComplete,
		GameID:    table.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"player_chips": playerChips,
		},
	})
}

// Start 實現 GameEngine 介面 - 啟動遊戲循環
func (e *PokerEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.mu.Unlock()

	// 啟動底層 Table 的 Run 循環
	go e.table.Run()

	// 監聽上下文取消信號
	<-e.ctx.Done()
	return nil
}

// Stop 實現 GameEngine 介面
func (e *PokerEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}

	close(e.table.CloseCh)
	close(e.eventCh)

	return nil
}

// HandleAction 實現 GameEngine 介面 - 統一動作入口
func (e *PokerEngine) HandleAction(ctx context.Context, action core.PlayerAction) (*core.ActionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 將通用 Action 轉換為德撲專用的 PlayerAction
	domainAction := e.convertAction(action)

	// 發送到 Table 的 ActionCh (這是你現有的邏輯)
	select {
	case e.table.ActionCh <- domainAction:
		// 成功發送
		return &core.ActionResult{
			Success: true,
			Message: "action queued",
		}, nil
	case <-ctx.Done():
		return &core.ActionResult{
			Success:   false,
			ErrorCode: "timeout",
			Message:   "action timeout",
		}, fmt.Errorf("action timeout")
	default:
		return &core.ActionResult{
			Success:   false,
			ErrorCode: "queue_full",
			Message:   "action queue is full",
		}, fmt.Errorf("action queue full")
	}
}

// convertAction 將通用 Action 轉為 domain.PlayerAction
func (e *PokerEngine) convertAction(action core.PlayerAction) domain.PlayerAction {
	// 映射 ActionType
	var actionType domain.ActionType
	switch action.Type {
	case core.ActionFold:
		actionType = domain.ActionFold
	case core.ActionCheck:
		actionType = domain.ActionCheck
	case core.ActionCall:
		actionType = domain.ActionCall
	case core.ActionBet, core.ActionRaise:
		actionType = domain.ActionRaise
	case core.ActionAllIn:
		actionType = domain.ActionRaise // All-in 在 domain 層會自動處理
	default:
		actionType = domain.ActionFold // 預設
	}

	return domain.PlayerAction{
		PlayerID: action.PlayerID,
		Type:     actionType,
		Amount:   action.Amount,
	}
}

// GetState 實現 GameEngine 介面 - 獲取當前狀態
func (e *PokerEngine) GetState() core.GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &PokerGameState{
		table: e.table,
	}
}

// AddPlayer 實現 GameEngine 介面
func (e *PokerEngine) AddPlayer(player *core.Player) error {
	// 找空座位（讀取 Seats 仍需同步）
	// 透過 ActionCh 發送，避免直接存取 table 資料
	seatIdx := -1
	e.mu.RLock()
	for i, seat := range e.table.Seats {
		if seat == nil {
			seatIdx = i
			break
		}
	}
	e.mu.RUnlock()

	if seatIdx == -1 {
		return fmt.Errorf("table is full")
	}

	// 創建 domain.Player
	domainPlayer := &domain.Player{
		ID:         player.ID,
		Chips:      player.Balance,
		SeatIdx:    seatIdx,
		Status:     domain.StatusSittingOut,
		HoleCards:  []domain.Card{},
		CurrentBet: 0,
		HasActed:   false,
	}

	// 透過 ActionCh 發送，由 Table.Run() 統一處理
	resultCh := make(chan domain.ActionResult, 1)
	e.table.ActionCh <- domain.PlayerAction{
		Type:     domain.ActionJoinTable,
		Player:   domainPlayer,
		SeatIdx:  seatIdx,
		ResultCh: resultCh,
	}
	result := <-resultCh

	if result.Err != nil {
		return result.Err
	}

	// 廣播玩家加入事件
	e.BroadcastEvent(core.GameEvent{
		EventType: core.EventPlayerJoin,
		GameID:    e.table.ID,
		Data: map[string]interface{}{
			"player_id": player.ID,
			"seat_idx":  seatIdx,
		},
	})

	return nil
}

// RemovePlayer 實現 GameEngine 介面
func (e *PokerEngine) RemovePlayer(playerID string) error {
	// 透過 ActionCh 發送，由 Table.Run() 統一處理
	resultCh := make(chan domain.ActionResult, 1)
	e.table.ActionCh <- domain.PlayerAction{
		Type:     domain.ActionLeaveTable,
		PlayerID: playerID,
		ResultCh: resultCh,
	}
	result := <-resultCh

	if result.Err != nil {
		return result.Err
	}

	// 廣播玩家離開事件
	e.BroadcastEvent(core.GameEvent{
		EventType: core.EventPlayerLeave,
		GameID:    e.table.ID,
		Data: map[string]interface{}{
			"player_id": playerID,
		},
	})

	return nil
}

// BroadcastEvent 實現 GameEngine 介面
func (e *PokerEngine) BroadcastEvent(event core.GameEvent) {
	select {
	case e.eventCh <- event:
	default:
		// 事件通道滿了，丟棄事件 (或記錄日誌)
	}
}

// PokerGameState 德撲遊戲狀態 (實現 core.GameState)
type PokerGameState struct {
	table *domain.Table
}

func (s *PokerGameState) GetID() string {
	return s.table.ID
}

func (s *PokerGameState) GetPhase() string {
	switch s.table.State {
	case domain.StateIdle:
		return "idle"
	case domain.StatePreFlop:
		return "preflop"
	case domain.StateFlop:
		return "flop"
	case domain.StateTurn:
		return "turn"
	case domain.StateRiver:
		return "river"
	case domain.StateShowdown:
		return "showdown"
	default:
		return "unknown"
	}
}

func (s *PokerGameState) Serialize() ([]byte, error) {
	// TODO: 實現序列化邏輯 (用於斷線重連)
	return nil, nil
}

func (s *PokerGameState) GetPlayers() []*core.Player {
	var players []*core.Player
	for _, p := range s.table.Players {
		players = append(players, &core.Player{
			ID:      p.ID,
			Balance: p.Chips,
		})
	}
	return players
}
