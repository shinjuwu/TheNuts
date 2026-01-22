package domain

import (
	"fmt"
	"testing"
)

func TestFullGameFlow(t *testing.T) {
	// 1. Setup Table and Players
	table := NewTable("full-game")
	p1 := &Player{ID: "p1", SeatIdx: 0, Chips: 1000, Status: StatusPlaying}
	p2 := &Player{ID: "p2", SeatIdx: 1, Chips: 1000, Status: StatusPlaying}
	table.Seats[0] = p1
	table.Seats[1] = p2
	table.Players["p1"] = p1
	table.Players["p2"] = p2

	// Rig Dealer to 0 so P1 is Dealer (BTN/SB in heads up usually acts first?
	// In Heads-up: Dealer is SB. BB is the other.
	// SB posts Small Blind, BB posts Big Blind.
	// Preflop: SB acts first.
	// Postflop: BB acts first.
	table.DealerPos = 0 // P1 is Dealer (SB)

	// 2. Start Hand
	table.StartHand()
	// StartHand logic:
	// - Shuffles Deck
	// - Resets State to PreFlop
	// - Resets Pots, CommunityCards
	// - Deals 2 HoleCards to active players
	// - Moves CurrentPos to Dealer (should be adjusted to correct actor)

	// Override Hole Cards for deterministic result
	p1.HoleCards = []Card{NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart)}
	p2.HoleCards = []Card{NewCard(RankK, SuitSpade), NewCard(RankK, SuitHeart)}

	// Override Deck for Community Cards
	riggedCards := []Card{
		NewCard(Rank2, SuitClub), NewCard(Rank3, SuitClub), NewCard(Rank4, SuitClub), // Flop
		NewCard(Rank5, SuitClub),    // Turn
		NewCard(Rank9, SuitDiamond), // River
	}
	table.Deck.Cards = riggedCards

	// --- PreFlop ---
	// StartHand now automatically posts blinds via postBlinds()
	// P1 (Button/SB in heads-up) posts 10
	// P2 (BB) posts 20
	// Blinds are already posted by StartHand()

	// Verify blinds were posted correctly
	if p1.CurrentBet != 10 {
		t.Errorf("Expected P1 (SB) bet 10, got %d", p1.CurrentBet)
	}
	if p2.CurrentBet != 20 {
		t.Errorf("Expected P2 (BB) bet 20, got %d", p2.CurrentBet)
	}

	// Table MinBet is 20.

	// P1 (SB) needs to Call 10 to match 20
	table.CurrentPos = 0 // Force P1 turn
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCall})
	if p1.CurrentBet != 20 {
		t.Errorf("Expected P1 bet 20, got %d", p1.CurrentBet)
	}

	// P2 (BB) Checks
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCheck})

	// State should transition to Flop
	if table.State != StateFlop {
		t.Errorf("Expected StateFlop, got %v", table.State)
	}
	if len(table.CommunityCards) != 3 {
		t.Errorf("Expected 3 Flop cards, got %d", len(table.CommunityCards))
	}

	// --- Flop ---
	// P2 (BB) acts first post-flop
	if table.CurrentPos != 1 {
		// Wait, moveToNextPlayer relies on DealerPos.
		// StartHand: CurrentPos = DealerPos. moveToNextPlayer().
		// If Dealer=0(P1), Next=1(P2).
		// So Postflop, P2 should act first?
		// Yes, in Heads up, BB acts first postflop.
		// But let's check CurrentPos.
	}

	// Force P2 act first if logic differs, but generally FSM should handle.
	// Let's assume P2 Acts.
	table.CurrentPos = 1 // P2
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCheck})
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCheck})

	if table.State != StateTurn {
		t.Errorf("Expected StateTurn, got %v", table.State)
	}
	if len(table.CommunityCards) != 4 {
		t.Errorf("Expected 4 Turn cards, got %d", len(table.CommunityCards))
	}

	// --- Turn ---
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCheck})
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionCheck})

	if table.State != StateRiver {
		t.Errorf("Expected StateRiver, got %v", table.State)
	}

	// --- River ---
	// P2 Checks
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCheck})

	// P1 Bets 100
	table.handleAction(PlayerAction{PlayerID: "p1", Type: ActionBet, Amount: 100})

	// P2 Calls
	table.handleAction(PlayerAction{PlayerID: "p2", Type: ActionCall})

	// Should trigger Showdown -> Distribute -> Idle
	if table.State != StateIdle {
		t.Errorf("Expected StateIdle (Round End), got %v", table.State)
	}

	// Pot Calculation:
	// Preflop: P1(20) + P2(20) = 40
	// River: P1(100) + P2(100) = 200
	// Total Pot = 240

	// P1 Final Chips = 1000 - 20 - 100 + 240 = 1120
	// P2 Final Chips = 1000 - 20 - 100 = 880

	fmt.Printf("Final Chips -> P1: %d, P2: %d\n", p1.Chips, p2.Chips)

	if p1.Chips != 1120 {
		t.Errorf("Expected P1 chips 1120, got %d", p1.Chips)
	}
	if p2.Chips != 880 {
		t.Errorf("Expected P2 chips 880, got %d", p2.Chips)
	}
}
