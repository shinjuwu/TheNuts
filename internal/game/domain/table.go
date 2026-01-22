package domain

import (
	"fmt"
)

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
	Deck           *Deck
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
		Deck:     NewDeck(),
		Players:  make(map[string]*Player),
		ActionCh: make(chan PlayerAction, 100),
		CloseCh:  make(chan struct{}),
		State:    StateIdle,
	}
}

// StartHand 開始新的一手牌
func (t *Table) StartHand() {
	// 1. 洗牌
	t.Deck = NewDeck()
	t.Deck.Shuffle()

	// 2. 重置狀態
	t.CommunityCards = make([]Card, 0)
	t.Pots = NewPotManager()
	t.State = StatePreFlop
	t.MinBet = 20 // 假設大盲為 20

	// 3. 發手牌 (每人 2 張)
	// 從 Dealer 下一位開始發? 通常是小盲先拿?
	// 簡化: 遍歷所有 Active 玩家發牌
	for _, p := range t.Seats {
		if p != nil && p.IsActive() {
			p.HoleCards = t.Deck.Draw(2)
			p.Status = StatusPlaying
			p.CurrentBet = 0
			p.HasActed = false
		}
	}

	// 4. 設定盲注 (Blind)
	t.postBlinds()

	// 5. 設定行動權
	// Preflop 由 BB 後一位 (UTG) 開始。若是 3 人桌: BTN, SB, BB -> BTN Action
	t.CurrentPos = t.DealerPos // 之後會 Call moveToNextPlayer 調整到正確位置
	t.moveToNextPlayer()
}

// postBlinds 收取小盲和大盲注
func (t *Table) postBlinds() {
	// 計算活躍玩家數量
	activePlayers := 0
	for _, p := range t.Seats {
		if p != nil && p.IsActive() {
			activePlayers++
		}
	}

	// 至少需要 2 個玩家才能下盲注
	if activePlayers < 2 {
		return
	}

	// 盲注金額（可以從配置讀取，這裡暫時硬編碼）
	smallBlindAmount := t.MinBet / 2 // 假設 MinBet 是大盲金額
	bigBlindAmount := t.MinBet

	// 確保至少有基本盲注
	if smallBlindAmount == 0 {
		smallBlindAmount = 10
		bigBlindAmount = 20
		t.MinBet = bigBlindAmount
	}

	// Heads-up (兩人對決) 時的特殊規則:
	// - Button (莊家) 是小盲
	// - 另一位是大盲
	if activePlayers == 2 {
		t.postBlindHeadsUp(smallBlindAmount, bigBlindAmount)
		return
	}

	// 3人以上的標準情況:
	// - DealerPos + 1 = 小盲 (Small Blind)
	// - DealerPos + 2 = 大盲 (Big Blind)
	sbPos := (t.DealerPos + 1) % 9
	bbPos := (t.DealerPos + 2) % 9

	// 收取小盲
	if sb := t.Seats[sbPos]; sb != nil && sb.IsActive() {
		amount := min(smallBlindAmount, sb.Chips)
		sb.Chips -= amount
		sb.CurrentBet = amount

		// 如果下注後籌碼為 0，標記為 All-in
		if sb.Chips == 0 {
			sb.Status = StatusAllIn
		}

		fmt.Printf("Player %s posts small blind: %d (remaining: %d)\n",
			sb.ID, amount, sb.Chips)
	}

	// 收取大盲
	if bb := t.Seats[bbPos]; bb != nil && bb.IsActive() {
		amount := min(bigBlindAmount, bb.Chips)
		bb.Chips -= amount
		bb.CurrentBet = amount

		// 如果下注後籌碼為 0，標記為 All-in
		if bb.Chips == 0 {
			bb.Status = StatusAllIn
		}

		fmt.Printf("Player %s posts big blind: %d (remaining: %d)\n",
			bb.ID, amount, bb.Chips)
	}
}

// postBlindHeadsUp 處理兩人單挑時的盲注（規則特殊）
func (t *Table) postBlindHeadsUp(smallBlindAmount, bigBlindAmount int64) {
	// Heads-up 時:
	// - Button (莊家) 先行動且是小盲
	// - 非 Button 是大盲

	var buttonPlayer, otherPlayer *Player

	// 找到 Button 玩家和另一位玩家
	for i := 0; i < 9; i++ {
		if p := t.Seats[i]; p != nil && p.IsActive() {
			if i == t.DealerPos {
				buttonPlayer = p
			} else {
				otherPlayer = p
			}
		}
	}

	// 收取小盲 (Button)
	if buttonPlayer != nil {
		amount := min(smallBlindAmount, buttonPlayer.Chips)
		buttonPlayer.Chips -= amount
		buttonPlayer.CurrentBet = amount

		if buttonPlayer.Chips == 0 {
			buttonPlayer.Status = StatusAllIn
		}

		fmt.Printf("Player %s (Button) posts small blind: %d (remaining: %d)\n",
			buttonPlayer.ID, amount, buttonPlayer.Chips)
	}

	// 收取大盲 (另一位玩家)
	if otherPlayer != nil {
		amount := min(bigBlindAmount, otherPlayer.Chips)
		otherPlayer.Chips -= amount
		otherPlayer.CurrentBet = amount

		if otherPlayer.Chips == 0 {
			otherPlayer.Status = StatusAllIn
		}

		fmt.Printf("Player %s posts big blind: %d (remaining: %d)\n",
			otherPlayer.ID, amount, otherPlayer.Chips)
	}
}

// min 返回兩個 int64 中的較小值
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
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
		// 發 3 張公牌
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(3)...)
		fmt.Printf("Dealing Flop: %v\n", t.CommunityCards)
	case StateFlop:
		t.State = StateTurn
		// 發 1 張公牌 (Turn)
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(1)...)
		fmt.Printf("Dealing Turn: %v\n", t.CommunityCards)
	case StateTurn:
		t.State = StateRiver
		// 發 1 張公牌 (River)
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(1)...)
		fmt.Printf("Dealing River: %v\n", t.CommunityCards)
	case StateRiver:
		t.State = StateShowdown
		// 結算
		fmt.Println("Showdown!")
		payouts := Distribute(t.Pots.Pots, t.Players, t.CommunityCards)

		// 分配籌碼
		for pid, amount := range payouts {
			if p, ok := t.Players[pid]; ok {
				p.Chips += amount
				fmt.Printf("Player %s wins %d\n", pid, amount)
			}
		}

		// 本局結束，重置或進入 Idle
		// 這裡簡單切回 Idle，實際應用可能會有 StateHandEnd 等待時間
		t.State = StateIdle

		// TODO: 自動開始下一局? 還是等待外部指令?
		// 為了測試方便，這裡暫時不自動 Loop
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
