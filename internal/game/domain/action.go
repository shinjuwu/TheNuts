package domain

// ActionType 代表玩家的動作類型
type ActionType int

const (
	ActionFold  ActionType = iota // 棄牌
	ActionCheck                   // 過牌
	ActionCall                    // 跟注
	ActionBet                     // 下注
	ActionRaise                   // 加注
	ActionAllIn                   // 全押
)

// PlayerAction 是核心邏輯使用的標準動作結構 (Royal Language)
// 它不依賴任何外部標籤 (如 json tag)
type PlayerAction struct {
	PlayerID string
	Type     ActionType
	Amount   int64 // 使用 int64 避免金額溢出
}
