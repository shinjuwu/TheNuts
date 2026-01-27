package ws

import "github.com/shinjuwu/TheNuts/internal/game/domain"

// MapActionType 将前端字符串转换为 Domain Enum
// 这是 Adapter 层的核心职责之一
func MapActionType(action string) domain.ActionType {
	switch action {
	case "FOLD":
		return domain.ActionFold
	case "CHECK":
		return domain.ActionCheck
	case "CALL":
		return domain.ActionCall
	case "BET":
		return domain.ActionBet
	case "RAISE":
		return domain.ActionRaise
	case "ALL_IN":
		return domain.ActionAllIn
	default:
		return domain.ActionFold // 默认或错误处理
	}
}
