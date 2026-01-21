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
	HoleCards  []int
}
