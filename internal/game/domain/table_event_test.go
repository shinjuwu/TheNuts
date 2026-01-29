package domain

import (
	"sync"
	"testing"
)

// eventCollector 收集 Table 發射的事件，用於測試驗證
type eventCollector struct {
	mu     sync.Mutex
	events []TableEvent
}

func newEventCollector() *eventCollector {
	return &eventCollector{events: make([]TableEvent, 0)}
}

func (ec *eventCollector) handler(event TableEvent) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.events = append(ec.events, event)
}

func (ec *eventCollector) getEvents() []TableEvent {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	copied := make([]TableEvent, len(ec.events))
	copy(copied, ec.events)
	return copied
}

func (ec *eventCollector) findByType(t TableEventType) []TableEvent {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	var result []TableEvent
	for _, e := range ec.events {
		if e.Type == t {
			result = append(result, e)
		}
	}
	return result
}

// setupThreePlayerTable 建立一個 3 人桌供測試使用
func setupThreePlayerTable() (*Table, *Player, *Player, *Player) {
	table := NewTable("event-test")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}
	p3 := &Player{ID: "p3", SeatIdx: 2, Chips: 1000, Status: StatusPlaying, HoleCards: []Card{}}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Seats[2] = p3
	table.Players["p1"] = p1
	table.Players["p2"] = p2
	table.Players["p3"] = p3

	return table, p1, p2, p3
}

// TestEventEmission_HandStart 開始手牌應發射 HAND_START + HOLE_CARDS + BLINDS_POSTED + YOUR_TURN
func TestEventEmission_HandStart(t *testing.T) {
	table, _, _, _ := setupThreePlayerTable()
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	table.DealerPos = 0
	table.StartHand()

	events := ec.getEvents()
	if len(events) == 0 {
		t.Fatal("Expected events to be emitted during StartHand")
	}

	// 應有 HAND_START
	handStarts := ec.findByType(EventHandStart)
	if len(handStarts) != 1 {
		t.Errorf("Expected 1 HAND_START event, got %d", len(handStarts))
	} else {
		if handStarts[0].TableID != "event-test" {
			t.Errorf("Expected TableID 'event-test', got '%s'", handStarts[0].TableID)
		}
		if handStarts[0].TargetPlayerID != "" {
			t.Error("HAND_START should be broadcast (empty TargetPlayerID)")
		}
		players, ok := handStarts[0].Data["players"].([]map[string]interface{})
		if !ok {
			t.Error("Expected players list in HAND_START data")
		} else if len(players) != 3 {
			t.Errorf("Expected 3 players in HAND_START, got %d", len(players))
		}
	}

	// 應有 3 個 HOLE_CARDS（每人一個定向事件）
	holeCards := ec.findByType(EventHoleCards)
	if len(holeCards) != 3 {
		t.Errorf("Expected 3 HOLE_CARDS events, got %d", len(holeCards))
	}
	for _, hc := range holeCards {
		if hc.TargetPlayerID == "" {
			t.Error("HOLE_CARDS should be targeted (non-empty TargetPlayerID)")
		}
		cards, ok := hc.Data["cards"].([]string)
		if !ok {
			t.Error("Expected cards list in HOLE_CARDS data")
		} else if len(cards) != 2 {
			t.Errorf("Expected 2 hole cards, got %d", len(cards))
		}
	}

	// 應有 BLINDS_POSTED
	blinds := ec.findByType(EventBlindsPosted)
	if len(blinds) != 1 {
		t.Errorf("Expected 1 BLINDS_POSTED event, got %d", len(blinds))
	} else {
		if blinds[0].TargetPlayerID != "" {
			t.Error("BLINDS_POSTED should be broadcast")
		}
	}

	// 應有 YOUR_TURN（給第一個行動者）
	yourTurns := ec.findByType(EventYourTurn)
	if len(yourTurns) != 1 {
		t.Errorf("Expected 1 YOUR_TURN event, got %d", len(yourTurns))
	} else {
		if yourTurns[0].TargetPlayerID == "" {
			t.Error("YOUR_TURN should be targeted")
		}
	}
}

// TestEventEmission_PlayerAction 執行動作應發射 PLAYER_ACTION + YOUR_TURN
func TestEventEmission_PlayerAction(t *testing.T) {
	table, _, _, _ := setupThreePlayerTable()

	// 手動設置 PreFlop 狀態
	table.State = StatePreFlop
	table.DealerPos = 0
	table.CurrentPos = 0
	table.MinBet = 20

	// 設定盲注
	table.Players["p2"].CurrentBet = 10
	table.Players["p2"].Chips -= 10
	table.Players["p3"].CurrentBet = 20
	table.Players["p3"].Chips -= 20

	// 註冊事件收集器
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	// p1 Call
	if err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall}); err != nil {
		t.Fatalf("Expected no error for valid call, got %v", err)
	}

	events := ec.getEvents()

	// 應有 PLAYER_ACTION
	actions := ec.findByType(EventPlayerAction)
	if len(actions) != 1 {
		t.Errorf("Expected 1 PLAYER_ACTION event, got %d", len(actions))
	} else {
		if actions[0].TargetPlayerID != "" {
			t.Error("PLAYER_ACTION should be broadcast")
		}
		if actions[0].Data["player_id"] != "p1" {
			t.Errorf("Expected player_id 'p1', got '%v'", actions[0].Data["player_id"])
		}
		if actions[0].Data["action"] != "CALL" {
			t.Errorf("Expected action 'CALL', got '%v'", actions[0].Data["action"])
		}
	}

	// 應有 YOUR_TURN（給下一位行動者 p2）
	yourTurns := ec.findByType(EventYourTurn)
	if len(yourTurns) != 1 {
		t.Errorf("Expected 1 YOUR_TURN event, got %d", len(yourTurns))
	} else {
		if yourTurns[0].TargetPlayerID != "p2" {
			t.Errorf("Expected YOUR_TURN for p2, got '%s'", yourTurns[0].TargetPlayerID)
		}
	}

	_ = events
}

// TestEventEmission_CommunityCards 推進到 Flop 應發射 COMMUNITY_CARDS
func TestEventEmission_CommunityCards(t *testing.T) {
	table, _, _, _ := setupThreePlayerTable()

	// 設置 PreFlop 且所有人已 call
	table.State = StatePreFlop
	table.DealerPos = 0
	table.CurrentPos = 0
	table.MinBet = 20
	table.Deck = NewDeck()
	table.Deck.Shuffle()

	// 模擬所有人下注相同且已表態
	for _, p := range table.Players {
		p.CurrentBet = 20
		p.Chips -= 20
		p.HasActed = true
	}

	// 註冊事件收集器
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	// 觸發 nextStreet（應進入 Flop）
	table.nextStreet()

	if table.State != StateFlop {
		t.Fatalf("Expected StateFlop, got %v", table.State)
	}

	// 應有 COMMUNITY_CARDS 事件
	ccEvents := ec.findByType(EventCommunityCards)
	if len(ccEvents) != 1 {
		t.Errorf("Expected 1 COMMUNITY_CARDS event, got %d", len(ccEvents))
	} else {
		if ccEvents[0].TargetPlayerID != "" {
			t.Error("COMMUNITY_CARDS should be broadcast")
		}
		street, _ := ccEvents[0].Data["street"].(string)
		if street != "FLOP" {
			t.Errorf("Expected street 'FLOP', got '%s'", street)
		}
		newCards, _ := ccEvents[0].Data["new_cards"].([]string)
		if len(newCards) != 3 {
			t.Errorf("Expected 3 new cards for FLOP, got %d", len(newCards))
		}
		allCards, _ := ccEvents[0].Data["community_cards"].([]string)
		if len(allCards) != 3 {
			t.Errorf("Expected 3 community cards after FLOP, got %d", len(allCards))
		}
	}
}

// TestEventEmission_Showdown 完整打到攤牌，驗證 SHOWDOWN_RESULT + HAND_END
func TestEventEmission_Showdown(t *testing.T) {
	table := NewTable("showdown-event-test")

	// 建立 2 個玩家
	p1 := &Player{
		ID:      "p1",
		SeatIdx: 0,
		Chips:   980,
		Status:  StatusPlaying,
		HoleCards: []Card{
			NewCard(RankA, SuitSpade),
			NewCard(RankK, SuitSpade),
		},
	}
	p2 := &Player{
		ID:      "p2",
		SeatIdx: 1,
		Chips:   980,
		Status:  StatusPlaying,
		HoleCards: []Card{
			NewCard(Rank2, SuitClub),
			NewCard(Rank3, SuitClub),
		},
	}

	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	// 設置公牌（River 階段）
	table.CommunityCards = []Card{
		NewCard(RankA, SuitHeart),
		NewCard(RankK, SuitHeart),
		NewCard(RankQ, SuitDiamond),
		NewCard(RankJ, SuitDiamond),
		NewCard(RankT, SuitClub),
	}

	// 設置底池
	table.Pots.Accumulate(map[string]int64{"p1": 20, "p2": 20})
	table.State = StateShowdown

	// 註冊事件收集器
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	// 執行 Showdown
	table.Showdown()

	// 應有 SHOWDOWN_RESULT
	showdowns := ec.findByType(EventShowdownResult)
	if len(showdowns) != 1 {
		t.Errorf("Expected 1 SHOWDOWN_RESULT event, got %d", len(showdowns))
	} else {
		if showdowns[0].TargetPlayerID != "" {
			t.Error("SHOWDOWN_RESULT should be broadcast")
		}
		winners, ok := showdowns[0].Data["winners"].([]map[string]interface{})
		if !ok {
			t.Error("Expected winners list in SHOWDOWN_RESULT data")
		} else if len(winners) == 0 {
			t.Error("Expected at least 1 winner")
		}
	}

	// 應有 HAND_END
	handEnds := ec.findByType(EventHandEnd)
	if len(handEnds) != 1 {
		t.Errorf("Expected 1 HAND_END event, got %d", len(handEnds))
	} else {
		if handEnds[0].TargetPlayerID != "" {
			t.Error("HAND_END should be broadcast")
		}
		players, ok := handEnds[0].Data["players"].([]map[string]interface{})
		if !ok {
			t.Error("Expected players list in HAND_END data")
		} else if len(players) == 0 {
			t.Error("Expected at least 1 player in HAND_END")
		}
	}
}

// TestEventEmission_AllFoldWin 所有人 Fold 後的勝利也應發射 HAND_END
func TestEventEmission_AllFoldWin(t *testing.T) {
	table, _, _, _ := setupThreePlayerTable()

	table.State = StateFlop
	table.DealerPos = 0
	table.CurrentPos = 0
	table.MinBet = 0
	table.CommunityCards = []Card{
		NewCard(RankA, SuitSpade),
		NewCard(RankK, SuitSpade),
		NewCard(RankQ, SuitSpade),
	}

	// Flop 階段，所有人下注為 0，p1 先行動
	for _, p := range table.Players {
		p.CurrentBet = 0
		p.HasActed = false
	}

	// 底池已收集
	table.Pots.Accumulate(map[string]int64{"p1": 20, "p2": 20, "p3": 20})

	// 註冊事件收集器
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	// p1 Bet 50
	if err := table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionBet, Amount: 50}); err != nil {
		t.Fatalf("Expected no error for valid bet, got %v", err)
	}
	// p2 Fold
	if err := table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionFold}); err != nil {
		t.Fatalf("Expected no error for valid fold, got %v", err)
	}
	// p3 Fold
	if err := table.handleAction(PlayerAction{PlayerID: "p3", Type: ActionFold}); err != nil {
		t.Fatalf("Expected no error for valid fold, got %v", err)
	}

	// 所有人 Fold，手牌結束，p1 贏得底池
	if table.State != StateIdle {
		t.Errorf("Expected StateIdle after all fold, got %v", table.State)
	}

	// 應有 WIN_BY_FOLD
	wins := ec.findByType(EventWinByFold)
	if len(wins) != 1 {
		t.Errorf("Expected 1 WIN_BY_FOLD event, got %d", len(wins))
	} else {
		if wins[0].TargetPlayerID != "" {
			t.Error("WIN_BY_FOLD should be broadcast")
		}
		if wins[0].Data["player_id"] != "p1" {
			t.Errorf("Expected winner p1, got '%v'", wins[0].Data["player_id"])
		}
		amount, ok := wins[0].Data["amount"].(int64)
		if !ok {
			t.Error("Expected amount in WIN_BY_FOLD data")
		} else if amount != 110 {
			// 底池 60（先前累計）+ 本輪 p1 下注 50 = 110
			t.Errorf("Expected win amount 110, got %d", amount)
		}
	}

	// 應有 HAND_END
	handEnds := ec.findByType(EventHandEnd)
	if len(handEnds) != 1 {
		t.Errorf("Expected 1 HAND_END event, got %d", len(handEnds))
	}
}

// TestFireEvent_SetsTableID 驗證 fireEvent 自動填入 TableID
func TestFireEvent_SetsTableID(t *testing.T) {
	table := NewTable("my-table-id")
	ec := newEventCollector()
	table.AddOnEvent(ec.handler)

	table.fireEvent(TableEvent{
		Type: EventHandStart,
		Data: map[string]interface{}{},
	})

	events := ec.getEvents()
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}
	if events[0].TableID != "my-table-id" {
		t.Errorf("Expected TableID 'my-table-id', got '%s'", events[0].TableID)
	}
}
