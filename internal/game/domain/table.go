package domain

import (
	"errors"
	"fmt"
	"time"
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

var ErrPlayerNotFound = errors.New("player not found at table")

// PlayerSitDown 讓指定玩家坐下（SittingOut → Playing）
// NOTE: 此方法直接操作 Players map，與 Table.Run() goroutine 存在潛在資料競爭。
// MVP 階段風險可控，未來應統一透過 ActionCh 序列化。
func (t *Table) PlayerSitDown(playerID string) error {
	player, exists := t.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}
	return player.SitDown()
}

// PlayerStandUp 讓指定玩家站起（→ SittingOut）
// NOTE: 此方法直接操作 Players map，與 Table.Run() goroutine 存在潛在資料競爭。
// MVP 階段風險可控，未來應統一透過 ActionCh 序列化。
func (t *Table) PlayerStandUp(playerID string) (wasInHand bool, err error) {
	player, exists := t.Players[playerID]
	if !exists {
		return false, ErrPlayerNotFound
	}
	return player.StandUp()
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
	// 需要跳過空座位

	// 找小盲位置（Dealer 後第一個有效座位）
	sbPos := t.findNextActiveSeat(t.DealerPos)
	if sbPos == -1 {
		return // 找不到有效座位
	}

	// 找大盲位置（小盲後第一個有效座位）
	bbPos := t.findNextActiveSeat(sbPos)
	if bbPos == -1 {
		return // 找不到有效座位
	}

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

func (t *Table) Run() {
	// 創建定時器，每秒檢查是否可以開始新手牌
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case action := <-t.ActionCh:
			t.handleAction(action)
		case <-ticker.C:
			// 定期檢查是否可以開始新手牌
			t.tryStartNewHand()
		case <-t.CloseCh:
			return
		}
	}
}

// tryStartNewHand 檢查是否可以開始新手牌，如果可以則自動開局
func (t *Table) tryStartNewHand() {
	// 只在 Idle 狀態下嘗試開始新手牌
	if t.State != StateIdle {
		return
	}

	// 統計有多少玩家準備好（StatusPlaying）
	readyPlayers := 0
	for _, p := range t.Seats {
		if p != nil && p.Status == StatusPlaying && p.Chips > 0 {
			readyPlayers++
		}
	}

	// 需要至少 2 個玩家才能開始
	if readyPlayers >= 2 {
		fmt.Printf("Auto-starting new hand with %d players\n", readyPlayers)
		t.StartHand()
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
	case ActionAllIn:
		// All-in: 玩家下注全部籌碼
		if player.Chips == 0 {
			return // Already all-in or no chips
		}

		totalBet := player.CurrentBet + player.Chips
		player.Chips = 0
		player.CurrentBet = totalBet
		player.Status = StatusAllIn
		player.HasActed = true

		// 如果 All-in 金額超過當前最低下注額，重置其他玩家的 HasActed
		if totalBet > t.MinBet {
			t.MinBet = totalBet
			for _, p := range t.Players {
				if p.ID != player.ID && p.Status != StatusFolded && p.Status != StatusAllIn {
					p.HasActed = false
				}
			}
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
	// 0. 檢查是否只剩一個未 Fold 的玩家（提前結束）
	activePlayers := 0
	var lastActivePlayer *Player
	for _, p := range t.Players {
		if p.IsActive() {
			activePlayers++
			lastActivePlayer = p
		}
	}

	if activePlayers == 1 && lastActivePlayer != nil {
		// 只剩一人，直接分配所有底池
		// 先收集當前輪的下注
		bets := make(map[string]int64)
		for _, p := range t.Players {
			if p.CurrentBet > 0 {
				bets[p.ID] = p.CurrentBet
			}
		}
		t.Pots.Accumulate(bets)

		// 分配所有底池給最後一位玩家
		totalPot := t.Pots.Total()
		lastActivePlayer.Chips += totalPot
		fmt.Printf("Player %s wins %d (all others folded)\n",
			lastActivePlayer.ID, totalPot)

		// 結束這手牌
		t.endHand()
		return
	}

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
		// Burn 一張牌
		t.Deck.Draw(1)
		// 發 3 張公牌
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(3)...)
		fmt.Printf("Dealing Flop: %v\n", t.CommunityCards)
	case StateFlop:
		t.State = StateTurn
		// Burn 一張牌
		t.Deck.Draw(1)
		// 發 1 張公牌 (Turn)
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(1)...)
		fmt.Printf("Dealing Turn: %v\n", t.CommunityCards)
	case StateTurn:
		t.State = StateRiver
		// Burn 一張牌
		t.Deck.Draw(1)
		// 發 1 張公牌 (River)
		t.CommunityCards = append(t.CommunityCards, t.Deck.Draw(1)...)
		fmt.Printf("Dealing River: %v\n", t.CommunityCards)
	case StateRiver:
		t.State = StateShowdown
		t.Showdown()
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

// findNextActiveSeat 從指定位置開始找下一個有活躍玩家的座位
// 返回座位索引，如果找不到返回 -1
func (t *Table) findNextActiveSeat(startPos int) int {
	for i := 1; i <= 9; i++ { // 從下一位開始找，最多找一圈
		pos := (startPos + i) % 9
		if p := t.Seats[pos]; p != nil && p.IsActive() {
			return pos
		}
	}
	return -1
}

// Showdown 執行攤牌邏輯並分配底池
func (t *Table) Showdown() {
	fmt.Printf("=== Showdown ===\n")

	// 使用 Distribute 函數計算 payouts
	payouts := Distribute(t.Pots.Pots, t.Players, t.CommunityCards)

	// 將 payouts 加到玩家籌碼
	for playerID, amount := range payouts {
		if player, exists := t.Players[playerID]; exists {
			player.Chips += amount
			fmt.Printf("Player %s wins %d chips (final: %d)\n",
				playerID, amount, player.Chips)
		}
	}

	// 手牌結束，執行清理
	t.endHand()
}

// endHand 結束當前手牌並準備下一手
func (t *Table) endHand() {
	fmt.Printf("=== Hand Complete ===\n")

	// 移動 Dealer Button
	t.rotateDealerButton()

	// 重置玩家狀態
	t.resetPlayersForNextHand()

	// 設置狀態為 Idle，準備下一手牌
	t.State = StateIdle
}

// rotateDealerButton 將 Dealer 位置移到下一個有效座位
// 有效座位的條件：座位有人、有籌碼、未暫離（不考慮當前手牌狀態如 Folded）
func (t *Table) rotateDealerButton() {
	for i := 1; i <= 9; i++ {
		pos := (t.DealerPos + i) % 9
		// 檢查座位是否有人、有籌碼且未暫離
		// 我們不使用 IsActive()，因為我們要包含 StatusFolded/StatusAllIn 的玩家
		// 下一手牌開始時，這些狀態會被重置為 StatusPlaying
		if p := t.Seats[pos]; p != nil && p.Chips > 0 && p.Status != StatusSittingOut {
			t.DealerPos = pos
			fmt.Printf("Dealer button moved to seat %d\n", t.DealerPos)
			return
		}
	}
	// 如果找不到有效座位，保持當前位置
	fmt.Printf("No valid seat found for dealer rotation, keeping at seat %d\n", t.DealerPos)
}

// resetPlayersForNextHand 重置所有玩家狀態以準備下一手牌
func (t *Table) resetPlayersForNextHand() {
	for _, p := range t.Players {
		// 清除手牌
		p.HoleCards = nil
		p.CurrentBet = 0
		p.HasActed = false

		// 重置狀態：Folded/AllIn → Playing (如果不是 SittingOut)
		if p.Status == StatusFolded || p.Status == StatusAllIn {
			if p.Chips > 0 {
				p.Status = StatusPlaying
			} else {
				// 籌碼為 0，自動站起
				p.Status = StatusSittingOut
				fmt.Printf("Player %s has no chips, sitting out\n", p.ID)
			}
		}
	}
}
