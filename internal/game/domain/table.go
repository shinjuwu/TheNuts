package domain

type GameState int

const (
	StateIdle GameState = iota
	StatePreFlop
	StateFlop
	StateTurn
	StateRiver
	StateShowdown
)

type Table struct {
	ID             string
	State          GameState
	Pot            int64
	CommunityCards []int
	DealerPos      int
	CurrentPos     int
}
