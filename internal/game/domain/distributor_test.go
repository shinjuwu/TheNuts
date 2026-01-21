package domain

import (
	"testing"
)

func TestDistribute_SimpleWinner(t *testing.T) {
	// P1 (Winner), P2 (Loser)
	p1 := &Player{ID: "p1", Status: StatusAllIn, HoleCards: []Card{NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade)}}
	p2 := &Player{ID: "p2", Status: StatusAllIn, HoleCards: []Card{NewCard(Rank2, SuitHeart), NewCard(Rank3, SuitDiamond)}}
	players := map[string]*Player{"p1": p1, "p2": p2}

	// Board: A-K-Q-J-10 (Royal Flush for P1)
	board := []Card{
		NewCard(RankQ, SuitSpade), NewCard(RankJ, SuitSpade), NewCard(RankT, SuitSpade),
		NewCard(Rank2, SuitClub), NewCard(Rank3, SuitClub),
	}

	// Pot: 200
	pot := NewPot()
	pot.Amount = 200
	pot.Contributors = map[string]bool{"p1": true, "p2": true}

	results := Distribute([]*Pot{pot}, players, board)

	if results["p1"] != 200 {
		t.Errorf("Expected p1 to win 200, got %d", results["p1"])
	}
	if results["p2"] != 0 {
		t.Errorf("Expected p2 to win 0, got %d", results["p2"])
	}
}

func TestDistribute_SplitPot(t *testing.T) {
	// P1, P2 have same hand (Board plays)
	p1 := &Player{ID: "p1", Status: StatusAllIn, HoleCards: []Card{NewCard(Rank2, SuitHeart), NewCard(Rank3, SuitDiamond)}}
	p2 := &Player{ID: "p2", Status: StatusAllIn, HoleCards: []Card{NewCard(Rank2, SuitClub), NewCard(Rank3, SuitSpade)}}
	players := map[string]*Player{"p1": p1, "p2": p2}

	// Board: A-A-K-K-Q (Two Pair on board, kickers match)
	board := []Card{
		NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart),
		NewCard(RankK, SuitClub), NewCard(RankK, SuitDiamond),
		NewCard(RankQ, SuitSpade),
	}

	pot := NewPot()
	pot.Amount = 300 // Odd amount if 300/2 = 150
	pot.Contributors = map[string]bool{"p1": true, "p2": true}

	results := Distribute([]*Pot{pot}, players, board)

	if results["p1"] != 150 {
		t.Errorf("Expected p1 to win 150, got %d", results["p1"])
	}
	if results["p2"] != 150 {
		t.Errorf("Expected p2 to win 150, got %d", results["p2"])
	}
}

func TestDistribute_SidePot(t *testing.T) {
	// P1 (Short stack, All-in) - Royal Flush
	// P2 (Medium stack, All-in) - Straight Flush
	// P3 (Big stack, All-in) - Four of a Kind

	// P1 wins Main Pot
	// P2 wins Side Pot (vs P3)

	p1 := &Player{ID: "p1", Status: StatusAllIn, HoleCards: []Card{NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade)}}
	p2 := &Player{ID: "p2", Status: StatusAllIn, HoleCards: []Card{NewCard(Rank9, SuitHeart), NewCard(Rank8, SuitHeart)}}
	p3 := &Player{ID: "p3", Status: StatusAllIn, HoleCards: []Card{NewCard(Rank2, SuitClub), NewCard(Rank2, SuitDiamond)}}

	players := map[string]*Player{"p1": p1, "p2": p2, "p3": p3}

	// Board: Qs Js Ts 7h 6h (Matches P1 Royal, P2 Straight Flush)
	board := []Card{
		NewCard(RankQ, SuitSpade), NewCard(RankJ, SuitSpade), NewCard(RankT, SuitSpade), // Royal for P1
		NewCard(Rank7, SuitHeart), NewCard(Rank6, SuitHeart), // Straight Flush for P2 (9h 8h 7h 6h + Th?? No)
		// Wait, P2: 9h 8h. Board: 7h 6h. Need 5h or Th.
		// Let's adjust board to ensure P2 has Straight Flush.
		// Board: Qs Js Ts 7h 6h
		// P2: 9h 8h -> 9 8 7 6 ... need 5 or T.
		// Let's make board: Qs Js Ts Th 5h
		// P1: As Ks Qs Js Ts -> Royal Flush
		// P2: 9h 8h 7h 6h 5h (Straight Flush). Need 7h 6h 5h on board or hand.
	}

	// Let's simplify hands.
	// P1: AA (AAs Ad)
	// P2: KK (KKs Kd)
	// P3: QQ (QQs Qd)
	// Board: 2c 3c 4c 5c 9d (No flush/straight danger)
	p1.HoleCards = []Card{NewCard(RankA, SuitSpade), NewCard(RankA, SuitDiamond)}
	p2.HoleCards = []Card{NewCard(RankK, SuitSpade), NewCard(RankK, SuitDiamond)}
	p3.HoleCards = []Card{NewCard(RankQ, SuitSpade), NewCard(RankQ, SuitDiamond)}

	board = []Card{
		NewCard(Rank2, SuitClub), NewCard(Rank3, SuitClub), NewCard(Rank4, SuitClub),
		NewCard(Rank5, SuitClub), NewCard(Rank9, SuitDiamond),
	}

	// Pot 1 (Main): 300 (100 from all). P1, P2, P3. Expect P1 win.
	pot1 := NewPot()
	pot1.Amount = 300
	pot1.Contributors = map[string]bool{"p1": true, "p2": true, "p3": true}

	// Pot 2 (Side): 200 (100 from P2, P3). P2, P3. Expect P2 win (KK > QQ).
	pot2 := NewPot()
	pot2.Amount = 200
	pot2.Contributors = map[string]bool{"p2": true, "p3": true}

	results := Distribute([]*Pot{pot1, pot2}, players, board)

	if results["p1"] != 300 {
		t.Errorf("Expected p1 to win Main Pot (300), got %d", results["p1"])
	}
	if results["p2"] != 200 {
		t.Errorf("Expected p2 to win Side Pot (200), got %d", results["p2"])
	}
	if results["p3"] != 0 {
		t.Errorf("Expected p3 to win 0, got %d", results["p3"])
	}
}
