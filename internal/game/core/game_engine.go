package core

import (
	"context"
	"time"
)

// GameEngine 是所有遊戲引擎必須實現的介面
// 不論是德州撲克、百家樂、老虎機都要遵循這個契約
type GameEngine interface {
	// GetType 返回遊戲類型 (poker, slot, baccarat...)
	GetType() GameType

	// Initialize 初始化遊戲實例
	Initialize(config GameConfig) error

	// Start 啟動遊戲循環
	Start(ctx context.Context) error

	// Stop 停止遊戲並清理資源
	Stop() error

	// HandleAction 處理玩家動作 (下注、發牌、搖骰等)
	HandleAction(ctx context.Context, action PlayerAction) (*ActionResult, error)

	// GetState 獲取當前遊戲狀態 (用於斷線重連)
	GetState() GameState

	// AddPlayer 玩家加入遊戲
	AddPlayer(player *Player) error

	// RemovePlayer 玩家離開遊戲
	RemovePlayer(playerID string) error

	// BroadcastEvent 廣播事件給所有觀眾/玩家
	BroadcastEvent(event GameEvent)

	// GetEventChannel 獲取事件通道 (供服務層監聽)
	GetEventChannel() <-chan GameEvent
}

// GameType 遊戲類型枚舉
type GameType string

const (
	GameTypePoker     GameType = "poker"     // 德州撲克
	GameTypeSlot      GameType = "slot"      // 電子老虎機
	GameTypeBaccarat  GameType = "baccarat"  // 百家樂
	GameTypeBlackjack GameType = "blackjack" // 21點
	GameTypeRoulette  GameType = "roulette"  // 輪盤
)

// GameConfig 遊戲配置 (每種遊戲有自己的擴展)
type GameConfig struct {
	GameID      string
	MaxPlayers  int
	MinBet      int64
	MaxBet      int64
	RakePercent float64 // 抽水比例
	Timeout     time.Duration
	CustomData  map[string]interface{} // 遊戲專屬配置
}

// PlayerAction 玩家動作 (通用結構)
type PlayerAction struct {
	PlayerID  string
	SessionID string
	GameID    string
	Type      ActionType
	Amount    int64
	Data      map[string]interface{} // 遊戲特定的擴展數據
	Timestamp time.Time
}

// ActionType 動作類型
type ActionType string

const (
	// 通用動作
	ActionJoin  ActionType = "join"
	ActionLeave ActionType = "leave"
	ActionBet   ActionType = "bet"
	ActionFold  ActionType = "fold"
	ActionCheck ActionType = "check"
	ActionCall  ActionType = "call"
	ActionRaise ActionType = "raise"
	ActionAllIn ActionType = "all_in"

	// 電子遊戲專用
	ActionSpin ActionType = "spin"
	ActionStop ActionType = "stop"

	// 視訊遊戲專用
	ActionDeal  ActionType = "deal"
	ActionHit   ActionType = "hit"
	ActionStand ActionType = "stand"
	ActionSplit ActionType = "split"
)

// ActionResult 動作執行結果
type ActionResult struct {
	Success   bool
	NewState  GameState
	Events    []GameEvent // 觸發的事件列表
	ErrorCode string
	Message   string
}

// GameState 遊戲狀態 (抽象介面)
type GameState interface {
	GetID() string
	GetPhase() string
	Serialize() ([]byte, error)
	GetPlayers() []*Player
}

// GameEvent 遊戲事件 (用於廣播)
type GameEvent struct {
	EventType EventType
	GameID    string
	Timestamp time.Time
	Data      interface{}
	// 視角過濾: 不同玩家收到的數據可能不同
	TargetPlayerID string // 空字符串表示廣播給所有人
}

// EventType 事件類型
type EventType string

const (
	EventGameStart    EventType = "game_start"
	EventGameEnd      EventType = "game_end"
	EventPlayerJoin   EventType = "player_join"
	EventPlayerLeave  EventType = "player_leave"
	EventStateChange  EventType = "state_change"
	EventBetPlaced    EventType = "bet_placed"
	EventHandComplete EventType = "hand_complete"
)

// Player 玩家通用結構
type Player struct {
	ID        string
	Name      string
	Balance   int64
	SessionID string
	JoinTime  time.Time
	// 遊戲特定數據由各遊戲引擎自行擴展
	GameData map[string]interface{}
}
