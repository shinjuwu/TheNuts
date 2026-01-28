package domain

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid player status transition")
	ErrCannotStandUpWhileAllIn = errors.New("cannot stand up while all-in")
)

type PlayerStatus int

const (
	StatusSittingOut PlayerStatus = iota
	StatusPlaying
	StatusFolded
	StatusAllIn
)

type Player struct {
	ID         string
	SeatIdx    int
	Chips      int64
	CurrentBet int64
	Status     PlayerStatus
	HoleCards  []Card
	HasActed   bool
}

// IsActive 回傳玩家是否仍在遊戲中 (非 Fold 且 非 SittingOut)
func (p *Player) IsActive() bool {
	return p.Status != StatusFolded && p.Status != StatusSittingOut
}

// CanAct 回傳玩家是否可以進行動作 (非 AllIn, 非 Fold, 非 SittingOut)
func (p *Player) CanAct() bool {
	return p.IsActive() && p.Status != StatusAllIn
}

// SitDown 將玩家從 SittingOut 狀態轉為 Playing 狀態
func (p *Player) SitDown() error {
	if p.Status != StatusSittingOut {
		return ErrInvalidStatusTransition
	}
	p.Status = StatusPlaying
	return nil
}

// StandUp 將玩家轉為 SittingOut 狀態
// 回傳 wasInHand 表示玩家是否在手牌中（Playing 狀態會自動 Fold）
func (p *Player) StandUp() (wasInHand bool, err error) {
	switch p.Status {
	case StatusSittingOut:
		// 冪等：已經是 SittingOut，無需操作
		return false, nil
	case StatusPlaying:
		// 正在遊戲中，自動 Fold 並站起
		p.HoleCards = nil
		p.CurrentBet = 0
		p.HasActed = true
		p.Status = StatusSittingOut
		return true, nil
	case StatusFolded:
		// 已經 Fold，直接站起
		p.Status = StatusSittingOut
		return false, nil
	case StatusAllIn:
		return false, ErrCannotStandUpWhileAllIn
	default:
		return false, ErrInvalidStatusTransition
	}
}
