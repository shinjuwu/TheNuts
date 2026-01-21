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
	Pots           *PotManager
	CommunityCards []Card
	DealerPos      int
	CurrentPos     int
	MinBet         int64
	Players        map[string]*Player
	Seats          [9]*Player
	ActionCh       chan PlayerAction
	CloseCh        chan struct{}
}

func NewTable(id string) *Table {
	return &Table{
		ID:       id,
		Pots:     NewPotManager(),
		Players:  make(map[string]*Player),
		ActionCh: make(chan PlayerAction, 100),
		CloseCh:  make(chan struct{}),
		State:    StateIdle,
	}
}

func (t *Table) Run() {
	for {
		select {
		case action := <-t.ActionCh:
			t.handleAction(action)
		case <-t.CloseCh:
			return
		}
	}
}

// handleAction 處理玩家動作
func (t *Table) handleAction(act PlayerAction) {
	// 1. 驗證是否輪到該玩家
	currentSeat := t.Seats[t.CurrentPos]
	if currentSeat == nil || currentSeat.ID != act.PlayerID {
		return // Not your turn
	}

	player := t.Players[act.PlayerID]

	// 2. 處理具體動作
	switch act.Type {
	case ActionFold:
		player.Status = StatusFolded
		player.HasActed = true
	case ActionCheck:
		if player.CurrentBet < t.MinBet {
			return // Cannot check if there is a bet
		}
		player.HasActed = true
	case ActionCall:
		amountToCall := t.MinBet - player.CurrentBet
		if player.Chips < amountToCall {
			// Not enough chips, treat as All-in (simplified)
			amountToCall = player.Chips
			player.Status = StatusAllIn
		}
		player.Chips -= amountToCall
		player.CurrentBet += amountToCall
		player.HasActed = true
	case ActionBet, ActionRaise:
		if act.Amount < t.MinBet {
			return // Invalid bet amount
		}
		diff := act.Amount - player.CurrentBet
		if player.Chips < diff {
			return // Not enough chips
		}

		// 加注發生，重置其他人的 HasActed
		// 注意: 這裡其實只重置 HasActed=false 即可，表示他們需要再次表態
		for _, p := range t.Players {
			if p.ID != player.ID && p.Status != StatusFolded && p.Status != StatusAllIn {
				p.HasActed = false
			}
		}

		player.Chips -= diff
		player.CurrentBet = act.Amount
		t.MinBet = act.Amount
		player.HasActed = true

		if player.Chips == 0 {
			player.Status = StatusAllIn
		}
	}

	// 3. 檢查回合是否結束
	if t.isRoundComplete() {
		t.nextStreet()
	} else {
		t.moveToNextPlayer()
	}
}

// isRoundComplete 判斷本輪下注是否結束
func (t *Table) isRoundComplete() bool {
	activePlayers := 0
	for _, p := range t.Seats {
		if p != nil && p.IsActive() {
			activePlayers++
			if p.Status != StatusAllIn {
				// 必須同時滿足: 1. 注額相等 2. 已表態過
				if p.CurrentBet != t.MinBet || !p.HasActed {
					return false
				}
			}
		}
	}
	// 如果只剩一人或更少，直接結束 (或是大家都 AllIn)
	return true
}

// nextStreet 進入下一階段
func (t *Table) nextStreet() {
	// 1. 收集所有玩家本輪下注額到 PotManager
	//    這會自動處理 Main Pot 和 Side Pots
	bets := make(map[string]int64)
	for _, p := range t.Players {
		if p.CurrentBet > 0 {
			bets[p.ID] = p.CurrentBet
		}
	}
	t.Pots.Accumulate(bets)

	// 2. 重置所有玩家本輪下注額與狀態
	for _, p := range t.Players {
		p.CurrentBet = 0
		p.HasActed = false
	}
	t.MinBet = 0

	switch t.State {
	case StatePreFlop:
		t.State = StateFlop
		// TODO: 發 3 張公牌
	case StateFlop:
		t.State = StateTurn
		// TODO: 發 1 張公牌
	case StateTurn:
		t.State = StateRiver
		// TODO: 發 1 張公牌
	case StateRiver:
		t.State = StateShowdown
		// TODO: 結算
	case StateShowdown:
		t.State = StateIdle
		// TODO: 重置遊戲
	}

	// 進入下一街後，行動權回到 Dealer 後第一位 Active 玩家
	if t.State != StateShowdown && t.State != StateIdle {
		t.CurrentPos = t.DealerPos
		t.moveToNextPlayer()
	}
}

// moveToNextPlayer 移動行動權給下一位可行動玩家
func (t *Table) moveToNextPlayer() {
	for i := 0; i < 9; i++ { // 最多找一圈
		t.CurrentPos = (t.CurrentPos + 1) % 9
		p := t.Seats[t.CurrentPos]
		if p != nil && p.CanAct() {
			return
		}
	}
}
