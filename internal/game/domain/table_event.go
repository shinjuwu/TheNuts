package domain

// TableEventType 遊戲事件類型
type TableEventType string

const (
	EventHandStart      TableEventType = "HAND_START"
	EventHoleCards      TableEventType = "HOLE_CARDS"
	EventBlindsPosted   TableEventType = "BLINDS_POSTED"
	EventYourTurn       TableEventType = "YOUR_TURN"
	EventPlayerAction   TableEventType = "PLAYER_ACTION"
	EventCommunityCards TableEventType = "COMMUNITY_CARDS"
	EventShowdownResult TableEventType = "SHOWDOWN_RESULT"
	EventWinByFold      TableEventType = "WIN_BY_FOLD"
	EventHandEnd        TableEventType = "HAND_END"
)

// TableEvent 遊戲事件，由 Table 發射，上層回調轉發到 WebSocket
type TableEvent struct {
	Type           TableEventType
	TableID        string
	Data           map[string]interface{}
	TargetPlayerID string // 空字串 = 廣播給桌上所有人；非空 = 定向發送
}
