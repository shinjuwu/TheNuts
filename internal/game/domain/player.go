package domain

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
