package domain

// ActionType 代表玩家的動作類型
type ActionType int

const (
	// 遊戲動作
	ActionFold  ActionType = iota // 棄牌
	ActionCheck                   // 過牌
	ActionCall                    // 跟注
	ActionBet                     // 下注
	ActionRaise                   // 加注
	ActionAllIn                   // 全押

	// 桌面管理命令
	ActionJoinTable  // 加入桌子
	ActionLeaveTable // 離開桌子
	ActionSitDown    // 坐下
	ActionStandUp    // 站起
	ActionDisconnect // 玩家斷線
	ActionReconnect  // 玩家重連
)

// String 回傳動作類型的字串表示
func (a ActionType) String() string {
	switch a {
	case ActionFold:
		return "FOLD"
	case ActionCheck:
		return "CHECK"
	case ActionCall:
		return "CALL"
	case ActionBet:
		return "BET"
	case ActionRaise:
		return "RAISE"
	case ActionAllIn:
		return "ALL_IN"
	default:
		return "UNKNOWN"
	}
}

// PlayerAction 是核心邏輯使用的標準動作結構 (Royal Language)
// 它不依賴任何外部標籤 (如 json tag)
type PlayerAction struct {
	PlayerID string
	Type     ActionType
	Amount   int64 // 使用 int64 避免金額溢出

	// JoinTable 專用欄位
	Player  *Player // 要加入的玩家（僅 ActionJoinTable 使用）
	SeatIdx int     // 目標座位（僅 ActionJoinTable 使用）

	// 同步回應通道（nil 表示 fire-and-forget）
	ResultCh chan<- ActionResult
}

// ActionResult 命令執行結果
type ActionResult struct {
	Err       error
	WasInHand bool // StandUp 回傳用
}
