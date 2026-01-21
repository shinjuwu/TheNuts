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
	ActionCh       chan interface{} // 接收玩家動作的通道
	CloseCh        chan struct{}
}

func NewTable(id string) *Table {
	return &Table{
		ID:       id,
		ActionCh: make(chan interface{}, 100),
		CloseCh:  make(chan struct{}),
	}
}

func (t *Table) Run() {
	for {
		select {
		case action := <-t.ActionCh:
			// TODO: 處理玩家動作
			_ = action
		case <-t.CloseCh:
			return
		}
	}
}
